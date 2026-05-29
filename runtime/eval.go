package runtime

import (
	"fmt"
	"nova/ast"
	"strings"
)

type Interpreter struct {
	env *Environment
}

func NewInterpreter() *Interpreter {
	env := NewEnvironment()
	RegisterBuiltins(env)
	registerModules(env)
	return &Interpreter{env: env}
}

func (interp *Interpreter) Run(prog *ast.Program) error {
	_, err := interp.execStmts(prog.Statements, interp.env)
	return err
}

// ── Statement execution ──────────────────────────────────────

type controlFlow struct {
	kind  string // "return" | "break" | "continue"
	value *Value
}

func (interp *Interpreter) execStmts(stmts []ast.Statement, env *Environment) (*Value, error) {
	for _, stmt := range stmts {
		val, err := interp.execStmt(stmt, env)
		if err != nil {
			return nil, err
		}
		if val != nil {
			return val, nil // propagate control flow
		}
	}
	return nil, nil
}

func (interp *Interpreter) execStmt(stmt ast.Statement, env *Environment) (*Value, error) {
	switch s := stmt.(type) {
	case *ast.AssignStatement:
		val, err := interp.evalExpr(s.Value, env)
		if err != nil {
			return nil, err
		}
		env.Update(s.Name, val)
		return nil, nil

	case *ast.ExpressionStatement:
		_, err := interp.evalExpr(s.Expression, env)
		return nil, err

	case *ast.FunctionDeclaration:
		fn := &Function{Name: s.Name, Params: s.Params, Body: s.Body, Env: env}
		env.Set(s.Name, &Value{Type: TypeFunction, FuncVal: fn})
		return nil, nil

	case *ast.ReturnStatement:
		val, err := interp.evalExpr(s.Value, env)
		if err != nil {
			return nil, err
		}
		return &Value{Type: TypeString, StrVal: "__return__", ListVal: []*Value{val}}, nil // carrier

	case *ast.IfStatement:
		return interp.execIf(s, env)

	case *ast.ForStatement:
		return interp.execFor(s, env)

	case *ast.WhileStatement:
		return interp.execWhile(s, env)

	case *ast.BreakStatement:
		return &Value{Type: TypeString, StrVal: "__break__"}, nil

	case *ast.ContinueStatement:
		return &Value{Type: TypeString, StrVal: "__continue__"}, nil

	case *ast.TryCatchStatement:
		return interp.execTryCatch(s, env)

	case *ast.TaskStatement:
		// Synchronous execution of task (async is Phase 3)
		_, err := interp.evalExpr(s.Call, env)
		return nil, err

	case *ast.WaitStatement:
		return nil, nil // no-op in sync mode

	case *ast.ImportStatement:
		return interp.execImport(s, env)

	case nil:
		return nil, nil
	}
	return nil, fmt.Errorf("unknown statement type %T", stmt)
}

func isControl(v *Value, kind string) bool {
	return v != nil && v.Type == TypeString && v.StrVal == kind
}

func (interp *Interpreter) execIf(s *ast.IfStatement, env *Environment) (*Value, error) {
	cond, err := interp.evalExpr(s.Condition, env)
	if err != nil {
		return nil, err
	}
	if cond.IsTruthy() {
		return interp.execStmts(s.Then, NewEnclosed(env))
	} else if len(s.Else) > 0 {
		return interp.execStmts(s.Else, NewEnclosed(env))
	}
	return nil, nil
}

func (interp *Interpreter) execFor(s *ast.ForStatement, env *Environment) (*Value, error) {
	iter, err := interp.evalExpr(s.Iterable, env)
	if err != nil {
		return nil, err
	}
	var items []*Value
	switch iter.Type {
	case TypeList:
		items = iter.ListVal
	case TypeRange:
		for i := iter.RangeStart; i <= iter.RangeEnd; i++ {
			items = append(items, NumberVal(float64(i)))
		}
	case TypeString:
		for _, ch := range iter.StrVal {
			items = append(items, StringVal(string(ch)))
		}
	case TypeMap:
		for _, k := range iter.MapOrder {
			items = append(items, StringVal(k))
		}
	default:
		return nil, fmt.Errorf("line %d: cannot iterate over %T", s.Line, iter)
	}
	for _, item := range items {
		loopEnv := NewEnclosed(env)
		loopEnv.Set(s.Variable, item)
		sig, err := interp.execStmts(s.Body, loopEnv)
		if err != nil {
			return nil, err
		}
		if isControl(sig, "__break__") {
			break
		}
		if isControl(sig, "__continue__") {
			continue
		}
		if isControl(sig, "__return__") {
			return sig, nil
		}
	}
	return nil, nil
}

func (interp *Interpreter) execWhile(s *ast.WhileStatement, env *Environment) (*Value, error) {
	for {
		cond, err := interp.evalExpr(s.Condition, env)
		if err != nil {
			return nil, err
		}
		if !cond.IsTruthy() {
			break
		}
		loopEnv := NewEnclosed(env)
		sig, err := interp.execStmts(s.Body, loopEnv)
		if err != nil {
			return nil, err
		}
		if isControl(sig, "__break__") {
			break
		}
		if isControl(sig, "__continue__") {
			continue
		}
		if isControl(sig, "__return__") {
			return sig, nil
		}
	}
	return nil, nil
}

func (interp *Interpreter) execTryCatch(s *ast.TryCatchStatement, env *Environment) (*Value, error) {
	_, err := interp.execStmts(s.Try, NewEnclosed(env))
	if err != nil {
		catchEnv := NewEnclosed(env)
		catchEnv.Set(s.ErrVar, StringVal(err.Error()))
		return interp.execStmts(s.Catch, catchEnv)
	}
	return nil, nil
}

func (interp *Interpreter) execImport(s *ast.ImportStatement, env *Environment) (*Value, error) {
	mod, ok := env.Get(s.Module)
	if !ok || mod.Type != TypeModule {
		return nil, fmt.Errorf("line %d: unknown module %q", s.Line, s.Module)
	}
	if len(s.Names) == 0 {
		env.Set(s.Module, mod)
	} else {
		for _, name := range s.Names {
			val, ok := mod.ModuleVal[name]
			if !ok {
				return nil, fmt.Errorf("line %d: %q has no export %q", s.Line, s.Module, name)
			}
			env.Set(name, val)
		}
	}
	return nil, nil
}

// ── Expression evaluation ────────────────────────────────────

func (interp *Interpreter) evalExpr(expr ast.Expression, env *Environment) (*Value, error) {
	switch e := expr.(type) {
	case *ast.NullLiteral:
		return Null, nil
	case *ast.BoolLiteral:
		return BoolVal(e.Value), nil
	case *ast.NumberLiteral:
		return NumberVal(e.Value), nil
	case *ast.StringLiteral:
		return StringVal(interpolate(e.Value, env, interp)), nil
	case *ast.Identifier:
		val, ok := env.Get(e.Value)
		if !ok {
			return nil, fmt.Errorf("line %d: undefined variable %q", e.Line, e.Value)
		}
		return val, nil
	case *ast.ListLiteral:
		return interp.evalList(e, env)
	case *ast.MapLiteral:
		return interp.evalMap(e, env)
	case *ast.RangeLiteral:
		return interp.evalRange(e, env)
	case *ast.BinaryExpression:
		return interp.evalBinary(e, env)
	case *ast.UnaryExpression:
		return interp.evalUnary(e, env)
	case *ast.CallExpression:
		return interp.evalCall(e, env)
	case *ast.MemberExpression:
		return interp.evalMember(e, env)
	case *ast.IndexExpression:
		return interp.evalIndex(e, env)
	}
	return nil, fmt.Errorf("unknown expression type %T", expr)
}

func (interp *Interpreter) evalList(e *ast.ListLiteral, env *Environment) (*Value, error) {
	elems := make([]*Value, len(e.Elements))
	for i, el := range e.Elements {
		v, err := interp.evalExpr(el, env)
		if err != nil {
			return nil, err
		}
		elems[i] = v
	}
	return &Value{Type: TypeList, ListVal: elems}, nil
}

func (interp *Interpreter) evalMap(e *ast.MapLiteral, env *Environment) (*Value, error) {
	m := &Value{Type: TypeMap, MapVal: make(map[string]*Value)}
	for _, k := range e.Order {
		keyVal, err := interp.evalExpr(k, env)
		if err != nil {
			return nil, err
		}
		valVal, err := interp.evalExpr(e.Pairs[k], env)
		if err != nil {
			return nil, err
		}
		keyStr := keyVal.String()
		m.MapVal[keyStr] = valVal
		m.MapOrder = append(m.MapOrder, keyStr)
	}
	return m, nil
}

func (interp *Interpreter) evalRange(e *ast.RangeLiteral, env *Environment) (*Value, error) {
	start, err := interp.evalExpr(e.Start, env)
	if err != nil {
		return nil, err
	}
	end, err := interp.evalExpr(e.End, env)
	if err != nil {
		return nil, err
	}
	if start.Type != TypeNumber || end.Type != TypeNumber {
		return nil, fmt.Errorf("line %d: range requires numbers", e.Line)
	}
	return &Value{Type: TypeRange, RangeStart: int(start.NumVal), RangeEnd: int(end.NumVal)}, nil
}

func (interp *Interpreter) evalBinary(e *ast.BinaryExpression, env *Environment) (*Value, error) {
	// Short-circuit for && and ||
	if e.Operator == "&&" {
		left, err := interp.evalExpr(e.Left, env)
		if err != nil {
			return nil, err
		}
		if !left.IsTruthy() {
			return False, nil
		}
		right, err := interp.evalExpr(e.Right, env)
		if err != nil {
			return nil, err
		}
		return BoolVal(right.IsTruthy()), nil
	}
	if e.Operator == "||" {
		left, err := interp.evalExpr(e.Left, env)
		if err != nil {
			return nil, err
		}
		if left.IsTruthy() {
			return True, nil
		}
		right, err := interp.evalExpr(e.Right, env)
		if err != nil {
			return nil, err
		}
		return BoolVal(right.IsTruthy()), nil
	}

	left, err := interp.evalExpr(e.Left, env)
	if err != nil {
		return nil, err
	}
	right, err := interp.evalExpr(e.Right, env)
	if err != nil {
		return nil, err
	}

	switch e.Operator {
	case "+":
		if left.Type == TypeString || right.Type == TypeString {
			return StringVal(left.String() + right.String()), nil
		}
		if left.Type == TypeList && right.Type == TypeList {
			combined := append(append([]*Value{}, left.ListVal...), right.ListVal...)
			return &Value{Type: TypeList, ListVal: combined}, nil
		}
		return NumberVal(left.NumVal + right.NumVal), nil
	case "-":
		return NumberVal(left.NumVal - right.NumVal), nil
	case "*":
		return NumberVal(left.NumVal * right.NumVal), nil
	case "/":
		if right.NumVal == 0 {
			return nil, fmt.Errorf("line %d: division by zero", e.Line)
		}
		return NumberVal(left.NumVal / right.NumVal), nil
	case "%":
		return NumberVal(float64(int64(left.NumVal) % int64(right.NumVal))), nil
	case "==":
		return BoolVal(valEqual(left, right)), nil
	case "!=":
		return BoolVal(!valEqual(left, right)), nil
	case ">":
		return BoolVal(left.NumVal > right.NumVal), nil
	case "<":
		return BoolVal(left.NumVal < right.NumVal), nil
	case ">=":
		return BoolVal(left.NumVal >= right.NumVal), nil
	case "<=":
		return BoolVal(left.NumVal <= right.NumVal), nil
	}
	return nil, fmt.Errorf("line %d: unknown operator %q", e.Line, e.Operator)
}

func valEqual(a, b *Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case TypeNull:
		return true
	case TypeBool:
		return a.BoolVal == b.BoolVal
	case TypeNumber:
		return a.NumVal == b.NumVal
	case TypeString:
		return a.StrVal == b.StrVal
	}
	return false
}

func (interp *Interpreter) evalUnary(e *ast.UnaryExpression, env *Environment) (*Value, error) {
	val, err := interp.evalExpr(e.Operand, env)
	if err != nil {
		return nil, err
	}
	switch e.Operator {
	case "!":
		return BoolVal(!val.IsTruthy()), nil
	case "-":
		if val.Type != TypeNumber {
			return nil, fmt.Errorf("line %d: unary - requires a number", e.Line)
		}
		return NumberVal(-val.NumVal), nil
	}
	return nil, fmt.Errorf("unknown unary operator %q", e.Operator)
}

func (interp *Interpreter) evalCall(e *ast.CallExpression, env *Environment) (*Value, error) {
	callee, err := interp.evalExpr(e.Callee, env)
	if err != nil {
		return nil, err
	}
	args := make([]*Value, len(e.Arguments))
	for i, arg := range e.Arguments {
		v, err := interp.evalExpr(arg, env)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}

	switch callee.Type {
	case TypeBuiltin:
		return callee.BuiltinVal(args)
	case TypeFunction:
		return interp.callFunction(callee.FuncVal, args, e.Line)
	}
	return nil, fmt.Errorf("line %d: %s is not callable", e.Line, callee.String())
}

func (interp *Interpreter) callFunction(fn *Function, args []*Value, line int) (*Value, error) {
	fnEnv := NewEnclosed(fn.Env)
	stmts := fn.Body.([]ast.Statement)
	for i, param := range fn.Params {
		if i < len(args) {
			fnEnv.Set(param, args[i])
		} else {
			fnEnv.Set(param, Null)
		}
	}
	sig, err := interp.execStmts(stmts, fnEnv)
	if err != nil {
		return nil, err
	}
	if isControl(sig, "__return__") {
		return sig.ListVal[0], nil
	}
	return Null, nil
}

func (interp *Interpreter) evalMember(e *ast.MemberExpression, env *Environment) (*Value, error) {
	obj, err := interp.evalExpr(e.Object, env)
	if err != nil {
		return nil, err
	}
	if obj.Type == TypeNull {
		if e.Safe {
			return Null, nil
		}
		return nil, fmt.Errorf("line %d: cannot access .%s on null", e.Line, e.Property)
	}
	switch obj.Type {
	case TypeMap:
		val, ok := obj.MapVal[e.Property]
		if !ok {
			if e.Safe {
				return Null, nil
			}
			return Null, nil // maps return null for missing keys
		}
		return val, nil
	case TypeModule:
		val, ok := obj.ModuleVal[e.Property]
		if !ok {
			return nil, fmt.Errorf("line %d: module has no member %q", e.Line, e.Property)
		}
		return val, nil
	case TypeString:
		// String method dispatch
		return interp.stringMethod(obj, e.Property, e.Line)
	case TypeList:
		return interp.listMethod(obj, e.Property, e.Line)
	}
	return nil, fmt.Errorf("line %d: cannot access .%s on %s", e.Line, e.Property, typeName(obj))
}

func (interp *Interpreter) stringMethod(s *Value, method string, line int) (*Value, error) {
	switch method {
	case "length", "len":
		return NumberVal(float64(len([]rune(s.StrVal)))), nil
	case "upper":
		return StringVal(strings.ToUpper(s.StrVal)), nil
	case "lower":
		return StringVal(strings.ToLower(s.StrVal)), nil
	case "trim":
		return StringVal(strings.TrimSpace(s.StrVal)), nil
	case "split":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			sep := ""
			if len(args) > 0 {
				sep = args[0].StrVal
			}
			parts := strings.Split(s.StrVal, sep)
			elems := make([]*Value, len(parts))
			for i, p := range parts {
				elems[i] = StringVal(p)
			}
			return &Value{Type: TypeList, ListVal: elems}, nil
		}}, nil
	case "contains":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("contains() expects 1 argument")
			}
			return BoolVal(strings.Contains(s.StrVal, args[0].StrVal)), nil
		}}, nil
	case "replace":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("replace() expects 2 arguments")
			}
			return StringVal(strings.ReplaceAll(s.StrVal, args[0].StrVal, args[1].StrVal)), nil
		}}, nil
	case "startsWith":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("startsWith() expects 1 argument")
			}
			return BoolVal(strings.HasPrefix(s.StrVal, args[0].StrVal)), nil
		}}, nil
	case "endsWith":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("endsWith() expects 1 argument")
			}
			return BoolVal(strings.HasSuffix(s.StrVal, args[0].StrVal)), nil
		}}, nil
	}
	return nil, fmt.Errorf("line %d: string has no method %q", line, method)
}

func (interp *Interpreter) listMethod(l *Value, method string, line int) (*Value, error) {
	switch method {
	case "length", "len":
		return NumberVal(float64(len(l.ListVal))), nil
	case "append", "push":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			l.ListVal = append(l.ListVal, args...)
			return l, nil
		}}, nil
	case "pop":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(l.ListVal) == 0 {
				return Null, nil
			}
			n := len(l.ListVal)
			val := l.ListVal[n-1]
			l.ListVal = l.ListVal[:n-1]
			return val, nil
		}}, nil
	case "contains":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("contains() expects 1 argument")
			}
			for _, el := range l.ListVal {
				if valEqual(el, args[0]) {
					return True, nil
				}
			}
			return False, nil
		}}, nil
	case "join":
		return &Value{Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			sep := ""
			if len(args) > 0 {
				sep = args[0].StrVal
			}
			parts := make([]string, len(l.ListVal))
			for i, v := range l.ListVal {
				parts[i] = v.String()
			}
			return StringVal(strings.Join(parts, sep)), nil
		}}, nil
	}
	return nil, fmt.Errorf("line %d: list has no method %q", line, method)
}

func (interp *Interpreter) evalIndex(e *ast.IndexExpression, env *Environment) (*Value, error) {
	left, err := interp.evalExpr(e.Left, env)
	if err != nil {
		return nil, err
	}
	idx, err := interp.evalExpr(e.Index, env)
	if err != nil {
		return nil, err
	}
	switch left.Type {
	case TypeList:
		if idx.Type != TypeNumber {
			return nil, fmt.Errorf("line %d: list index must be a number", e.Line)
		}
		i := int(idx.NumVal)
		if i < 0 {
			i = len(left.ListVal) + i
		}
		if i < 0 || i >= len(left.ListVal) {
			return nil, fmt.Errorf("line %d: list index %d out of range", e.Line, int(idx.NumVal))
		}
		return left.ListVal[i], nil
	case TypeMap:
		val, ok := left.MapVal[idx.String()]
		if !ok {
			return Null, nil
		}
		return val, nil
	case TypeString:
		if idx.Type != TypeNumber {
			return nil, fmt.Errorf("line %d: string index must be a number", e.Line)
		}
		runes := []rune(left.StrVal)
		i := int(idx.NumVal)
		if i < 0 {
			i = len(runes) + i
		}
		if i < 0 || i >= len(runes) {
			return nil, fmt.Errorf("line %d: string index %d out of range", e.Line, int(idx.NumVal))
		}
		return StringVal(string(runes[i])), nil
	}
	return nil, fmt.Errorf("line %d: cannot index %s", e.Line, typeName(left))
}

// interpolate handles "{expr}" inside strings
func interpolate(s string, env *Environment, interp *Interpreter) string {
	var result strings.Builder
	i := 0
	runes := []rune(s)
	for i < len(runes) {
		if runes[i] == '{' && i+1 < len(runes) && runes[i+1] != '{' {
			// find closing }
			j := i + 1
			depth := 1
			for j < len(runes) && depth > 0 {
				if runes[j] == '{' {
					depth++
				} else if runes[j] == '}' {
					depth--
				}
				j++
			}
			expr := string(runes[i+1 : j-1])
			val := evalInterpolated(expr, env, interp)
			result.WriteString(val)
			i = j
		} else {
			result.WriteRune(runes[i])
			i++
		}
	}
	return result.String()
}

func evalInterpolated(expr string, env *Environment, interp *Interpreter) string {
	// Quick lookup: if it's just a variable name
	val, ok := env.Get(strings.TrimSpace(expr))
	if ok {
		return val.String()
	}
	return "{" + expr + "}"
}

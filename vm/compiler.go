package vm

import (
	"fmt"
	"nova/ast"
	"nova/runtime"
	"strings"
)

// loopCtx tracks break/continue patch-points for a single loop.
type loopCtx struct {
	startIP      int   // where OP_LOOP should jump back to (after iter setup)
	breakPatches []int // offsets whose u16 needs patching to loop-exit
}

// Compiler walks an AST and emits bytecode into a FuncProto.
type Compiler struct {
	proto  *FuncProto
	chunk  *Chunk   // shortcut to proto.Chunk
	loops  []loopCtx
	errors []string
}

func newCompiler(name string, params []string) *Compiler {
	proto := &FuncProto{
		Name:   name,
		Params: params,
		Chunk:  NewChunk(),
	}
	return &Compiler{proto: proto, chunk: proto.Chunk}
}

// CompileProgram compiles a top-level program into a FuncProto with no params.
func CompileProgram(prog *ast.Program) (*FuncProto, error) {
	c := newCompiler("<main>", nil)
	for _, stmt := range prog.Statements {
		if err := c.compileStmt(stmt); err != nil {
			return nil, err
		}
	}
	c.chunk.Emit(OP_NULL, 0)
	c.chunk.Emit(OP_RETURN, 0)
	return c.proto, nil
}

// ── Statements ───────────────────────────────────────────────────────────────

func (c *Compiler) compileStmt(stmt ast.Statement) error {
	switch s := stmt.(type) {
	case *ast.AssignStatement:
		return c.compileAssign(s)
	case *ast.ExpressionStatement:
		if err := c.compileExpr(s.Expression); err != nil {
			return err
		}
		c.chunk.Emit(OP_POP, s.Line) // discard value; statement context
	case *ast.FunctionDeclaration:
		return c.compileFuncDecl(s)
	case *ast.ReturnStatement:
		return c.compileReturn(s)
	case *ast.IfStatement:
		return c.compileIf(s)
	case *ast.ForStatement:
		return c.compileFor(s)
	case *ast.WhileStatement:
		return c.compileWhile(s)
	case *ast.BreakStatement:
		return c.compileBreak(s.Line)
	case *ast.ContinueStatement:
		return c.compileContinue(s.Line)
	case *ast.TryCatchStatement:
		return c.compileTryCatch(s)
	case *ast.TaskStatement:
		return c.compileTask(s)
	case *ast.WaitStatement:
		c.chunk.Emit(OP_WAIT, s.Line)
	case *ast.ImportStatement:
		return c.compileImport(s)
	case *ast.ServerStatement:
		return c.compileServer(s)
	}
	return nil
}

func (c *Compiler) compileAssign(s *ast.AssignStatement) error {
	if err := c.compileExpr(s.Value); err != nil {
		return err
	}
	idx := c.chunk.InternName(s.Name)
	c.chunk.EmitU16(OP_STORE_VAR, idx, s.Line)
	return nil
}

func (c *Compiler) compileFuncDecl(s *ast.FunctionDeclaration) error {
	inner := newCompiler(s.Name, s.Params)
	for _, stmt := range s.Body {
		if err := inner.compileStmt(stmt); err != nil {
			return err
		}
	}
	inner.chunk.Emit(OP_NULL, s.Line)
	inner.chunk.Emit(OP_RETURN, s.Line)

	protoVal := &runtime.Value{
		Type:     runtime.TypeNull, // placeholder; VM detects FuncProto via ProtoVal
		ProtoVal: inner.proto,
	}
	idx := c.chunk.AddConst(protoVal)
	c.chunk.EmitU16(OP_MAKE_FUNC, idx, s.Line)

	nameIdx := c.chunk.InternName(s.Name)
	c.chunk.EmitU16(OP_STORE_VAR, nameIdx, s.Line)
	return nil
}

func (c *Compiler) compileReturn(s *ast.ReturnStatement) error {
	if err := c.compileExpr(s.Value); err != nil {
		return err
	}
	c.chunk.Emit(OP_RETURN, s.Line)
	return nil
}

func (c *Compiler) compileIf(s *ast.IfStatement) error {
	if err := c.compileExpr(s.Condition); err != nil {
		return err
	}
	// jump over Then block if condition is false
	falseJump := c.chunk.EmitJump(OP_JUMP_IF_FALSE, s.Line)

	for _, stmt := range s.Then {
		if err := c.compileStmt(stmt); err != nil {
			return err
		}
	}

	if len(s.Else) > 0 {
		// jump over Else block after Then completes
		doneJump := c.chunk.EmitJump(OP_JUMP, s.Line)
		c.chunk.PatchJump(falseJump)
		for _, stmt := range s.Else {
			if err := c.compileStmt(stmt); err != nil {
				return err
			}
		}
		c.chunk.PatchJump(doneJump)
	} else {
		c.chunk.PatchJump(falseJump)
	}
	return nil
}

func (c *Compiler) compileFor(s *ast.ForStatement) error {
	// Evaluate the iterable once and push it, then push index 0.
	if err := c.compileExpr(s.Iterable); err != nil {
		return err
	}
	c.chunk.Emit(OP_ITER_START, s.Line) // stack: [..., iterable, 0]

	loopStart := c.chunk.Len()
	c.loops = append(c.loops, loopCtx{startIP: loopStart})

	// OP_ITER_NEXT: advance or jump past loop.
	// We emit the opcode + var_idx + placeholder jump_addr (6 bytes total).
	varIdx := c.chunk.InternName(s.Variable)
	c.chunk.Emit(OP_ITER_NEXT, s.Line)
	c.chunk.writeByte(byte(varIdx>>8), s.Line)
	c.chunk.writeByte(byte(varIdx), s.Line)
	exitPatch := c.chunk.Len() // location of the 2-byte jump placeholder
	c.chunk.writeByte(0xff, s.Line)
	c.chunk.writeByte(0xff, s.Line)

	// Body
	for _, stmt := range s.Body {
		if err := c.compileStmt(stmt); err != nil {
			return err
		}
	}

	// Loop back
	c.chunk.EmitU16(OP_LOOP, uint16(loopStart), s.Line)

	// Patch exit
	exitIP := uint16(c.chunk.Len())
	c.chunk.WriteU16At(exitPatch, exitIP)

	// Patch all breaks
	lctx := c.loops[len(c.loops)-1]
	for _, bp := range lctx.breakPatches {
		c.chunk.WriteU16At(bp, exitIP)
	}
	c.loops = c.loops[:len(c.loops)-1]
	return nil
}

func (c *Compiler) compileWhile(s *ast.WhileStatement) error {
	loopStart := c.chunk.Len()
	c.loops = append(c.loops, loopCtx{startIP: loopStart})

	if err := c.compileExpr(s.Condition); err != nil {
		return err
	}
	exitPatch := c.chunk.EmitJump(OP_JUMP_IF_FALSE, s.Line)

	for _, stmt := range s.Body {
		if err := c.compileStmt(stmt); err != nil {
			return err
		}
	}

	c.chunk.EmitU16(OP_LOOP, uint16(loopStart), s.Line)

	exitIP := uint16(c.chunk.Len())
	c.chunk.PatchJump(exitPatch)

	lctx := c.loops[len(c.loops)-1]
	for _, bp := range lctx.breakPatches {
		c.chunk.WriteU16At(bp, exitIP)
	}
	c.loops = c.loops[:len(c.loops)-1]
	return nil
}

func (c *Compiler) compileBreak(line int) error {
	if len(c.loops) == 0 {
		return fmt.Errorf("line %d: break outside loop", line)
	}
	patch := c.chunk.EmitJump(OP_JUMP, line)
	c.loops[len(c.loops)-1].breakPatches = append(c.loops[len(c.loops)-1].breakPatches, patch)
	return nil
}

func (c *Compiler) compileContinue(line int) error {
	if len(c.loops) == 0 {
		return fmt.Errorf("line %d: continue outside loop", line)
	}
	start := uint16(c.loops[len(c.loops)-1].startIP)
	c.chunk.EmitU16(OP_LOOP, start, line)
	return nil
}

func (c *Compiler) compileTryCatch(s *ast.TryCatchStatement) error {
	// Push handler pointing at catch block
	handlerPatch := c.chunk.EmitJump(OP_PUSH_HANDLER, s.Line)

	for _, stmt := range s.Try {
		if err := c.compileStmt(stmt); err != nil {
			return err
		}
	}
	c.chunk.Emit(OP_POP_HANDLER, s.Line)

	// Jump over catch block (try succeeded)
	donePatch := c.chunk.EmitJump(OP_JUMP, s.Line)

	// Catch block entry
	c.chunk.PatchJump(handlerPatch)
	if s.ErrVar != "" {
		errIdx := c.chunk.InternName(s.ErrVar)
		c.chunk.EmitU16(OP_LOAD_EXCEPT, errIdx, s.Line)
	}
	for _, stmt := range s.Catch {
		if err := c.compileStmt(stmt); err != nil {
			return err
		}
	}
	c.chunk.PatchJump(donePatch)
	return nil
}

func (c *Compiler) compileTask(s *ast.TaskStatement) error {
	call, ok := s.Call.(*ast.CallExpression)
	if !ok {
		return fmt.Errorf("line %d: task requires a function call", s.Line)
	}
	// Push callee and arguments WITHOUT calling — OP_SPAWN does the goroutine spawn.
	if err := c.compileExpr(call.Callee); err != nil {
		return err
	}
	for _, arg := range call.Arguments {
		if err := c.compileExpr(arg); err != nil {
			return err
		}
	}
	c.chunk.EmitU16(OP_SPAWN, uint16(len(call.Arguments)), s.Line)
	return nil
}

func (c *Compiler) compileServer(s *ast.ServerStatement) error {
	for _, route := range s.Routes {
		// Push method string
		mIdx := c.chunk.AddConst(runtime.StringVal(route.Method))
		c.chunk.EmitU16(OP_CONSTANT, mIdx, route.Line)
		// Push path string
		pIdx := c.chunk.AddConst(runtime.StringVal(route.Path))
		c.chunk.EmitU16(OP_CONSTANT, pIdx, route.Line)
		// Compile handler body as a zero-param function (request available via env)
		inner := newCompiler("handler:"+route.Method+":"+route.Path, []string{"request"})
		for _, stmt := range route.Body {
			if err := inner.compileStmt(stmt); err != nil {
				return err
			}
		}
		inner.chunk.Emit(OP_NULL, route.Line)
		inner.chunk.Emit(OP_RETURN, route.Line)
		protoVal := &runtime.Value{Type: runtime.TypeNull, ProtoVal: inner.proto}
		hIdx := c.chunk.AddConst(protoVal)
		c.chunk.EmitU16(OP_MAKE_FUNC, hIdx, route.Line)
		// OP_BUILD_ROUTE: pops handler, path, method → pushes route triple (stored as list)
		c.chunk.Emit(OP_BUILD_ROUTE, route.Line)
	}
	c.chunk.EmitU16(OP_START_SERVER, uint16(s.Port), s.Line)
	c.chunk.writeByte(byte(len(s.Routes)>>8), s.Line)
	c.chunk.writeByte(byte(len(s.Routes)), s.Line)
	return nil
}

func (c *Compiler) compileImport(s *ast.ImportStatement) error {
	modIdx := c.chunk.InternName(s.Module)
	if len(s.Names) == 0 {
		// import math → store whole module as "math"
		c.chunk.EmitU16(OP_IMPORT, modIdx, s.Line)
		storeIdx := c.chunk.InternName(s.Module)
		c.chunk.EmitU16(OP_STORE_VAR, storeIdx, s.Line)
	} else {
		// from math import pi, sqrt
		for _, name := range s.Names {
			c.chunk.EmitU16(OP_IMPORT, modIdx, s.Line)
			memberIdx := c.chunk.InternName(name)
			c.chunk.EmitU16(OP_GET_MEMBER, memberIdx, s.Line)
			storeIdx := c.chunk.InternName(name)
			c.chunk.EmitU16(OP_STORE_VAR, storeIdx, s.Line)
		}
	}
	return nil
}

// ── Expressions ──────────────────────────────────────────────────────────────

func (c *Compiler) compileExpr(expr ast.Expression) error {
	switch e := expr.(type) {
	case *ast.NullLiteral:
		c.chunk.Emit(OP_NULL, e.Line)
	case *ast.BoolLiteral:
		if e.Value {
			c.chunk.Emit(OP_TRUE, e.Line)
		} else {
			c.chunk.Emit(OP_FALSE, e.Line)
		}
	case *ast.NumberLiteral:
		idx := c.chunk.AddConst(runtime.NumberVal(e.Value))
		c.chunk.EmitU16(OP_CONSTANT, idx, e.Line)
	case *ast.StringLiteral:
		return c.compileString(e)
	case *ast.Identifier:
		idx := c.chunk.InternName(e.Value)
		c.chunk.EmitU16(OP_LOAD_VAR, idx, e.Line)
	case *ast.ListLiteral:
		return c.compileList(e)
	case *ast.MapLiteral:
		return c.compileMap(e)
	case *ast.RangeLiteral:
		return c.compileRange(e)
	case *ast.UnaryExpression:
		return c.compileUnary(e)
	case *ast.BinaryExpression:
		return c.compileBinary(e)
	case *ast.CallExpression:
		return c.compileCall(e)
	case *ast.MemberExpression:
		return c.compileMember(e)
	case *ast.IndexExpression:
		return c.compileIndex(e)
	default:
		return fmt.Errorf("compiler: unknown expression type %T", expr)
	}
	return nil
}

func (c *Compiler) compileString(e *ast.StringLiteral) error {
	if !strings.Contains(e.Value, "{") {
		idx := c.chunk.AddConst(runtime.StringVal(e.Value))
		c.chunk.EmitU16(OP_CONSTANT, idx, e.Line)
		return nil
	}
	// String has potential interpolation: store template, resolve at runtime
	idx := c.chunk.AddConst(runtime.StringVal(e.Value))
	c.chunk.EmitU16(OP_INTERP, idx, e.Line)
	return nil
}

func (c *Compiler) compileList(e *ast.ListLiteral) error {
	for _, el := range e.Elements {
		if err := c.compileExpr(el); err != nil {
			return err
		}
	}
	c.chunk.EmitU16(OP_BUILD_LIST, uint16(len(e.Elements)), e.Line)
	return nil
}

func (c *Compiler) compileMap(e *ast.MapLiteral) error {
	for _, key := range e.Order {
		if err := c.compileExpr(key); err != nil {
			return err
		}
		if err := c.compileExpr(e.Pairs[key]); err != nil {
			return err
		}
	}
	c.chunk.EmitU16(OP_BUILD_MAP, uint16(len(e.Order)), e.Line)
	return nil
}

func (c *Compiler) compileRange(e *ast.RangeLiteral) error {
	if err := c.compileExpr(e.Start); err != nil {
		return err
	}
	if err := c.compileExpr(e.End); err != nil {
		return err
	}
	c.chunk.Emit(OP_BUILD_RANGE, e.Line)
	return nil
}

func (c *Compiler) compileUnary(e *ast.UnaryExpression) error {
	if err := c.compileExpr(e.Operand); err != nil {
		return err
	}
	switch e.Operator {
	case "-":
		c.chunk.Emit(OP_NEG, e.Line)
	case "!":
		c.chunk.Emit(OP_NOT, e.Line)
	}
	return nil
}

func (c *Compiler) compileBinary(e *ast.BinaryExpression) error {
	// Short-circuit operators
	if e.Operator == "&&" {
		if err := c.compileExpr(e.Left); err != nil {
			return err
		}
		patch := c.chunk.EmitJump(OP_JUMP_IF_FALSE_SC, e.Line)
		c.chunk.Emit(OP_POP, e.Line) // pop the truthy left value
		if err := c.compileExpr(e.Right); err != nil {
			return err
		}
		c.chunk.PatchJump(patch)
		return nil
	}
	if e.Operator == "||" {
		if err := c.compileExpr(e.Left); err != nil {
			return err
		}
		patch := c.chunk.EmitJump(OP_JUMP_IF_TRUE_SC, e.Line)
		c.chunk.Emit(OP_POP, e.Line)
		if err := c.compileExpr(e.Right); err != nil {
			return err
		}
		c.chunk.PatchJump(patch)
		return nil
	}

	if err := c.compileExpr(e.Left); err != nil {
		return err
	}
	if err := c.compileExpr(e.Right); err != nil {
		return err
	}
	switch e.Operator {
	case "+":
		c.chunk.Emit(OP_ADD, e.Line)
	case "-":
		c.chunk.Emit(OP_SUB, e.Line)
	case "*":
		c.chunk.Emit(OP_MUL, e.Line)
	case "/":
		c.chunk.Emit(OP_DIV, e.Line)
	case "%":
		c.chunk.Emit(OP_MOD, e.Line)
	case "==":
		c.chunk.Emit(OP_EQ, e.Line)
	case "!=":
		c.chunk.Emit(OP_NEQ, e.Line)
	case "<":
		c.chunk.Emit(OP_LT, e.Line)
	case ">":
		c.chunk.Emit(OP_GT, e.Line)
	case "<=":
		c.chunk.Emit(OP_LTE, e.Line)
	case ">=":
		c.chunk.Emit(OP_GTE, e.Line)
	default:
		return fmt.Errorf("compiler: unknown binary operator %q", e.Operator)
	}
	return nil
}

func (c *Compiler) compileCall(e *ast.CallExpression) error {
	if err := c.compileExpr(e.Callee); err != nil {
		return err
	}
	for _, arg := range e.Arguments {
		if err := c.compileExpr(arg); err != nil {
			return err
		}
	}
	c.chunk.EmitU16(OP_CALL, uint16(len(e.Arguments)), e.Line)
	return nil
}

func (c *Compiler) compileMember(e *ast.MemberExpression) error {
	if err := c.compileExpr(e.Object); err != nil {
		return err
	}
	idx := c.chunk.InternName(e.Property)
	if e.Safe {
		c.chunk.EmitU16(OP_SAFE_GET_MEMBER, idx, e.Line)
	} else {
		c.chunk.EmitU16(OP_GET_MEMBER, idx, e.Line)
	}
	return nil
}

func (c *Compiler) compileIndex(e *ast.IndexExpression) error {
	if err := c.compileExpr(e.Left); err != nil {
		return err
	}
	if err := c.compileExpr(e.Index); err != nil {
		return err
	}
	c.chunk.Emit(OP_GET_INDEX, e.Line)
	return nil
}

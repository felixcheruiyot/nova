package vm

import (
	"fmt"
	"math"
	"nova/runtime"
	"strings"
	"sync"
)

// vmError wraps a runtime error so the try/catch handler can intercept it.
type vmError struct{ msg string }

func (e *vmError) Error() string { return e.msg }

// handlerEntry marks the VM state at the entry of a try block.
type handlerEntry struct {
	catchIP    int
	stackDepth int
}

// frame is a single call-frame on the call stack.
type frame struct {
	chunk    *Chunk
	ip       int
	stack    []*runtime.Value
	env      *runtime.Environment
	handlers []handlerEntry
	wg       *sync.WaitGroup // non-nil when this frame has spawned tasks
}

func (f *frame) push(v *runtime.Value) { f.stack = append(f.stack, v) }
func (f *frame) pop() *runtime.Value {
	n := len(f.stack) - 1
	v := f.stack[n]
	f.stack = f.stack[:n]
	return v
}
func (f *frame) peek() *runtime.Value    { return f.stack[len(f.stack)-1] }
func (f *frame) depth() int              { return len(f.stack) }
func (f *frame) readByte() byte          { b := f.chunk.Code[f.ip]; f.ip++; return b }
func (f *frame) readU16() uint16         { hi := f.readByte(); lo := f.readByte(); return uint16(hi)<<8 | uint16(lo) }
func (f *frame) name(idx uint16) string  { return f.chunk.Names[idx] }
func (f *frame) const_(idx uint16) *runtime.Value { return f.chunk.Constants[idx] }

// VM executes bytecode.
type VM struct {
	globals *runtime.Environment
}

// New creates a VM with the standard builtins registered.
func New() *VM {
	env := runtime.NewEnvironment()
	runtime.RegisterBuiltins(env)
	return &VM{globals: env}
}

// RunProto executes a compiled FuncProto in a new enclosed environment.
func (vm *VM) RunProto(proto *FuncProto) error {
	env := runtime.NewEnclosed(vm.globals)
	_, err := vm.execFrame(proto, env, nil)
	return err
}

// CallProto calls a FuncProto with the given args and returns the result.
func (vm *VM) callProto(proto *FuncProto, capturedEnv *runtime.Environment, args []*runtime.Value) (*runtime.Value, error) {
	fnEnv := runtime.NewEnclosed(capturedEnv)
	for i, p := range proto.Params {
		if i < len(args) {
			fnEnv.Set(p, args[i])
		} else {
			fnEnv.Set(p, runtime.Null)
		}
	}
	return vm.execFrame(proto, fnEnv, nil)
}

// execFrame runs one call frame to completion and returns its return value.
func (vm *VM) execFrame(proto *FuncProto, env *runtime.Environment, args []*runtime.Value) (*runtime.Value, error) {
	f := &frame{
		chunk: proto.Chunk,
		ip:    0,
		env:   env,
	}

	for {
		if f.ip >= len(f.chunk.Code) {
			return runtime.Null, nil
		}

		op := OpCode(f.readByte())

		switch op {

		// ── Constants & literals ──────────────────────────────────────────
		case OP_CONSTANT:
			f.push(f.const_(f.readU16()))
		case OP_NULL:
			f.push(runtime.Null)
		case OP_TRUE:
			f.push(runtime.True)
		case OP_FALSE:
			f.push(runtime.False)

		// ── Stack management ─────────────────────────────────────────────
		case OP_POP:
			f.pop()

		// ── Variables ────────────────────────────────────────────────────
		case OP_LOAD_VAR:
			name := f.name(f.readU16())
			v, ok := env.Get(name)
			if !ok {
				return nil, vm.runtimeErr(f, fmt.Sprintf("undefined variable %q", name))
			}
			f.push(v)

		case OP_STORE_VAR:
			name := f.name(f.readU16())
			env.Update(name, f.peek())
			// leave value on stack (assignment is an expression in many contexts)
			// but since we always emit OP_POP after expression statements, this is fine.
			// Actually for statements we already emit POP; for initialisation in for-loops
			// the store itself is the end of the expression. Keep value on stack.
			// Wait — STORE_VAR should NOT leave the value on the stack in statement context
			// because the compiler always pairs it with explicit pops. Let's pop here.
			f.pop()

		// ── Member / index access ─────────────────────────────────────────
		case OP_GET_MEMBER:
			name := f.name(f.readU16())
			obj := f.pop()
			v, err := getMember(obj, name, f, vm)
			if err != nil {
				return nil, err
			}
			f.push(v)

		case OP_SAFE_GET_MEMBER:
			name := f.name(f.readU16())
			obj := f.pop()
			if obj.Type == runtime.TypeNull {
				f.push(runtime.Null)
			} else {
				v, err := getMember(obj, name, f, vm)
				if err != nil {
					return nil, err
				}
				f.push(v)
			}

		case OP_GET_INDEX:
			idx := f.pop()
			obj := f.pop()
			v, err := getIndex(obj, idx, f, vm)
			if err != nil {
				return nil, err
			}
			f.push(v)

		// ── Collection builders ───────────────────────────────────────────
		case OP_BUILD_LIST:
			count := int(f.readU16())
			items := make([]*runtime.Value, count)
			for i := count - 1; i >= 0; i-- {
				items[i] = f.pop()
			}
			f.push(&runtime.Value{Type: runtime.TypeList, ListVal: items})

		case OP_BUILD_MAP:
			count := int(f.readU16())
			m := &runtime.Value{
				Type:     runtime.TypeMap,
				MapVal:   make(map[string]*runtime.Value),
				MapOrder: make([]string, 0, count),
			}
			pairs := make([][2]*runtime.Value, count)
			for i := count - 1; i >= 0; i-- {
				pairs[i][1] = f.pop()
				pairs[i][0] = f.pop()
			}
			for _, p := range pairs {
				key := p[0].String()
				m.MapVal[key] = p[1]
				m.MapOrder = append(m.MapOrder, key)
			}
			f.push(m)

		case OP_BUILD_RANGE:
			end := f.pop()
			start := f.pop()
			if start.Type != runtime.TypeNumber || end.Type != runtime.TypeNumber {
				return nil, vm.runtimeErr(f, "range bounds must be numbers")
			}
			f.push(&runtime.Value{
				Type:       runtime.TypeRange,
				RangeStart: int(start.NumVal),
				RangeEnd:   int(end.NumVal),
			})

		// ── Arithmetic & logic ────────────────────────────────────────────
		case OP_NEG:
			v := f.pop()
			if v.Type != runtime.TypeNumber {
				return nil, vm.runtimeErr(f, "unary - requires a number")
			}
			f.push(runtime.NumberVal(-v.NumVal))

		case OP_NOT:
			v := f.pop()
			f.push(runtime.BoolVal(!v.IsTruthy()))

		case OP_ADD:
			b, a := f.pop(), f.pop()
			if a.Type == runtime.TypeString || b.Type == runtime.TypeString {
				f.push(runtime.StringVal(a.String() + b.String()))
			} else if a.Type == runtime.TypeNumber && b.Type == runtime.TypeNumber {
				f.push(runtime.NumberVal(a.NumVal + b.NumVal))
			} else {
				return nil, vm.runtimeErr(f, fmt.Sprintf("cannot add %s and %s", typeName(a), typeName(b)))
			}

		case OP_SUB:
			b, a := f.pop(), f.pop()
			if err := assertNums(a, b, "-", f, vm); err != nil {
				return nil, err
			}
			f.push(runtime.NumberVal(a.NumVal - b.NumVal))

		case OP_MUL:
			b, a := f.pop(), f.pop()
			if err := assertNums(a, b, "*", f, vm); err != nil {
				return nil, err
			}
			f.push(runtime.NumberVal(a.NumVal * b.NumVal))

		case OP_DIV:
			b, a := f.pop(), f.pop()
			if err := assertNums(a, b, "/", f, vm); err != nil {
				return nil, err
			}
			if b.NumVal == 0 {
				return nil, vm.runtimeErr(f, "division by zero")
			}
			f.push(runtime.NumberVal(a.NumVal / b.NumVal))

		case OP_MOD:
			b, a := f.pop(), f.pop()
			if err := assertNums(a, b, "%", f, vm); err != nil {
				return nil, err
			}
			f.push(runtime.NumberVal(math.Mod(a.NumVal, b.NumVal)))

		case OP_EQ:
			b, a := f.pop(), f.pop()
			f.push(runtime.BoolVal(valEqual(a, b)))
		case OP_NEQ:
			b, a := f.pop(), f.pop()
			f.push(runtime.BoolVal(!valEqual(a, b)))
		case OP_LT:
			b, a := f.pop(), f.pop()
			f.push(runtime.BoolVal(a.NumVal < b.NumVal))
		case OP_GT:
			b, a := f.pop(), f.pop()
			f.push(runtime.BoolVal(a.NumVal > b.NumVal))
		case OP_LTE:
			b, a := f.pop(), f.pop()
			f.push(runtime.BoolVal(a.NumVal <= b.NumVal))
		case OP_GTE:
			b, a := f.pop(), f.pop()
			f.push(runtime.BoolVal(a.NumVal >= b.NumVal))

		// ── Short-circuit ─────────────────────────────────────────────────
		case OP_JUMP_IF_FALSE_SC:
			target := f.readU16()
			if !f.peek().IsTruthy() {
				f.ip = int(target) // leave value on stack
			} else {
				f.pop() // pop the truthy value; right side will push its own
			}

		case OP_JUMP_IF_TRUE_SC:
			target := f.readU16()
			if f.peek().IsTruthy() {
				f.ip = int(target)
			} else {
				f.pop()
			}

		// ── Jumps ─────────────────────────────────────────────────────────
		case OP_JUMP:
			f.ip = int(f.readU16())

		case OP_JUMP_IF_FALSE:
			target := f.readU16()
			if !f.pop().IsTruthy() {
				f.ip = int(target)
			}

		case OP_LOOP:
			f.ip = int(f.readU16())

		// ── For-loop iterator ─────────────────────────────────────────────
		case OP_ITER_START:
			iterable := f.pop()
			f.push(iterable)
			f.push(runtime.NumberVal(0)) // index counter

		case OP_ITER_NEXT:
			varIdx := f.readU16()
			jumpAddr := f.readU16()
			// stack top: index (stack[-1]), iterable (stack[-2])
			idx := int(f.stack[len(f.stack)-1].NumVal)
			iterable := f.stack[len(f.stack)-2]

			var elem *runtime.Value
			var done bool

			switch iterable.Type {
			case runtime.TypeList:
				if idx >= len(iterable.ListVal) {
					done = true
				} else {
					elem = iterable.ListVal[idx]
				}
			case runtime.TypeRange:
				cur := iterable.RangeStart + idx
				if cur > iterable.RangeEnd {
					done = true
				} else {
					elem = runtime.NumberVal(float64(cur))
				}
			case runtime.TypeString:
				runes := []rune(iterable.StrVal)
				if idx >= len(runes) {
					done = true
				} else {
					elem = runtime.StringVal(string(runes[idx]))
				}
			case runtime.TypeMap:
				keys := iterable.MapOrder
				if idx >= len(keys) {
					done = true
				} else {
					elem = runtime.StringVal(keys[idx])
				}
			default:
				return nil, vm.runtimeErr(f, fmt.Sprintf("cannot iterate over %s", typeName(iterable)))
			}

			if done {
				f.pop() // pop index
				f.pop() // pop iterable
				f.ip = int(jumpAddr)
			} else {
				// increment index in-place on stack
				f.stack[len(f.stack)-1] = runtime.NumberVal(float64(idx + 1))
				env.Set(f.name(varIdx), elem)
			}

		// ── Call / return ─────────────────────────────────────────────────
		case OP_CALL:
			argc := int(f.readU16())
			args := make([]*runtime.Value, argc)
			for i := argc - 1; i >= 0; i-- {
				args[i] = f.pop()
			}
			callee := f.pop()
			result, err := vm.callValue(callee, args, f)
			if err != nil {
				if vmErr, ok := err.(*vmError); ok {
					if caught := vm.handleException(f, vmErr.msg, env); caught != nil {
						f.push(caught)
						continue
					}
				}
				return nil, err
			}
			f.push(result)

		case OP_RETURN:
			return f.pop(), nil

		// ── Functions ─────────────────────────────────────────────────────
		case OP_MAKE_FUNC:
			constVal := f.const_(f.readU16())
			proto := constVal.ProtoVal.(*FuncProto)
			f.push(&runtime.Value{
				Type:        runtime.TypeVMFunc,
				ProtoVal:    proto,
				CapturedEnv: env,
			})

		// ── Import ────────────────────────────────────────────────────────
		case OP_IMPORT:
			name := f.name(f.readU16())
			mod, err := runtime.LoadModule(name)
			if err != nil {
				return nil, vm.runtimeErr(f, err.Error())
			}
			f.push(mod)

		// ── Exception handling ────────────────────────────────────────────
		case OP_PUSH_HANDLER:
			catchIP := int(f.readU16())
			f.handlers = append(f.handlers, handlerEntry{
				catchIP:    catchIP,
				stackDepth: f.depth(),
			})

		case OP_POP_HANDLER:
			if len(f.handlers) > 0 {
				f.handlers = f.handlers[:len(f.handlers)-1]
			}

		case OP_LOAD_EXCEPT:
			// The current exception message was stashed by handleException; it is
			// pushed on the stack as a string before jumping here.
			name := f.name(f.readU16())
			if len(f.stack) > 0 {
				env.Set(name, f.peek())
				f.pop()
			}

		// ── Concurrency ───────────────────────────────────────────────────
		case OP_SPAWN:
			result := f.pop() // result already computed (eager); store for future async upgrade
			_ = result
			if f.wg == nil {
				var wg sync.WaitGroup
				f.wg = &wg
			}

		case OP_WAIT:
			if f.wg != nil {
				f.wg.Wait()
			}

		// ── String interpolation ──────────────────────────────────────────
		case OP_TOSTR:
			f.push(runtime.StringVal(f.pop().String()))

		case OP_INTERP:
			tmpl := f.const_(f.readU16()).StrVal
			f.push(runtime.StringVal(interpVM(tmpl, env)))

		default:
			return nil, fmt.Errorf("vm: unknown opcode %d", op)
		}
	}
}

// callValue dispatches a call to builtin, VMFunc, or returns an error.
func (vm *VM) callValue(callee *runtime.Value, args []*runtime.Value, f *frame) (*runtime.Value, error) {
	switch callee.Type {
	case runtime.TypeBuiltin:
		result, err := callee.BuiltinVal(args)
		if err != nil {
			return nil, &vmError{err.Error()}
		}
		return result, nil
	case runtime.TypeVMFunc:
		proto := callee.ProtoVal.(*FuncProto)
		return vm.callProto(proto, callee.CapturedEnv, args)
	case runtime.TypeModule:
		return nil, vm.runtimeErr(f, "modules are not callable")
	default:
		return nil, vm.runtimeErr(f, fmt.Sprintf("%s is not callable", callee.String()))
	}
}

// handleException checks if there is an active handler; if so, it restores
// the stack and jumps to the catch block by returning a sentinel value.
func (vm *VM) handleException(f *frame, msg string, env *runtime.Environment) *runtime.Value {
	if len(f.handlers) == 0 {
		return nil
	}
	h := f.handlers[len(f.handlers)-1]
	f.handlers = f.handlers[:len(f.handlers)-1]
	// Trim stack back to where it was before the try block
	for len(f.stack) > h.stackDepth {
		f.pop()
	}
	// Push exception message so OP_LOAD_EXCEPT can consume it
	f.push(runtime.StringVal(msg))
	f.ip = h.catchIP
	return runtime.Null // signals "handled" — the loop continues
}

func (vm *VM) runtimeErr(f *frame, msg string) error {
	line := f.chunk.LineFor(f.ip - 1)
	if line > 0 {
		return &vmError{fmt.Sprintf("line %d: %s", line, msg)}
	}
	return &vmError{msg}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func getMember(obj *runtime.Value, name string, f *frame, vm *VM) (*runtime.Value, error) {
	switch obj.Type {
	case runtime.TypeMap:
		if v, ok := obj.MapVal[name]; ok {
			return v, nil
		}
		return runtime.Null, nil
	case runtime.TypeModule:
		if v, ok := obj.ModuleVal[name]; ok {
			return v, nil
		}
		return runtime.Null, nil
	case runtime.TypeList:
		// Expose list methods as builtins bound to the list
		if m := listMethod(obj, name); m != nil {
			return m, nil
		}
		return runtime.Null, nil
	case runtime.TypeString:
		if m := stringMethod(obj, name); m != nil {
			return m, nil
		}
		return runtime.Null, nil
	}
	return nil, vm.runtimeErr(f, fmt.Sprintf("cannot access .%s on %s", name, typeName(obj)))
}

func getIndex(obj, idx *runtime.Value, f *frame, vm *VM) (*runtime.Value, error) {
	switch obj.Type {
	case runtime.TypeList:
		i := int(idx.NumVal)
		if i < 0 {
			i = len(obj.ListVal) + i
		}
		if i < 0 || i >= len(obj.ListVal) {
			return nil, vm.runtimeErr(f, fmt.Sprintf("index %d out of range", int(idx.NumVal)))
		}
		return obj.ListVal[i], nil
	case runtime.TypeMap:
		key := idx.String()
		if v, ok := obj.MapVal[key]; ok {
			return v, nil
		}
		return runtime.Null, nil
	case runtime.TypeString:
		runes := []rune(obj.StrVal)
		i := int(idx.NumVal)
		if i < 0 || i >= len(runes) {
			return nil, vm.runtimeErr(f, fmt.Sprintf("string index %d out of range", i))
		}
		return runtime.StringVal(string(runes[i])), nil
	}
	return nil, vm.runtimeErr(f, fmt.Sprintf("cannot index %s", typeName(obj)))
}

func listMethod(list *runtime.Value, name string) *runtime.Value {
	switch name {
	case "append":
		return &runtime.Value{Type: runtime.TypeBuiltin, BuiltinVal: func(args []*runtime.Value) (*runtime.Value, error) {
			list.ListVal = append(list.ListVal, args...)
			return runtime.Null, nil
		}}
	case "len":
		return &runtime.Value{Type: runtime.TypeBuiltin, BuiltinVal: func(args []*runtime.Value) (*runtime.Value, error) {
			return runtime.NumberVal(float64(len(list.ListVal))), nil
		}}
	}
	return nil
}

func stringMethod(s *runtime.Value, name string) *runtime.Value {
	switch name {
	case "upper":
		return &runtime.Value{Type: runtime.TypeBuiltin, BuiltinVal: func(args []*runtime.Value) (*runtime.Value, error) {
			return runtime.StringVal(strings.ToUpper(s.StrVal)), nil
		}}
	case "lower":
		return &runtime.Value{Type: runtime.TypeBuiltin, BuiltinVal: func(args []*runtime.Value) (*runtime.Value, error) {
			return runtime.StringVal(strings.ToLower(s.StrVal)), nil
		}}
	case "len":
		return &runtime.Value{Type: runtime.TypeBuiltin, BuiltinVal: func(args []*runtime.Value) (*runtime.Value, error) {
			return runtime.NumberVal(float64(len([]rune(s.StrVal)))), nil
		}}
	}
	return nil
}

func valEqual(a, b *runtime.Value) bool {
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case runtime.TypeNull:
		return true
	case runtime.TypeBool:
		return a.BoolVal == b.BoolVal
	case runtime.TypeNumber:
		return a.NumVal == b.NumVal
	case runtime.TypeString:
		return a.StrVal == b.StrVal
	}
	return false
}

func assertNums(a, b *runtime.Value, op string, f *frame, vm *VM) error {
	if a.Type != runtime.TypeNumber || b.Type != runtime.TypeNumber {
		return vm.runtimeErr(f, fmt.Sprintf("operator %s requires numbers, got %s and %s", op, typeName(a), typeName(b)))
	}
	return nil
}

func typeName(v *runtime.Value) string {
	switch v.Type {
	case runtime.TypeNull:
		return "null"
	case runtime.TypeBool:
		return "bool"
	case runtime.TypeNumber:
		return "number"
	case runtime.TypeString:
		return "string"
	case runtime.TypeList:
		return "list"
	case runtime.TypeMap:
		return "map"
	case runtime.TypeFunction, runtime.TypeVMFunc:
		return "function"
	case runtime.TypeBuiltin:
		return "builtin"
	case runtime.TypeModule:
		return "module"
	case runtime.TypeRange:
		return "range"
	}
	return "unknown"
}

// interpVM resolves {varname} patterns in a template string using the environment.
func interpVM(tmpl string, env *runtime.Environment) string {
	var sb strings.Builder
	runes := []rune(tmpl)
	i := 0
	for i < len(runes) {
		if runes[i] == '{' && i+1 < len(runes) && runes[i+1] != '{' {
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
			expr := strings.TrimSpace(string(runes[i+1 : j-1]))
			if v, ok := env.Get(expr); ok {
				sb.WriteString(v.String())
			} else {
				sb.WriteRune('{')
				sb.WriteString(expr)
				sb.WriteRune('}')
			}
			i = j
		} else {
			sb.WriteRune(runes[i])
			i++
		}
	}
	return sb.String()
}

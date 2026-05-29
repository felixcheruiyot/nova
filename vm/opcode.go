package vm

// OpCode is a single bytecode instruction byte.
type OpCode byte

const (
	OP_CONSTANT    OpCode = iota // [u16] push constants[u16]
	OP_NULL                      // push null
	OP_TRUE                      // push true
	OP_FALSE                     // push false
	OP_POP                       // discard top of stack

	OP_LOAD_VAR  // [u16] push value of names[u16]
	OP_STORE_VAR // [u16] pop and store into names[u16]

	OP_GET_MEMBER      // [u16] pop obj; push obj.names[u16]
	OP_SAFE_GET_MEMBER // [u16] pop obj; push obj?.names[u16] (null if obj is null)
	OP_GET_INDEX       // pop idx; pop obj; push obj[idx]

	OP_BUILD_LIST  // [u16] pop u16 items (bottom→top), push list
	OP_BUILD_MAP   // [u16] pop u16*2 items (key,val pairs), push map
	OP_BUILD_RANGE // pop end; pop start; push range

	OP_NEG // negate numeric top
	OP_NOT // logical-not top

	OP_ADD
	OP_SUB
	OP_MUL
	OP_DIV
	OP_MOD
	OP_EQ
	OP_NEQ
	OP_LT
	OP_GT
	OP_LTE
	OP_GTE

	// Short-circuit: peek (don't pop) top; if condition met, jump; else pop and continue
	OP_JUMP_IF_FALSE_SC // [u16] for &&: if top falsy, jump (leave value); else pop
	OP_JUMP_IF_TRUE_SC  // [u16] for ||: if top truthy, jump (leave value); else pop

	OP_JUMP          // [u16] unconditional absolute jump
	OP_JUMP_IF_FALSE // [u16] pop top; jump if falsy
	OP_LOOP          // [u16] unconditional absolute jump (backward, same encoding as JUMP)

	// For-loop iterator
	// OP_ITER_START: pops iterable, pushes (iterable, 0) — two stack slots
	OP_ITER_START
	// OP_ITER_NEXT [var_u16] [jump_u16]:
	//   peek iterable at stack[-2], index at stack[-1]
	//   if index < len(iterable): store iterable[index] in names[var_u16], increment index, continue
	//   else: pop index+iterable, jump to jump_u16
	OP_ITER_NEXT

	OP_CALL      // [u16] argc; pops argc args then callee; pushes result
	OP_RETURN    // pops return value; exits current frame

	OP_MAKE_FUNC // [u16] wraps constants[u16] (a *FuncProto) with current env; pushes VMFunc

	OP_IMPORT      // [u16] import built-in or file module names[u16]; pushes module
	OP_IMPORT_FROM // [mod_u16][name_u16] import names[name_u16] from module names[mod_u16]

	// Exception handling
	OP_PUSH_HANDLER // [u16] push catch entry-point; must be followed by try body
	OP_POP_HANDLER  // pop catch handler after try body succeeds
	OP_LOAD_EXCEPT  // [u16] store current exception in names[u16]; used at start of catch

	// Concurrency (lightweight goroutine-based)
	OP_SPAWN // pops a CallExpression closure (argc already baked in); spawns goroutine
	OP_WAIT  // waits for all spawned goroutines in the current frame

	OP_TOSTR // convert top-of-stack to string (for interpolation)
	OP_INTERP // [u16] pop env-resolved template from constants[u16]; push interpolated string
)

func (op OpCode) String() string {
	names := [...]string{
		"CONSTANT", "NULL", "TRUE", "FALSE", "POP",
		"LOAD_VAR", "STORE_VAR",
		"GET_MEMBER", "SAFE_GET_MEMBER", "GET_INDEX",
		"BUILD_LIST", "BUILD_MAP", "BUILD_RANGE",
		"NEG", "NOT",
		"ADD", "SUB", "MUL", "DIV", "MOD",
		"EQ", "NEQ", "LT", "GT", "LTE", "GTE",
		"JUMP_IF_FALSE_SC", "JUMP_IF_TRUE_SC",
		"JUMP", "JUMP_IF_FALSE", "LOOP",
		"ITER_START", "ITER_NEXT",
		"CALL", "RETURN",
		"MAKE_FUNC",
		"IMPORT", "IMPORT_FROM",
		"PUSH_HANDLER", "POP_HANDLER", "LOAD_EXCEPT",
		"SPAWN", "WAIT",
		"TOSTR", "INTERP",
	}
	if int(op) < len(names) {
		return names[op]
	}
	return "UNKNOWN"
}

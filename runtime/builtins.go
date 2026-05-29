package runtime

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"strconv"
)

type signal int

const (
	sigNone signal = iota
	sigReturn
	sigBreak
	sigContinue
)

type returnSignal struct {
	val *Value
}

type breakSignal struct{}
type continueSignal struct{}

func RegisterBuiltins(env *Environment) {
	env.Set("print", &Value{Type: TypeBuiltin, BuiltinVal: builtinPrint})
	env.Set("println", &Value{Type: TypeBuiltin, BuiltinVal: builtinPrint})
	env.Set("len", &Value{Type: TypeBuiltin, BuiltinVal: builtinLen})
	env.Set("type", &Value{Type: TypeBuiltin, BuiltinVal: builtinType})
	env.Set("str", &Value{Type: TypeBuiltin, BuiltinVal: builtinStr})
	env.Set("int", &Value{Type: TypeBuiltin, BuiltinVal: builtinInt})
	env.Set("float", &Value{Type: TypeBuiltin, BuiltinVal: builtinFloat})
	env.Set("bool", &Value{Type: TypeBuiltin, BuiltinVal: builtinBool})
	env.Set("list", &Value{Type: TypeBuiltin, BuiltinVal: builtinList})
	env.Set("range", &Value{Type: TypeBuiltin, BuiltinVal: builtinRange})
	env.Set("append", &Value{Type: TypeBuiltin, BuiltinVal: builtinAppend})
	env.Set("pop", &Value{Type: TypeBuiltin, BuiltinVal: builtinPop})
	env.Set("keys", &Value{Type: TypeBuiltin, BuiltinVal: builtinKeys})
	env.Set("values", &Value{Type: TypeBuiltin, BuiltinVal: builtinValues})
	env.Set("has", &Value{Type: TypeBuiltin, BuiltinVal: builtinHas})
	env.Set("delete", &Value{Type: TypeBuiltin, BuiltinVal: builtinDelete})
	env.Set("upper", &Value{Type: TypeBuiltin, BuiltinVal: builtinUpper})
	env.Set("lower", &Value{Type: TypeBuiltin, BuiltinVal: builtinLower})
	env.Set("split", &Value{Type: TypeBuiltin, BuiltinVal: builtinSplit})
	env.Set("join", &Value{Type: TypeBuiltin, BuiltinVal: builtinJoin})
	env.Set("contains", &Value{Type: TypeBuiltin, BuiltinVal: builtinContains})
	env.Set("trim", &Value{Type: TypeBuiltin, BuiltinVal: builtinTrim})
	env.Set("replace", &Value{Type: TypeBuiltin, BuiltinVal: builtinReplace})
	env.Set("abs", &Value{Type: TypeBuiltin, BuiltinVal: builtinAbs})
	env.Set("sqrt", &Value{Type: TypeBuiltin, BuiltinVal: builtinSqrt})
	env.Set("pow", &Value{Type: TypeBuiltin, BuiltinVal: builtinPow})
	env.Set("floor", &Value{Type: TypeBuiltin, BuiltinVal: builtinFloor})
	env.Set("ceil", &Value{Type: TypeBuiltin, BuiltinVal: builtinCeil})
	env.Set("round", &Value{Type: TypeBuiltin, BuiltinVal: builtinRound})
	env.Set("max", &Value{Type: TypeBuiltin, BuiltinVal: builtinMax})
	env.Set("min", &Value{Type: TypeBuiltin, BuiltinVal: builtinMin})
	env.Set("rand", &Value{Type: TypeBuiltin, BuiltinVal: builtinRand})
	env.Set("sort", &Value{Type: TypeBuiltin, BuiltinVal: builtinSort})
	env.Set("input", &Value{Type: TypeBuiltin, BuiltinVal: builtinInput})
	env.Set("error", &Value{Type: TypeBuiltin, BuiltinVal: builtinError})
}

func builtinPrint(args []*Value) (*Value, error) {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.String()
	}
	fmt.Println(strings.Join(parts, " "))
	return Null, nil
}

func builtinLen(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len() expects 1 argument")
	}
	switch args[0].Type {
	case TypeString:
		return NumberVal(float64(len([]rune(args[0].StrVal)))), nil
	case TypeList:
		return NumberVal(float64(len(args[0].ListVal))), nil
	case TypeMap:
		return NumberVal(float64(len(args[0].MapVal))), nil
	}
	return nil, fmt.Errorf("len() not supported for %s", typeName(args[0]))
}

func builtinType(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("type() expects 1 argument")
	}
	return StringVal(typeName(args[0])), nil
}

func typeName(v *Value) string {
	switch v.Type {
	case TypeNull:
		return "null"
	case TypeBool:
		return "bool"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeList:
		return "list"
	case TypeMap:
		return "map"
	case TypeFunction, TypeBuiltin:
		return "function"
	case TypeRange:
		return "range"
	case TypeModule:
		return "module"
	}
	return "unknown"
}

func builtinStr(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("str() expects 1 argument")
	}
	return StringVal(args[0].String()), nil
}

func builtinInt(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("int() expects 1 argument")
	}
	switch args[0].Type {
	case TypeNumber:
		return NumberVal(math.Trunc(args[0].NumVal)), nil
	case TypeString:
		n, err := strconv.ParseFloat(args[0].StrVal, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int", args[0].StrVal)
		}
		return NumberVal(math.Trunc(n)), nil
	case TypeBool:
		if args[0].BoolVal {
			return NumberVal(1), nil
		}
		return NumberVal(0), nil
	}
	return nil, fmt.Errorf("cannot convert %s to int", typeName(args[0]))
}

func builtinFloat(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("float() expects 1 argument")
	}
	switch args[0].Type {
	case TypeNumber:
		return args[0], nil
	case TypeString:
		n, err := strconv.ParseFloat(args[0].StrVal, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float", args[0].StrVal)
		}
		return NumberVal(n), nil
	}
	return nil, fmt.Errorf("cannot convert %s to float", typeName(args[0]))
}

func builtinBool(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("bool() expects 1 argument")
	}
	return BoolVal(args[0].IsTruthy()), nil
}

func builtinList(args []*Value) (*Value, error) {
	if len(args) == 0 {
		return &Value{Type: TypeList}, nil
	}
	if len(args) == 1 && args[0].Type == TypeRange {
		var elems []*Value
		for i := args[0].RangeStart; i <= args[0].RangeEnd; i++ {
			elems = append(elems, NumberVal(float64(i)))
		}
		return &Value{Type: TypeList, ListVal: elems}, nil
	}
	return nil, fmt.Errorf("list() expects 0 arguments or a range")
}

func builtinRange(args []*Value) (*Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("range() expects 1 or 2 arguments")
	}
	if args[0].Type != TypeNumber {
		return nil, fmt.Errorf("range() expects numbers")
	}
	start, end := 0, int(args[0].NumVal)
	if len(args) == 2 {
		if args[1].Type != TypeNumber {
			return nil, fmt.Errorf("range() expects numbers")
		}
		start = int(args[0].NumVal)
		end = int(args[1].NumVal)
	}
	return &Value{Type: TypeRange, RangeStart: start, RangeEnd: end - 1}, nil
}

func builtinAppend(args []*Value) (*Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("append() expects at least 2 arguments")
	}
	if args[0].Type != TypeList {
		return nil, fmt.Errorf("append() first argument must be a list")
	}
	args[0].ListVal = append(args[0].ListVal, args[1:]...)
	return args[0], nil
}

func builtinPop(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("pop() expects 1 argument")
	}
	if args[0].Type != TypeList || len(args[0].ListVal) == 0 {
		return nil, fmt.Errorf("pop() requires a non-empty list")
	}
	n := len(args[0].ListVal)
	val := args[0].ListVal[n-1]
	args[0].ListVal = args[0].ListVal[:n-1]
	return val, nil
}

func builtinKeys(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeMap {
		return nil, fmt.Errorf("keys() expects a map")
	}
	elems := make([]*Value, len(args[0].MapOrder))
	for i, k := range args[0].MapOrder {
		elems[i] = StringVal(k)
	}
	return &Value{Type: TypeList, ListVal: elems}, nil
}

func builtinValues(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeMap {
		return nil, fmt.Errorf("values() expects a map")
	}
	elems := make([]*Value, len(args[0].MapOrder))
	for i, k := range args[0].MapOrder {
		elems[i] = args[0].MapVal[k]
	}
	return &Value{Type: TypeList, ListVal: elems}, nil
}

func builtinHas(args []*Value) (*Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("has() expects 2 arguments")
	}
	if args[0].Type != TypeMap {
		return nil, fmt.Errorf("has() first argument must be a map")
	}
	_, ok := args[0].MapVal[args[1].String()]
	return BoolVal(ok), nil
}

func builtinDelete(args []*Value) (*Value, error) {
	if len(args) != 2 || args[0].Type != TypeMap {
		return nil, fmt.Errorf("delete() expects a map and a key")
	}
	k := args[1].String()
	delete(args[0].MapVal, k)
	newOrder := args[0].MapOrder[:0]
	for _, o := range args[0].MapOrder {
		if o != k {
			newOrder = append(newOrder, o)
		}
	}
	args[0].MapOrder = newOrder
	return Null, nil
}

func builtinUpper(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeString {
		return nil, fmt.Errorf("upper() expects a string")
	}
	return StringVal(strings.ToUpper(args[0].StrVal)), nil
}

func builtinLower(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeString {
		return nil, fmt.Errorf("lower() expects a string")
	}
	return StringVal(strings.ToLower(args[0].StrVal)), nil
}

func builtinSplit(args []*Value) (*Value, error) {
	if len(args) != 2 || args[0].Type != TypeString || args[1].Type != TypeString {
		return nil, fmt.Errorf("split() expects two strings")
	}
	parts := strings.Split(args[0].StrVal, args[1].StrVal)
	elems := make([]*Value, len(parts))
	for i, p := range parts {
		elems[i] = StringVal(p)
	}
	return &Value{Type: TypeList, ListVal: elems}, nil
}

func builtinJoin(args []*Value) (*Value, error) {
	if len(args) != 2 || args[0].Type != TypeList || args[1].Type != TypeString {
		return nil, fmt.Errorf("join() expects a list and a separator string")
	}
	parts := make([]string, len(args[0].ListVal))
	for i, v := range args[0].ListVal {
		parts[i] = v.String()
	}
	return StringVal(strings.Join(parts, args[1].StrVal)), nil
}

func builtinContains(args []*Value) (*Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("contains() expects 2 arguments")
	}
	switch args[0].Type {
	case TypeString:
		return BoolVal(strings.Contains(args[0].StrVal, args[1].String())), nil
	case TypeList:
		for _, el := range args[0].ListVal {
			if el.String() == args[1].String() {
				return True, nil
			}
		}
		return False, nil
	}
	return nil, fmt.Errorf("contains() expects a string or list")
}

func builtinTrim(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeString {
		return nil, fmt.Errorf("trim() expects a string")
	}
	return StringVal(strings.TrimSpace(args[0].StrVal)), nil
}

func builtinReplace(args []*Value) (*Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("replace() expects 3 arguments")
	}
	return StringVal(strings.ReplaceAll(args[0].StrVal, args[1].StrVal, args[2].StrVal)), nil
}

func builtinAbs(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeNumber {
		return nil, fmt.Errorf("abs() expects a number")
	}
	return NumberVal(math.Abs(args[0].NumVal)), nil
}

func builtinSqrt(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeNumber {
		return nil, fmt.Errorf("sqrt() expects a number")
	}
	return NumberVal(math.Sqrt(args[0].NumVal)), nil
}

func builtinPow(args []*Value) (*Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pow() expects 2 numbers")
	}
	return NumberVal(math.Pow(args[0].NumVal, args[1].NumVal)), nil
}

func builtinFloor(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeNumber {
		return nil, fmt.Errorf("floor() expects a number")
	}
	return NumberVal(math.Floor(args[0].NumVal)), nil
}

func builtinCeil(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeNumber {
		return nil, fmt.Errorf("ceil() expects a number")
	}
	return NumberVal(math.Ceil(args[0].NumVal)), nil
}

func builtinRound(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeNumber {
		return nil, fmt.Errorf("round() expects a number")
	}
	return NumberVal(math.Round(args[0].NumVal)), nil
}

func builtinMax(args []*Value) (*Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("max() expects at least 1 argument")
	}
	m := args[0].NumVal
	for _, a := range args[1:] {
		if a.NumVal > m {
			m = a.NumVal
		}
	}
	return NumberVal(m), nil
}

func builtinMin(args []*Value) (*Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("min() expects at least 1 argument")
	}
	m := args[0].NumVal
	for _, a := range args[1:] {
		if a.NumVal < m {
			m = a.NumVal
		}
	}
	return NumberVal(m), nil
}

func builtinRand(args []*Value) (*Value, error) {
	return NumberVal(rand.Float64()), nil
}

func builtinSort(args []*Value) (*Value, error) {
	if len(args) != 1 || args[0].Type != TypeList {
		return nil, fmt.Errorf("sort() expects a list")
	}
	lst := make([]*Value, len(args[0].ListVal))
	copy(lst, args[0].ListVal)
	sort.Slice(lst, func(i, j int) bool {
		a, b := lst[i], lst[j]
		if a.Type == TypeNumber && b.Type == TypeNumber {
			return a.NumVal < b.NumVal
		}
		return a.String() < b.String()
	})
	return &Value{Type: TypeList, ListVal: lst}, nil
}

func builtinInput(args []*Value) (*Value, error) {
	if len(args) > 0 {
		fmt.Print(args[0].String())
	}
	var line string
	fmt.Scanln(&line)
	return StringVal(line), nil
}

func builtinError(args []*Value) (*Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("error() expects 1 argument")
	}
	return nil, fmt.Errorf("%s", args[0].String())
}

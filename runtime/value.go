package runtime

import (
	"fmt"
	"math"
	"strings"
)

type ValueType int

const (
	TypeNull ValueType = iota
	TypeBool
	TypeNumber
	TypeString
	TypeList
	TypeMap
	TypeFunction
	TypeBuiltin
	TypeRange
	TypeModule
)

type Value struct {
	Type     ValueType
	BoolVal  bool
	NumVal   float64
	StrVal   string
	ListVal  []*Value
	MapVal   map[string]*Value
	MapOrder []string
	RangeStart, RangeEnd int
	FuncVal  *Function
	BuiltinVal func(args []*Value) (*Value, error)
	ModuleVal  map[string]*Value
}

type Function struct {
	Name   string
	Params []string
	Body   interface{} // []ast.Statement — avoid circular import, cast in evaluator
	Env    *Environment
}

var Null = &Value{Type: TypeNull}
var True = &Value{Type: TypeBool, BoolVal: true}
var False = &Value{Type: TypeBool, BoolVal: false}

func NumberVal(n float64) *Value  { return &Value{Type: TypeNumber, NumVal: n} }
func StringVal(s string) *Value   { return &Value{Type: TypeString, StrVal: s} }
func BoolVal(b bool) *Value {
	if b {
		return True
	}
	return False
}

func (v *Value) IsTruthy() bool {
	switch v.Type {
	case TypeNull:
		return false
	case TypeBool:
		return v.BoolVal
	case TypeNumber:
		return v.NumVal != 0
	case TypeString:
		return v.StrVal != ""
	case TypeList:
		return len(v.ListVal) > 0
	case TypeMap:
		return len(v.MapVal) > 0
	}
	return true
}

func (v *Value) String() string {
	switch v.Type {
	case TypeNull:
		return "null"
	case TypeBool:
		if v.BoolVal {
			return "true"
		}
		return "false"
	case TypeNumber:
		if v.NumVal == math.Trunc(v.NumVal) {
			return fmt.Sprintf("%d", int64(v.NumVal))
		}
		return fmt.Sprintf("%g", v.NumVal)
	case TypeString:
		return v.StrVal
	case TypeList:
		parts := make([]string, len(v.ListVal))
		for i, el := range v.ListVal {
			parts[i] = el.Repr()
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case TypeMap:
		parts := make([]string, 0, len(v.MapOrder))
		for _, k := range v.MapOrder {
			parts = append(parts, fmt.Sprintf("%q: %s", k, v.MapVal[k].Repr()))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case TypeFunction:
		return fmt.Sprintf("<func %s>", v.FuncVal.Name)
	case TypeBuiltin:
		return "<builtin>"
	case TypeRange:
		return fmt.Sprintf("%d..%d", v.RangeStart, v.RangeEnd)
	case TypeModule:
		return "<module>"
	}
	return "?"
}

// Repr returns a quoted representation for display inside collections
func (v *Value) Repr() string {
	if v.Type == TypeString {
		return fmt.Sprintf("%q", v.StrVal)
	}
	return v.String()
}

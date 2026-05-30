package runtime

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

var builtinModules = map[string]func() *Value{
	"io":   moduleIO,
	"math": moduleMath,
	"json": moduleJSON,
	"time": moduleTime,
	"fs":   moduleFS,
	"ai":   moduleAI,
}

func registerModules(env *Environment) {
	for name, fn := range builtinModules {
		env.Set(name, fn())
	}
}

// LoadModule returns a built-in module by name, or an error if unknown.
// It also searches nova_modules/ in the current directory for user packages.
func LoadModule(name string) (*Value, error) {
	if fn, ok := builtinModules[name]; ok {
		return fn(), nil
	}
	// User package: look for nova_modules/<name>.nv or nova_modules/<name>/init.nv
	candidates := []string{
		"nova_modules/" + name + ".nv",
		"nova_modules/" + name + "/init.nv",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return loadFileModule(path, name)
		}
	}
	return nil, fmt.Errorf("unknown module %q", name)
}

// loadFileModule executes a .nv source file and exposes its globals as a module.
func loadFileModule(path, name string) (*Value, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("module %q: %w", name, err)
	}
	// FileRunner is set by main to parse+run a source string in a given env.
	if FileRunner == nil {
		return nil, fmt.Errorf("module %q: file module loader not initialised", name)
	}
	env := NewEnvironment()
	RegisterBuiltins(env)
	registerModules(env)
	if err := FileRunner(string(src), env); err != nil {
		return nil, fmt.Errorf("module %q: %w", name, err)
	}
	return &Value{Type: TypeModule, ModuleVal: env.vars}, nil
}

// FileRunner is set by main.go to parse+evaluate source without creating an
// import cycle between runtime and the lexer/parser packages.
var FileRunner func(src string, env *Environment) error

func moduleIO() *Value {
	m := map[string]*Value{
		"print": {Type: TypeBuiltin, BuiltinVal: builtinPrint},
		"read": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			var line string
			fmt.Scanln(&line)
			return StringVal(line), nil
		}},
		"write": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("io.write() expects at least 1 argument")
			}
			fmt.Print(args[0].String())
			return Null, nil
		}},
	}
	return &Value{Type: TypeModule, ModuleVal: m}
}

func moduleMath() *Value {
	m := map[string]*Value{
		"pi":  NumberVal(math.Pi),
		"e":   NumberVal(math.E),
		"inf": NumberVal(math.Inf(1)),
		"abs": {Type: TypeBuiltin, BuiltinVal: builtinAbs},
		"sqrt": {Type: TypeBuiltin, BuiltinVal: builtinSqrt},
		"pow": {Type: TypeBuiltin, BuiltinVal: builtinPow},
		"floor": {Type: TypeBuiltin, BuiltinVal: builtinFloor},
		"ceil": {Type: TypeBuiltin, BuiltinVal: builtinCeil},
		"round": {Type: TypeBuiltin, BuiltinVal: builtinRound},
		"max": {Type: TypeBuiltin, BuiltinVal: builtinMax},
		"min": {Type: TypeBuiltin, BuiltinVal: builtinMin},
		"log": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeNumber {
				return nil, fmt.Errorf("math.log() expects a number")
			}
			return NumberVal(math.Log(args[0].NumVal)), nil
		}},
		"sin": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeNumber {
				return nil, fmt.Errorf("math.sin() expects a number")
			}
			return NumberVal(math.Sin(args[0].NumVal)), nil
		}},
		"cos": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeNumber {
				return nil, fmt.Errorf("math.cos() expects a number")
			}
			return NumberVal(math.Cos(args[0].NumVal)), nil
		}},
	}
	return &Value{Type: TypeModule, ModuleVal: m}
}

func moduleJSON() *Value {
	m := map[string]*Value{
		"stringify": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("json.stringify() expects 1 argument")
			}
			data, err := valueToJSON(args[0])
			if err != nil {
				return nil, err
			}
			b, err := json.Marshal(data)
			if err != nil {
				return nil, err
			}
			return StringVal(string(b)), nil
		}},
		"parse": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeString {
				return nil, fmt.Errorf("json.parse() expects a string")
			}
			var data interface{}
			if err := json.Unmarshal([]byte(args[0].StrVal), &data); err != nil {
				return nil, fmt.Errorf("json.parse(): %v", err)
			}
			return jsonToValue(data), nil
		}},
	}
	return &Value{Type: TypeModule, ModuleVal: m}
}

func valueToJSON(v *Value) (interface{}, error) {
	switch v.Type {
	case TypeNull:
		return nil, nil
	case TypeBool:
		return v.BoolVal, nil
	case TypeNumber:
		return v.NumVal, nil
	case TypeString:
		return v.StrVal, nil
	case TypeList:
		arr := make([]interface{}, len(v.ListVal))
		for i, el := range v.ListVal {
			j, err := valueToJSON(el)
			if err != nil {
				return nil, err
			}
			arr[i] = j
		}
		return arr, nil
	case TypeMap:
		obj := make(map[string]interface{})
		for k, val := range v.MapVal {
			j, err := valueToJSON(val)
			if err != nil {
				return nil, err
			}
			obj[k] = j
		}
		return obj, nil
	}
	return nil, fmt.Errorf("cannot JSON encode %s", typeName(v))
}

func jsonToValue(v interface{}) *Value {
	if v == nil {
		return Null
	}
	switch val := v.(type) {
	case bool:
		return BoolVal(val)
	case float64:
		return NumberVal(val)
	case string:
		return StringVal(val)
	case []interface{}:
		elems := make([]*Value, len(val))
		for i, el := range val {
			elems[i] = jsonToValue(el)
		}
		return &Value{Type: TypeList, ListVal: elems}
	case map[string]interface{}:
		m := &Value{Type: TypeMap, MapVal: make(map[string]*Value)}
		for k, el := range val {
			m.MapVal[k] = jsonToValue(el)
			m.MapOrder = append(m.MapOrder, k)
		}
		return m
	}
	return Null
}

func moduleTime() *Value {
	m := map[string]*Value{
		"now": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			return NumberVal(float64(time.Now().UnixMilli())), nil
		}},
		"sleep": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeNumber {
				return nil, fmt.Errorf("time.sleep() expects milliseconds as a number")
			}
			time.Sleep(time.Duration(args[0].NumVal) * time.Millisecond)
			return Null, nil
		}},
		"format": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			t := time.Now()
			layout := "2006-01-02 15:04:05"
			if len(args) > 0 {
				layout = args[0].StrVal
			}
			return StringVal(t.Format(layout)), nil
		}},
	}
	return &Value{Type: TypeModule, ModuleVal: m}
}

func moduleFS() *Value {
	m := map[string]*Value{
		"read": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeString {
				return nil, fmt.Errorf("fs.read() expects a file path")
			}
			data, err := os.ReadFile(args[0].StrVal)
			if err != nil {
				return nil, err
			}
			return StringVal(string(data)), nil
		}},
		"write": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("fs.write() expects path and content")
			}
			return Null, os.WriteFile(args[0].StrVal, []byte(args[1].StrVal), 0644)
		}},
		"exists": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeString {
				return nil, fmt.Errorf("fs.exists() expects a file path")
			}
			_, err := os.Stat(args[0].StrVal)
			return BoolVal(err == nil), nil
		}},
		"lines": {Type: TypeBuiltin, BuiltinVal: func(args []*Value) (*Value, error) {
			if len(args) != 1 || args[0].Type != TypeString {
				return nil, fmt.Errorf("fs.lines() expects a file path")
			}
			data, err := os.ReadFile(args[0].StrVal)
			if err != nil {
				return nil, err
			}
			parts := strings.Split(string(data), "\n")
			elems := make([]*Value, len(parts))
			for i, p := range parts {
				elems[i] = StringVal(p)
			}
			return &Value{Type: TypeList, ListVal: elems}, nil
		}},
	}
	return &Value{Type: TypeModule, ModuleVal: m}
}

package runtime

type Environment struct {
	vars   map[string]*Value
	outer  *Environment
}

func NewEnvironment() *Environment {
	return &Environment{vars: make(map[string]*Value)}
}

func NewEnclosed(outer *Environment) *Environment {
	e := NewEnvironment()
	e.outer = outer
	return e
}

func (e *Environment) Get(name string) (*Value, bool) {
	v, ok := e.vars[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return v, ok
}

func (e *Environment) Set(name string, val *Value) {
	e.vars[name] = val
}

// Update walks up the chain to update an existing binding; sets locally if not found
func (e *Environment) Update(name string, val *Value) {
	env := e
	for env != nil {
		if _, ok := env.vars[name]; ok {
			env.vars[name] = val
			return
		}
		env = env.outer
	}
	e.vars[name] = val
}

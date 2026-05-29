package vm

// FuncProto is the compiled (static) representation of a Nova function.
// It is stored as a constant in the enclosing chunk and wrapped with a
// captured environment at runtime to form a VMFunc closure.
type FuncProto struct {
	Name   string
	Params []string
	Chunk  *Chunk
}

func (p *FuncProto) GetName() string { return p.Name }

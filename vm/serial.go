package vm

import (
	"encoding/gob"
	"io"
	"nova/runtime"
)

// ── Serialisable mirror types ─────────────────────────────────────────────────
// gob cannot encode interface{} without registration, so we use concrete types.

type serialConst struct {
	Kind    int     // 0=null, 1=bool, 2=number, 3=string, 4=proto
	BoolVal bool
	NumVal  float64
	StrVal  string
	Proto   *serialProto
}

type serialChunk struct {
	Code   []byte
	Names  []string
	Lines  []int
	Consts []serialConst
}

type serialProto struct {
	Name   string
	Params []string
	Chunk  serialChunk
}

func init() {
	gob.Register(&serialProto{})
}

// Encode writes proto to w in gob format.
func Encode(w io.Writer, proto *FuncProto) error {
	enc := gob.NewEncoder(w)
	return enc.Encode(marshalProto(proto))
}

// Decode reads and returns a FuncProto from r.
func Decode(r io.Reader) (*FuncProto, error) {
	var sp serialProto
	if err := gob.NewDecoder(r).Decode(&sp); err != nil {
		return nil, err
	}
	return unmarshalProto(&sp), nil
}

func marshalProto(p *FuncProto) *serialProto {
	return &serialProto{
		Name:   p.Name,
		Params: p.Params,
		Chunk:  marshalChunk(p.Chunk),
	}
}

func marshalChunk(c *Chunk) serialChunk {
	consts := make([]serialConst, len(c.Constants))
	for i, v := range c.Constants {
		consts[i] = marshalConst(v)
	}
	return serialChunk{
		Code:   c.Code,
		Names:  c.Names,
		Lines:  c.Lines,
		Consts: consts,
	}
}

func marshalConst(v *runtime.Value) serialConst {
	// FuncProto constants are stored as TypeNull placeholders with ProtoVal set —
	// check this first before switching on Type.
	if v.ProtoVal != nil {
		return serialConst{Kind: 4, Proto: marshalProto(v.ProtoVal.(*FuncProto))}
	}
	switch v.Type {
	case runtime.TypeBool:
		return serialConst{Kind: 1, BoolVal: v.BoolVal}
	case runtime.TypeNumber:
		return serialConst{Kind: 2, NumVal: v.NumVal}
	case runtime.TypeString:
		return serialConst{Kind: 3, StrVal: v.StrVal}
	}
	return serialConst{Kind: 0} // null
}

func unmarshalProto(sp *serialProto) *FuncProto {
	return &FuncProto{
		Name:   sp.Name,
		Params: sp.Params,
		Chunk:  unmarshalChunk(sp.Chunk),
	}
}

func unmarshalChunk(sc serialChunk) *Chunk {
	c := &Chunk{
		Code:  sc.Code,
		Names: sc.Names,
		Lines: sc.Lines,
	}
	for _, sc := range sc.Consts {
		c.Constants = append(c.Constants, unmarshalConst(sc))
	}
	return c
}

func unmarshalConst(sc serialConst) *runtime.Value {
	switch sc.Kind {
	case 1:
		return runtime.BoolVal(sc.BoolVal)
	case 2:
		return runtime.NumberVal(sc.NumVal)
	case 3:
		return runtime.StringVal(sc.StrVal)
	case 4:
		proto := unmarshalProto(sc.Proto)
		return &runtime.Value{Type: runtime.TypeNull, ProtoVal: proto}
	}
	return runtime.Null
}

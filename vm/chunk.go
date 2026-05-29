package vm

import "nova/runtime"

// Chunk is the bytecode and metadata for one function scope.
type Chunk struct {
	Code      []byte
	Constants []*runtime.Value
	Names     []string // interned string pool: variable / member names
	Lines     []int    // source line per byte in Code
}

func NewChunk() *Chunk { return &Chunk{} }

func (c *Chunk) writeByte(b byte, line int) {
	c.Code = append(c.Code, b)
	c.Lines = append(c.Lines, line)
}

func (c *Chunk) Emit(op OpCode, line int) {
	c.writeByte(byte(op), line)
}

func (c *Chunk) EmitU16(op OpCode, arg uint16, line int) {
	c.writeByte(byte(op), line)
	c.writeByte(byte(arg>>8), line)
	c.writeByte(byte(arg), line)
}

func (c *Chunk) WriteU16At(offset int, arg uint16) {
	c.Code[offset] = byte(arg >> 8)
	c.Code[offset+1] = byte(arg)
}

// EmitJump emits op with a 0xFFFF placeholder and returns the offset to patch.
func (c *Chunk) EmitJump(op OpCode, line int) int {
	c.writeByte(byte(op), line)
	c.writeByte(0xff, line)
	c.writeByte(0xff, line)
	return len(c.Code) - 2
}

// PatchJump fills in the jump target at patchOffset to point at current end.
func (c *Chunk) PatchJump(patchOffset int) {
	c.WriteU16At(patchOffset, uint16(len(c.Code)))
}

// AddConst appends v to the constant pool and returns its index.
func (c *Chunk) AddConst(v *runtime.Value) uint16 {
	c.Constants = append(c.Constants, v)
	return uint16(len(c.Constants) - 1)
}

// InternName returns the pool index of name, adding it if absent.
func (c *Chunk) InternName(name string) uint16 {
	for i, n := range c.Names {
		if n == name {
			return uint16(i)
		}
	}
	c.Names = append(c.Names, name)
	return uint16(len(c.Names) - 1)
}

func (c *Chunk) Len() int { return len(c.Code) }

// LineFor returns the source line for the instruction at ip.
func (c *Chunk) LineFor(ip int) int {
	if ip >= 0 && ip < len(c.Lines) {
		return c.Lines[ip]
	}
	return -1
}

// ReadU16 reads a big-endian uint16 at Code[ip].
func (c *Chunk) ReadU16(ip int) uint16 {
	return uint16(c.Code[ip])<<8 | uint16(c.Code[ip+1])
}

package st

import (
	"bytes"
	"fmt"
	"slices"
	"sync"

	"golang.org/x/crypto/sha3"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// MaxCodeSize is the maximum size of a contract stored on a Ethereum
// compatible block chain.
const MaxCodeSize = 1<<14 + 1<<13 // = 24576

// Code is an immutable representation of EVM byte code which may be freely
// copied and shared through shallow copies.
type Code struct {
	code           []byte
	isCode         []bool
	hash           [32]byte
	hashCalculated bool
	hashMutex      sync.Mutex
}

// ErrInvalidPosition is an error produced by observer functions on the Code if
// specified positions are invalid.
const ErrInvalidPosition = ConstErr("invalid position")

// NewCode creates an immutable code representation based on the given raw
// code representation. The resulting code contains a copy of the provided code
// to guarantee immutability.
func NewCode(code []byte) *Code {
	isCode := make([]bool, 0, len(code)+32)
	for i := 0; i < len(code); i++ {
		isCode = append(isCode, true)
		op := OpCode(code[i])
		if PUSH1 <= op && op <= PUSH32 {
			width := int(op - PUSH1 + 1)
			isCode = append(isCode, make([]bool, width)...)
			i += width
		}
	}

	return &Code{
		code:   slices.Clone(code),
		isCode: isCode,
	}
}

func (c *Code) Clone() *Code {
	return c
}

func (c *Code) Length() int {
	return len(c.code)
}

func (c *Code) Hash() [32]byte {
	c.hashMutex.Lock()
	defer c.hashMutex.Unlock()

	if !c.hashCalculated {
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(c.code)
		copy(c.hash[:], hasher.Sum(nil)[:])
		c.hashCalculated = true
	}
	return c.hash
}

func (c *Code) IsCode(pos int) bool {
	if pos < 0 || pos >= len(c.isCode) {
		return true // out-of-bounds STOP
	}
	return c.isCode[pos]
}

func (c *Code) IsData(pos int) bool {
	return !c.IsCode(pos)
}

func (c *Code) GetOperation(pos int) (OpCode, error) {
	if pos < 0 || pos >= len(c.isCode) {
		return STOP, nil
	}
	if !c.isCode[pos] {
		return INVALID, ErrInvalidPosition
	}
	return OpCode(c.code[pos]), nil
}

func (c *Code) GetData(pos int) (byte, error) {
	if !c.IsData(pos) {
		return 0, ErrInvalidPosition
	}
	if pos >= len(c.code) {
		return 0, nil
	}
	return c.code[pos], nil
}

func (c *Code) GetSlice(start, end int) []byte {
	if start == end {
		return []byte{}
	}
	if start > c.Length() || start > end {
		return []byte{}
	}
	return c.code[start:end]
}

func (c *Code) Eq(other *Code) bool {
	return c.Hash() == other.Hash() && bytes.Equal(c.code, other.code)
}

func (a *Code) Diff(b *Code) (res []string) {
	if a.Length() != b.Length() {
		res = append(res, fmt.Sprintf("Different code size: %v vs %v", a.Length(), b.Length()))
		return
	}
	for i := 0; i < a.Length(); i++ {
		if aValue, bValue := a.code[i], b.code[i]; aValue != bValue {
			res = append(res, fmt.Sprintf("Different code/data at position %d: 0x%02x vs 0x%02x", i, aValue, bValue))
		}
	}
	return
}

func (c *Code) CopyTo(dst []byte) int {
	return copy(dst, c.code)
}

func (c *Code) String() string {
	return fmt.Sprintf("%x", c.code)
}

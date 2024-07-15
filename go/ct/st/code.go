// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"sync"

	"golang.org/x/crypto/sha3"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
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
		op := vm.OpCode(code[i])
		if vm.PUSH1 <= op && op <= vm.PUSH32 {
			width := int(op - vm.PUSH1 + 1)
			isCode = append(isCode, make([]bool, width)...)
			i += width
		}
	}

	return &Code{
		code:   slices.Clone(code)[:len(code):len(code)],
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

func (c *Code) GetOperation(pos int) (vm.OpCode, error) {
	if pos < 0 || pos >= len(c.isCode) {
		return vm.STOP, nil
	}
	if !c.isCode[pos] {
		return vm.INVALID, ErrInvalidPosition
	}
	return vm.OpCode(c.code[pos]), nil
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

// CopyCodeSlice copies code from the slice [start:end] to dst.
// Returns the number of elements copied.
func (c *Code) CopyCodeSlice(start, end int, dst []byte) int {
	return copy(dst, c.code[start:end])
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

func (c *Code) Copy() []byte {
	return bytes.Clone(c.code)
}

func (c *Code) String() string {
	return fmt.Sprintf("%x", c.code)
}

// ToHumanReadableString returns a string with the length of the code and the
// human readable form for the opcodes in range [start, start+length).
// - If the slice to be printed overflows the existing code, the overlapping code is printed.
// - If start exceeds the code length, the length of the code is printed.
// - Data found in the code is printed as decimal numbers (to differentiate from unused opcodes).
func (c *Code) ToHumanReadableString(start int, length int) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("len(%d)", len(c.code)))
	if start >= len(c.code) {
		return builder.String()
	}

	end := min(length, len(c.code)-start) + start
	for i, op := range c.code[start:end] {
		var entry string
		if c.IsCode(start + i) {
			entry = vm.OpCode(op).String()
		} else {
			entry = fmt.Sprintf("%d", op)
		}
		builder.WriteString(" ")
		builder.WriteString(entry)
	}
	return builder.String()
}

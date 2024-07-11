// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package tosca

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"math/bits"
	"strings"

	"github.com/holiman/uint256"
)

func (a Address) String() string {
	return fmt.Sprintf("0x%x", a[:])
}

func (a Address) MarshalText() ([]byte, error) {
	return bytesToText(a[:])
}

func (a *Address) UnmarshalText(data []byte) error {
	return textToBytes(a[:], data)
}

func (k Key) String() string {
	return fmt.Sprintf("0x%x", k[:])
}

func (w Word) String() string {
	return fmt.Sprintf("0x%x", w[:])
}

func (v Value) ToBig() *big.Int {
	return new(big.Int).SetBytes(v[:])
}

func (v Value) ToUint256() *uint256.Int {
	return new(uint256.Int).SetBytes(v[:])
}

func (v Value) String() string {
	return v.ToUint256().String()
}

func (v Value) Cmp(o Value) int {
	return bytes.Compare(v[:], o[:])
}

// NewValue creates a new Value instance from up to 4 uint64 arguments. The
// arguments are given in the order from most significant to least significant
// by padding leading zeros as needed. No argument results in a value of zero.
func NewValue(args ...uint64) (result Value) {
	if len(args) > 4 {
		panic("Too many arguments")
	}
	offset := 4 - len(args)
	for i := 0; i < len(args) && i < 4; i++ {
		start := (offset * 8) + i*8
		end := start + 8
		binary.BigEndian.PutUint64(result[start:end], args[i])
	}
	return
}

// ValueFromUint256 converts a *uint256.Int to a Value.
// If the input is nil, it returns 0.
func ValueFromUint256(value *uint256.Int) (result Value) {
	if value == nil {
		return result
	}
	return value.Bytes32()
}

func Add(a, b Value) (z Value) {
	res, carry := bits.Add64(a.getInternalUint64(0), b.getInternalUint64(0), 0)
	binary.BigEndian.PutUint64(z[24:32], res)

	res, carry = bits.Add64(a.getInternalUint64(1), b.getInternalUint64(1), carry)
	binary.BigEndian.PutUint64(z[16:24], res)

	res, carry = bits.Add64(a.getInternalUint64(2), b.getInternalUint64(2), carry)
	binary.BigEndian.PutUint64(z[8:16], res)

	res, _ = bits.Add64(a.getInternalUint64(3), b.getInternalUint64(3), carry)
	binary.BigEndian.PutUint64(z[0:8], res)

	return z
}

func Sub(a, b Value) (z Value) {
	res, carry := bits.Sub64(a.getInternalUint64(0), b.getInternalUint64(0), 0)
	binary.BigEndian.PutUint64(z[24:32], res)

	res, carry = bits.Sub64(a.getInternalUint64(1), b.getInternalUint64(1), carry)
	binary.BigEndian.PutUint64(z[16:24], res)

	res, carry = bits.Sub64(a.getInternalUint64(2), b.getInternalUint64(2), carry)
	binary.BigEndian.PutUint64(z[8:16], res)

	res, _ = bits.Sub64(a.getInternalUint64(3), b.getInternalUint64(3), carry)
	binary.BigEndian.PutUint64(z[0:8], res)

	return z
}

func (v Value) Scale(s uint64) Value {
	sU256 := new(uint256.Int).SetUint64(s)
	return ValueFromUint256(new(uint256.Int).Mul(v.ToUint256(), sU256))
}

func (v Value) MarshalText() ([]byte, error) {
	return bytesToText(v[:])
}

func (v *Value) UnmarshalText(data []byte) error {
	return textToBytes(v[:], data)
}

func bytesToText(data []byte) ([]byte, error) {
	return []byte(fmt.Sprintf("0x%x", data)), nil
}

func textToBytes(trg []byte, data []byte) error {
	s := string(data)
	if !strings.HasPrefix(s, "0x") {
		return fmt.Errorf("invalid format, does not start with 0x: %v", s)
	}
	data, err := hex.DecodeString(s[2:])
	if err != nil {
		return err
	}
	if want, got := len(trg), len(data); want != got {
		return fmt.Errorf("invalid format, wanted %d bytes, got %d", want, got)
	}
	copy(trg[:], data)
	return nil
}

func (k CallKind) String() string {
	switch k {
	case Call:
		return "call"
	case StaticCall:
		return "static_call"
	case DelegateCall:
		return "delegate_call"
	case CallCode:
		return "call_code"
	case Create:
		return "create"
	case Create2:
		return "create2"
	default:
		return "unknown"
	}
}

func (k CallKind) MarshalJSON() ([]byte, error) {
	var res string
	switch k {
	case Call, StaticCall, DelegateCall, CallCode, Create, Create2:
		res = k.String()
	default:
		return nil, fmt.Errorf("invalid call kind: %v", k)
	}
	return json.Marshal(res)
}

func (k *CallKind) UnmarshalJSON(data []byte) error {
	var kind string
	if err := json.Unmarshal(data, &kind); err != nil {
		return err
	}
	switch strings.ToLower(kind) {
	case "call":
		*k = Call
	case "static_call":
		*k = StaticCall
	case "delegate_call":
		*k = DelegateCall
	case "call_code":
		*k = CallCode
	case "create":
		*k = Create
	case "create2":
		*k = Create2
	default:
		return fmt.Errorf("unknown call kind: %s", kind)
	}
	return nil
}

func (v Value) getInternalUint64(index int) uint64 {
	start := 24 - index*8
	end := start + 8
	return binary.BigEndian.Uint64(v[start:end])
}

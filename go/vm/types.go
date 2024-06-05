//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package vm

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
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

func (v Value) ToBig() *big.Int {
	return new(big.Int).SetBytes(v[:])
}

func (v Value) ToU256() *uint256.Int {
	return new(uint256.Int).SetBytes(v[:])
}

func (v Value) String() string {
	return fmt.Sprintf("0x%x", v[:])
}

func (v Value) Cmp(o Value) int {
	return v.ToU256().Cmp(o.ToU256())
}

func ValueFromUint64(value uint64) Value {
	return Uint256ToValue(new(uint256.Int).SetUint64(value))
}

func (v Value) Add(o Value) Value {
	a := v.ToU256()
	b := o.ToU256()
	return Uint256ToValue(new(uint256.Int).Add(a, b))
}

func (v Value) Sub(o Value) Value {
	a := v.ToU256()
	b := o.ToU256()
	return Uint256ToValue(new(uint256.Int).Sub(a, b))
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

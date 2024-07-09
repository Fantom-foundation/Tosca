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
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
)

func TestAddress_NewAddress(t *testing.T) {
	address := Address{}

	if address != [20]byte{} {
		t.Errorf("New address must be default value.")
	}
}

func TestAddress_JSON_Encoding(t *testing.T) {
	tests := []struct {
		address Address
		json    string
	}{
		{Address{}, "\"0x0000000000000000000000000000000000000000\""},
		{Address{1}, "\"0x0100000000000000000000000000000000000000\""},
		{Address{0xAB}, "\"0xab00000000000000000000000000000000000000\""},
		{
			Address{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			"\"0x000102030405060708090a0b0c0d0e0f10111213\"",
		},
	}

	for _, test := range tests {
		encoded, err := json.Marshal(test.address)
		if err != nil {
			t.Fatalf("failed to encode into JSON: %v", err)
		}

		if want, got := test.json, string(encoded); want != got {
			t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
		}

		var restored Address
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatalf("failed to restore address: %v", err)
		}
		if test.address != restored {
			t.Errorf("unexpected restored value, wanted %v, got %v", test.address, restored)
		}

	}
}

func TestAddress_JSON_InvalidValueDecodingFails(t *testing.T) {
	tests := map[string]string{
		"empty":                 "\"\"",
		"empty with hex prefix": "\"0x\"",
		"no hex prefix":         "\"0000000000000000000000000000000000000000\"",
		"too short":             "\"0x00000000000000000000000000000000000000\"",
		"just too short":        "\"0x000102030405060708090a0b0c0d0e0f1011121\"",
		"just too long":         "\"0x000102030405060708090a0b0c0d0e0f101112131\"",
		"too long":              "\"0x000000000000000000000000000000000000000000\"",
		"invalid hex":           "\"0x0g00000000000000000000000000000000000000\"",
		"not hex":               "\"hello, this is a test with 20 characters\"",
		"not a JSON string":     "0x000102030405060708090a0b0c0d0e0f10111213",
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			var address Address
			if json.Unmarshal([]byte(data), &address) == nil {
				t.Errorf("expected decoding to fail, but instead it produced %v", address)
			}
		})
	}
}

func TestValue_NewValue(t *testing.T) {

	tests := []struct {
		value Value
		index int
	}{
		{NewValue(1), 31},
		{NewValue(1, 0), 23},
		{NewValue(1, 0, 0), 15},
		{NewValue(1, 0, 0, 0), 7},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v[%d]", test.value, test.index), func(t *testing.T) {

			if test.value[test.index] != 1 {
				t.Errorf("NewValue failed to set the correct value.")
			}

		})
	}
}

func TestValue_getInternalUint64ProducesExpectedUint64Value(t *testing.T) {

	tests := []struct {
		value Value
		index int
		want  uint64
	}{
		{NewValue(1), 0, 1},
		{NewValue(1, 0), 1, 1},
		{NewValue(1, 0, 0), 2, 1},
		{NewValue(1, 0, 0, 0), 3, 1},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v[%d]", test.value, test.index), func(t *testing.T) {
			if got := test.value.getInternalUint64(test.index); test.want != got {
				t.Errorf("wanted %v but got %v.", test.want, got)
			}
		})
	}
}

func TestValue_StringProducesDecimalPrint(t *testing.T) {
	tests := []struct {
		value Value
		want  string
	}{
		{NewValue(), "0"},
		{NewValue(1), "1"},
		{NewValue(2), "2"},
		{NewValue(256), "256"},
		{ValueFromUint256(uint256.MustFromDecimal("1")), "1"},
		{ValueFromUint256(uint256.MustFromDecimal("1234567890123456789")), "1234567890123456789"},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			if want, got := test.want, test.value.String(); want != got {
				t.Errorf("unexpected string conversion, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestValue_ToBigConversion(t *testing.T) {
	tests := []struct {
		value Value
		big   *big.Int
	}{
		{Value{}, big.NewInt(0)},
		{Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, big.NewInt(1)},
		{Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}, big.NewInt(2)},
		{Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 127, 255, 255, 255, 255, 255, 255, 255}, big.NewInt(math.MaxInt64)},
		{Value{128}, new(big.Int).Lsh(big.NewInt(1), 255)},
	}

	for _, test := range tests {
		t.Run(test.big.String(), func(t *testing.T) {
			if want, got := test.big, test.value.ToBig(); want.Cmp(got) != 0 {
				t.Errorf("unexpected big.Int conversion, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestValue_ToUint256Conversion(t *testing.T) {
	tests := []struct {
		value   Value
		uint256 *uint256.Int
	}{
		{Value{}, uint256.NewInt(0)},
		{Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, uint256.NewInt(1)},
		{Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}, uint256.NewInt(2)},
		{Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 127, 255, 255, 255, 255, 255, 255, 255}, uint256.NewInt(math.MaxInt64)},
		{Value{128}, new(uint256.Int).Lsh(uint256.NewInt(1), 255)},
	}

	for _, test := range tests {
		t.Run(test.uint256.String(), func(t *testing.T) {
			if want, got := test.uint256, test.value.ToUint256(); want.Cmp(got) != 0 {
				t.Errorf("unexpected uint256.Int conversion, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestValue_FromUint256Conversion(t *testing.T) {
	tests := []struct {
		uint256 *uint256.Int
		value   Value
	}{
		{nil, NewValue(0)},
		{uint256.NewInt(0), NewValue(0)},
		{uint256.NewInt(1), NewValue(1)},
		{uint256.NewInt(2), NewValue(2)},
		{uint256.NewInt(math.MaxInt64), NewValue(math.MaxInt64)},
		{new(uint256.Int).Lsh(uint256.NewInt(1), 255), Value{128}},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.uint256), func(t *testing.T) {
			if want, got := test.value, ValueFromUint256(test.uint256); want != got {
				t.Errorf("unexpected Value conversion, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestValue_Comparison(t *testing.T) {
	values := []Value{
		{}, {1}, {2},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
	}

	for _, a := range values {
		for _, b := range values {
			want := a.ToBig().Cmp(b.ToBig())
			got := a.Cmp(b)
			if want != got {
				t.Errorf("unexpected comparison result for %v and %v, wanted %v, got %v", a, b, want, got)
			}
		}
	}
}

func TestValue_FromUint64(t *testing.T) {
	tests := []struct {
		in  uint64
		out Value
	}{
		{0, Value{}},
		{1, Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}},
		{2, Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}},
		{256, Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0}},
		{65536, Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0}},
		{math.MaxUint64, Value{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d", test.in), func(t *testing.T) {
			if want, got := test.out, NewValue(test.in); want != got {
				t.Errorf("unexpected conversion result, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestValue_Arithmetic(t *testing.T) {
	values := []Value{
		{}, {1}, {2},
		NewValue(1), NewValue(2), NewValue(3),
		NewValue(math.MaxInt64),
		NewValue(math.MaxUint64),
		{
			0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
			0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
			0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
			0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		},
	}

	for _, a := range values {
		for _, b := range values {
			want := new(uint256.Int).Add(a.ToUint256(), b.ToUint256())
			got := Add(a, b).ToUint256()
			if want.Cmp(got) != 0 {
				t.Errorf("unexpected addition result for %v and %v, wanted %v, got %v", a, b, want, got)
			}

			want = new(uint256.Int).Sub(a.ToUint256(), b.ToUint256())
			got = Sub(a, b).ToUint256()
			if want.Cmp(got) != 0 {
				t.Errorf("unexpected subtraction result for %v and %v, wanted %v, got %v", a, b, want, got)
			}
		}
	}
}

func TestValue_ArithmeticAddCarry(t *testing.T) {

	const max64 = math.MaxUint64
	tests := map[string]struct {
		x, t, want Value
	}{
		"carry to second": {NewValue(max64), NewValue(1), NewValue(1, 0)},
		"carry to third":  {NewValue(max64, max64), NewValue(0, 1), NewValue(1, 0, 0)},
		"carry to fourth": {NewValue(max64, max64, max64), NewValue(0, 0, 1), NewValue(1, 0, 0, 0)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if want, got := test.want, Add(test.x, test.t); want != got {
				t.Errorf("unexpected addition result, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestValue_ArithmeticSubCarry(t *testing.T) {
	const max64 = math.MaxUint64
	tests := map[string]struct {
		x, t, want Value
	}{
		"carry to first":  {NewValue(1, 0), NewValue(1), NewValue(max64)},
		"carry to second": {NewValue(1, 0, 0), NewValue(1), NewValue(max64, max64)},
		"carry to third":  {NewValue(1, 0, 0, 0), NewValue(1), NewValue(max64, max64, max64)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if want, got := test.want, Sub(test.x, test.t); want != got {
				t.Errorf("unexpected subtraction result, wanted %v, got %v", want, got)
			}
		})
	}
}

func TestValue_JSON_Encoding(t *testing.T) {
	tests := []struct {
		value Value
		json  string
	}{
		{Value{}, "\"0x0000000000000000000000000000000000000000000000000000000000000000\""},
		{Value{1}, "\"0x0100000000000000000000000000000000000000000000000000000000000000\""},
		{Value{0xAB}, "\"0xab00000000000000000000000000000000000000000000000000000000000000\""},
		{
			Value{
				0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
				10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
				20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
				30, 31,
			},
			"\"0x000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f\"",
		},
	}

	for _, test := range tests {
		encoded, err := json.Marshal(test.value)
		if err != nil {
			t.Fatalf("failed to encode into JSON: %v", err)
		}

		if want, got := test.json, string(encoded); want != got {
			t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
		}

		var restored Value
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatalf("failed to restore address: %v", err)
		}
		if test.value != restored {
			t.Errorf("unexpected restored value, wanted %v, got %v", test.value, restored)
		}

	}
}

func TestCallKind_JSON_Encoding(t *testing.T) {
	tests := []struct {
		kind CallKind
		json string
	}{
		{Call, "\"call\""},
		{StaticCall, "\"static_call\""},
		{DelegateCall, "\"delegate_call\""},
		{CallCode, "\"call_code\""},
		{Create, "\"create\""},
		{Create2, "\"create2\""},
	}

	for _, test := range tests {
		encoded, err := json.Marshal(test.kind)
		if err != nil {
			t.Fatalf("failed to encode into JSON: %v", err)
		}

		if want, got := test.json, string(encoded); want != got {
			t.Errorf("unexpected JSON encoding, wanted %v, got %v", want, got)
		}

		var restored CallKind
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatalf("failed to restore address: %v", err)
		}
		if test.kind != restored {
			t.Errorf("unexpected restored value, wanted %v, got %v", test.kind, restored)
		}

	}
}

func TestCallKind_JSON_InvalidValueEncodingFails(t *testing.T) {
	if _, err := json.Marshal(CallKind(99)); err == nil {
		t.Errorf("expected encoding to fail")
	}
}

func BenchmarkValue_Add(b *testing.B) {

	x := Value{1}
	y := Value{2}

	for i := 0; i < b.N; i++ {
		Add(x, y)
	}
}

// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package common

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"pgregory.net/rand"
)

func TestNewU256FromBytes_WithLessThan32Bytes(t *testing.T) {
	x := NewU256FromBytes([]byte{1, 2, 3, 4}...)
	xBytes := x.Bytes32be()
	if !bytes.Equal(xBytes[:], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4}) {
		t.Fail()
	}
}

func TestNewU256FromBytes_With32Bytes(t *testing.T) {
	x := NewU256FromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}...)
	xBytes := x.Bytes32be()
	if !bytes.Equal(xBytes[:], []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}) {
		t.Fail()
	}
}

func TestNewU256FromBytes_PanicsWithMoreThan32Bytes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fail()
		}
	}()
	_ = NewU256FromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33}...)
}

func TestNewU256FromUint256(t *testing.T) {
	tests := []struct {
		in  *uint256.Int
		out U256
	}{
		{uint256.NewInt(0), NewU256()},
		{uint256.NewInt(1), NewU256(1)},
		{uint256.NewInt(256), NewU256(256)},
		{new(uint256.Int).Lsh(uint256.NewInt(1), 64), NewU256(1, 0)},
		{new(uint256.Int).Lsh(uint256.NewInt(1), 128), NewU256(1, 0, 0)},
		{new(uint256.Int).Lsh(uint256.NewInt(1), 192), NewU256(1, 0, 0, 0)},
	}

	for _, test := range tests {
		got := NewU256FromUint256(test.in)
		want := test.out
		if want != got {
			t.Errorf("failed to convert %v to U256, wanted %v, got %v", test.in, want, got)
		}
	}
}

func TestU256IsZero(t *testing.T) {
	zero := NewU256()
	if !zero.IsZero() {
		t.Fail()
	}
	one := NewU256(1)
	if one.IsZero() {
		t.Fail()
	}
}

func TestU256Bytes32be(t *testing.T) {
	x := NewU256(1, 2, 3, 4)
	xBytes := x.Bytes32be()
	if !bytes.Equal(xBytes[:], []byte{0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4}) {
		t.Fail()
	}
}

func TestU256Bytes20be(t *testing.T) {
	x := NewU256(1, 2, 3, 4)
	xBytes := x.Bytes20be()
	if !bytes.Equal(xBytes[:], []byte{0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4}) {
		t.Fail()
	}
}

func TestU256Eq(t *testing.T) {
	a := NewU256(1, 2, 3, 4)
	b := NewU256(0, 0, 0, 4)
	if !a.Eq(a) {
		t.Fail()
	}
	if a.Eq(b) {
		t.Fail()
	}
}

func TestU256Ne(t *testing.T) {
	a := NewU256(1, 2, 3, 4)
	b := NewU256(0, 0, 0, 4)
	if a.Ne(a) {
		t.Fail()
	}
	if !a.Ne(b) {
		t.Fail()
	}
}

func TestU256Lt(t *testing.T) {
	a := NewU256(1, 2, 3, 4)
	b := NewU256(0, 0, 0, 4)
	if a.Lt(a) {
		t.Fail()
	}
	if a.Lt(b) {
		t.Fail()
	}
	if !b.Lt(a) {
		t.Fail()
	}
}

func TestU256Slt(t *testing.T) {
	if !MaxU256().Slt(NewU256(0)) {
		t.Fail()
	}
}

func TestU256Gt(t *testing.T) {
	a := NewU256(1, 2, 3, 4)
	b := NewU256(0, 0, 0, 4)
	if a.Gt(a) {
		t.Fail()
	}
	if !a.Gt(b) {
		t.Fail()
	}
	if b.Gt(a) {
		t.Fail()
	}
}

func TestU256Sgt(t *testing.T) {
	zero := NewU256(0)
	if !zero.Sgt(MaxU256()) {
		t.Fail()
	}
}

func TestU256Add(t *testing.T) {
	a := NewU256(17)
	b := NewU256(13)
	if a.Add(b).Ne(NewU256(17 + 13)) {
		t.Fail()
	}
	if MaxU256().Add(NewU256(1)).Ne(NewU256(0)) {
		t.Fail()
	}
}

func TestU256AddMod(t *testing.T) {
	a := NewU256(10)
	if a.AddMod(NewU256(10), NewU256(8)).Ne(NewU256(4)) {
		t.Fail()
	}
	if MaxU256().AddMod(NewU256(2), NewU256(2)).Ne(NewU256(1)) {
		t.Fail()
	}
}

func TestU256Sub(t *testing.T) {
	a := NewU256(17)
	b := NewU256(13)
	if a.Sub(b).Ne(NewU256(17 - 13)) {
		t.Fail()
	}
	zero := NewU256(0)
	if zero.Sub(NewU256(1)).Ne(MaxU256()) {
		t.Fail()
	}
}

func TestU256Mul(t *testing.T) {
	a := NewU256(17)
	b := NewU256(13)
	if a.Mul(b).Ne(NewU256(17 * 13)) {
		t.Fail()
	}
}

func TestU256MulMod(t *testing.T) {
	a := NewU256(10)
	if a.MulMod(NewU256(10), NewU256(8)).Ne(NewU256(4)) {
		t.Fail()
	}
	if MaxU256().MulMod(MaxU256(), NewU256(12)).Ne(NewU256(9)) {
		t.Fail()
	}
}

func TestU256Div(t *testing.T) {
	a := NewU256(24)
	b := NewU256(8)
	if a.Div(b).Ne(NewU256(24 / 8)) {
		t.Fail()
	}
}

func TestU256Mod(t *testing.T) {
	a := NewU256(25)
	b := NewU256(8)
	if a.Mod(b).Ne(NewU256(25 % 8)) {
		t.Fail()
	}
}

func TestU256SDiv(t *testing.T) {
	a := MaxU256().Sub(NewU256(1))
	b := MaxU256()
	if a.SDiv(b).Ne(NewU256(2)) {
		t.Fail()
	}
}

func TestU256SMod(t *testing.T) {
	a := MaxU256().Sub(NewU256(7))
	b := MaxU256().Sub(NewU256(2))
	if a.SMod(b).Ne(MaxU256().Sub(NewU256(1))) {
		t.Fail()
	}
}

func TestU256Exp(t *testing.T) {
	a := NewU256(7)
	b := NewU256(5)
	if a.Exp(b).Ne(NewU256(16807)) {
		t.Fail()
	}
}

func TestU256Not(t *testing.T) {
	zero := NewU256(0)
	if zero.Not().Ne(MaxU256()) {
		t.Fail()
	}
}

func TestU256Shl(t *testing.T) {
	x := NewU256(42)
	if x.Shl(NewU256(64)).Ne(NewU256(42, 0)) {
		t.Fail()
	}
}
func TestU256Shr(t *testing.T) {
	x := NewU256(42, 0)
	if x.Shr(NewU256(64)).Ne(NewU256(42)) {
		t.Fail()
	}
}

func TestU256String(t *testing.T) {
	tests := []struct {
		value U256
		print string
	}{
		{U256{}, "0000000000000000 0000000000000000 0000000000000000 0000000000000000"},
		{NewU256(0), "0000000000000000 0000000000000000 0000000000000000 0000000000000000"},
		{NewU256(1), "0000000000000000 0000000000000000 0000000000000000 0000000000000001"},
		{NewU256(2), "0000000000000000 0000000000000000 0000000000000000 0000000000000002"},
		{NewU256(1, 2), "0000000000000000 0000000000000000 0000000000000001 0000000000000002"},
		{NewU256(1, 2, 3), "0000000000000000 0000000000000001 0000000000000002 0000000000000003"},
		{NewU256(42, 13, 47, 1), "000000000000002a 000000000000000d 000000000000002f 0000000000000001"},
	}

	for _, test := range tests {
		if want, got := test.print, test.value.String(); want != got {
			t.Errorf("Unexpected print, wanted %s, got %s", want, got)
		}
	}
}

func TestU256ToBigInt(t *testing.T) {
	tooBigInt := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	bigExp := new(big.Int).Sub(tooBigInt, big.NewInt(1))
	testCases := map[string]struct {
		input U256
		want  big.Int
	}{
		"regular": {NewU256(123456789), *big.NewInt(123456789)},
		"maxU256": {MaxU256(), *bigExp},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := tc.input.ToBigInt()
			if tc.want.Cmp(got) != 0 {
				t.Fatalf("Unexpected value after conversion from U256 to big.Int, want %v, got %v", tc.want, got)
			}

		})
	}
}

func TestNewU256FromBigInt(t *testing.T) {
	want := NewU256(123456789)
	got := NewU256FromBigInt(big.NewInt(123456789))
	if !want.Eq(got) {
		t.Fatalf("Unexpected value after conversion from big int to U256: want %v, got %v", want, got)
	}
}

func TestNewU256FromBigInt_PanicsWithInvalidInput(t *testing.T) {
	tooBigInt := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil)
	testCases := map[string]struct {
		input *big.Int
		want  U256
	}{
		"negative": {big.NewInt(-1), NewU256(0)},
		"overflow": {tooBigInt, NewU256(0)},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic when converting big int to U256 with value %v", tc.input)
				}
			}()
			_ = NewU256FromBigInt(tc.input)
		})
	}
}

func TestU256_Marshalling(t *testing.T) {
	tests := []struct {
		value      U256
		marshalled []byte
	}{
		{U256{}, []byte("0000000000000000 0000000000000000 0000000000000000 0000000000000000")},
		{NewU256(0), []byte("0000000000000000 0000000000000000 0000000000000000 0000000000000000")},
		{NewU256(1), []byte("0000000000000000 0000000000000000 0000000000000000 0000000000000001")},
		{NewU256(2), []byte("0000000000000000 0000000000000000 0000000000000000 0000000000000002")},
		{NewU256(1, 2), []byte("0000000000000000 0000000000000000 0000000000000001 0000000000000002")},
		{NewU256(1, 2, 3), []byte("0000000000000000 0000000000000001 0000000000000002 0000000000000003")},
		{NewU256(42, 13, 47, 1), []byte("000000000000002a 000000000000000d 000000000000002f 0000000000000001")},
		{NewU256(0xa000000000000000, 0xb000000000000000, 0xc000000000000000, 0xd000000000000000), []byte("a000000000000000 b000000000000000 c000000000000000 d000000000000000")},
		{NewU256(0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff), []byte("ffffffffffffffff ffffffffffffffff ffffffffffffffff ffffffffffffffff")},
	}

	for _, test := range tests {
		marshalled, err := test.value.MarshalText()
		if err != nil {
			t.Fatalf("Unexpected error when marshalling U256: %v", err)
		}
		if !bytes.Equal(marshalled, test.marshalled) {
			t.Errorf("Unexpected marshalled value: want %v, got %v", test.marshalled, marshalled)
		}
	}
}

func TestU256_Unmarshalling(t *testing.T) {
	tests := []struct {
		marshalled []byte
		want       U256
	}{
		{[]byte("0000000000000000 0000000000000000 0000000000000000 0000000000000000"), U256{}},
		{[]byte("0000000000000000 0000000000000000 0000000000000000 0000000000000001"), NewU256(1)},
		{[]byte("0000000000000000 0000000000000000 0000000000000000 0000000000000002"), NewU256(2)},
		{[]byte("0000000000000000 0000000000000000 0000000000000001 0000000000000002"), NewU256(1, 2)},
		{[]byte("0000000000000000 0000000000000001 0000000000000002 0000000000000003"), NewU256(1, 2, 3)},
		{[]byte("000000000000002a 000000000000000d 000000000000002f 0000000000000001"), NewU256(42, 13, 47, 1)},
		{[]byte("a000000000000000 B000000000000000 C000000000000000 d000000000000000"), NewU256(0xa000000000000000, 0xb000000000000000, 0xc000000000000000, 0xd000000000000000)},
		{[]byte("ffffffffffffffff ffffffffffffffff ffffffffffffffff ffffffffffffffff"), NewU256(0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff)},
	}

	for _, test := range tests {
		var u U256
		err := u.UnmarshalText(test.marshalled)
		if err != nil {
			t.Fatalf("Unexpected error when unmarshalling U256: %v", err)
		}
		if !u.Eq(test.want) {
			t.Errorf("Unexpected unmarshalled value: want %v, got %v", test.want, u)
		}
	}
}

func TestU256_UnmarshallingError(t *testing.T) {
	testCases := map[string][]byte{
		"first value too short":  []byte("000000000000000 0000000000000001 0000000000000002 0000000000000003"),
		"second value too short": []byte("0000000000000004 000000000000000 0000000000000005 0000000000000006"),
		"third value too short":  []byte("0000000000000007 0000000000000008 000000000000000 0000000000000009"),
		"fourth value too short": []byte("000000000000000a 000000000000000b 000000000000000c 000000000000000"),
		"one value missing":      []byte("000000000000000d 000000000000000e 000000000000000f"),
		"two values missing":     []byte("1000000000000000 2000000000000000"),
		"three values missing":   []byte("3000000000000000"),
		"four values missing":    []byte(""),
		"first value invalid":    []byte("000000000000000g 4000000000000000 5000000000000000 6000000000000000"),
		"second value invalid":   []byte("7000000000000000 000000000000000g 8000000000000000 9000000000000000"),
		"third value invalid":    []byte("a000000000000000 b000000000000000 000000000000000g c000000000000000"),
		"fourth value invalid":   []byte("d000000000000000 e000000000000000 f000000000000000 000000000000000g"),
		"first value too long":   []byte("00000000000000000 0000000000000000 0000000000000000 0000000000000000"),
		"second value too long":  []byte("0000000000000000 00000000000000000 0000000000000000 0000000000000000"),
		"third value too long":   []byte("0000000000000000 0000000000000000 00000000000000000 0000000000000000"),
		"fourth value too long":  []byte("0000000000000000 0000000000000000 0000000000000000 00000000000000000"),
		"one value too many":     []byte("0000000000000000 0000000000000000 0000000000000000 0000000000000000 0000000000000000"),
		"leading space":          []byte(" 0000000000000000 0000000000000000 0000000000000000 0000000000000000"),
		"trailing space":         []byte("0000000000000000 0000000000000000 0000000000000000 0000000000000000 "),
		"more than one space":    []byte("0000000000000000  0000000000000000 0000000000000000 0000000000000000"),
		"tab separated":          []byte("0000000000000000\t0000000000000000\t0000000000000000\t0000000000000000"),
		"newline separated":      []byte("0000000000000000\n0000000000000000\n0000000000000000\n0000000000000000"),
		"no separator":           []byte("0000000000000000000000000000000000000000000000000000000000000000"),
	}

	for name, input := range testCases {
		t.Run(name, func(t *testing.T) {
			var u U256
			err := u.UnmarshalText(input)
			if err == nil {
				t.Fatalf("Expected error when unmarshalling input with: %s", name)
			}
		})
	}
}

func TestU256_MarshallingRoundTrip(t *testing.T) {
	tests := []struct {
		value U256
	}{
		{U256{}},
		{NewU256(1)},
		{NewU256(2)},
		{NewU256(1, 2)},
		{NewU256(1, 2, 3)},
		{NewU256(42, 13, 47, 1)},
		{NewU256(0xa000000000000000, 0xb000000000000000, 0xc000000000000000, 0xd000000000000000)},
		{NewU256(0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff)},
	}

	for _, test := range tests {
		marshalled, err := test.value.MarshalText()
		if err != nil {
			t.Fatalf("Unexpected error when marshalling U256: %v", err)
		}

		var u U256
		err = u.UnmarshalText(marshalled)
		if err != nil {
			t.Fatalf("Unexpected error when unmarshalling U256: %v", err)
		}
		if !u.Eq(test.value) {
			t.Errorf("Unexpected unmarshalled value: want %v, got %v", test.value, u)
		}
	}
}

func TestU256_RandU256(t *testing.T) {
	rands := []U256{}
	rnd := rand.New()
	for i := 0; i < 10; i++ {
		rands = append(rands, RandU256(rnd))
		for j := 0; j < i; j++ {
			if rands[i] == rands[j] {
				t.Errorf("random U256 are not random, got %v and %v", rands[i], rands[j])
			}
		}
	}
}

func TestU256_RandU256Between(t *testing.T) {
	rnd := rand.New()
	for min := uint64(0); min < 10; min++ {
		for max := min; max < 10; max++ {
			for i := 0; i < 100; i++ {
				r := RandU256Between(rnd, NewU256(min), NewU256(max))
				if r.Uint64() < min || r.Uint64() > max {
					t.Errorf("random U256 is not between %d and %d, got %v", min, max, r)
				}
			}
		}
	}
}

func TestU256_IsUint64(t *testing.T) {
	tests := []struct {
		value U256
		want  bool
	}{
		{U256{}, true},
		{NewU256(1), true},
		{NewU256(1, 2), false},
		{NewU256(1, 2, 3), false},
		{NewU256(42, 13, 47, 1), false},
		{NewU256(0xa000000000000000, 0xb000000000000000, 0xc000000000000000, 0xd000000000000000), false},
		{NewU256(0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff), false},
	}

	for _, test := range tests {
		if want, got := test.want, test.value.IsUint64(); want != got {
			t.Errorf("Unexpected result from IsUint64, want %v, got %v", want, got)
		}
	}
}

func TestU256_Uint64(t *testing.T) {
	tests := []struct {
		value U256
		want  uint64
	}{
		{U256{}, 0},
		{NewU256(1), 1},
		{NewU256(0xffffffffffffffff), 0xffffffffffffffff},
	}

	for _, test := range tests {
		if want, got := test.want, test.value.Uint64(); want != got {
			t.Errorf("Unexpected result from Uint64, want %v, got %v", want, got)
		}
	}
}

func TestU256_Uint256Conversion(t *testing.T) {
	tests := []struct {
		original  *uint256.Int
		converted U256
	}{
		{uint256.NewInt(0), U256{}},
		{uint256.NewInt(1), NewU256(1)},
		{uint256.NewInt(256), NewU256(256)},
		{new(uint256.Int).Lsh(uint256.NewInt(1), 64), NewU256(1, 0)},
	}

	for _, test := range tests {
		if want, got := test.original, test.converted.Uint256(); !want.Eq(&got) {
			t.Errorf("Unexpected result from Uint256, want %v, got %v", want, got)
		}
		if want, got := test.converted, NewU256FromUint256(test.original); !want.Eq(got) {
			t.Errorf("Unexpected result from NewU256FromUint256, want %v, got %v", want, got)
		}
	}
}

func TestU256_SignExtend(t *testing.T) {

	maxu64 := uint64(0xffffffffffffffff)

	tests := []struct {
		value    U256
		argument U256
		want     U256
	}{
		{U256{}, NewU256(1), U256{}},
		{U256{}, NewU256(0xfff), U256{}},
		{NewU256(1), NewU256(1), NewU256(1)},
		{NewU256(1), NewU256(0xfff), NewU256(1)},
		{NewU256(0xff), NewU256(1), NewU256(0xff)},
		{NewU256(0xff), NewU256(0xfff), NewU256(0xff)},
		{NewU256(maxu64), NewU256(1), NewU256(maxu64, maxu64, maxu64, maxu64)},
		{NewU256(maxu64), NewU256(0xfff), NewU256(maxu64)},
		{NewU256(maxu64, maxu64, maxu64, maxu64), NewU256(1), NewU256(maxu64, maxu64, maxu64, maxu64)},
		{NewU256(maxu64, maxu64, maxu64, maxu64), NewU256(0xfff), NewU256(maxu64, maxu64, maxu64, maxu64)},
	}

	for _, test := range tests {
		t.Run(test.value.String(), func(t *testing.T) {
			if want, got := test.want, test.value.SignExtend(test.argument); !want.Eq(got) {
				t.Errorf("Unexpected result from SignExtend, want %v, got %v", want, got)
			}
		})
	}
}

func TestU256_And(t *testing.T) {
	maxu64 := uint64(0xffffffffffffffff)
	tests := []struct {
		a    U256
		b    U256
		want U256
	}{
		{U256{}, U256{}, U256{}},
		{NewU256(1), NewU256(1), NewU256(1)},
		{NewU256(0xff), NewU256(0xff), NewU256(0xff)},
		{NewU256(maxu64), NewU256(maxu64), NewU256(maxu64)},
		{NewU256(0x12345678), NewU256(0x87654321), NewU256(0x12345678 & 0x87654321)},
		{
			NewU256(0xffff00, 0xffff00, 0xffff00, 0xffff00),
			NewU256(0x00ffff, 0x00ffff, 0x00ffff, 0x00ffff),
			NewU256(0x00ff00, 0x00ff00, 0x00ff00, 0x00ff00),
		},
		{
			NewU256(maxu64, maxu64, maxu64, maxu64),
			NewU256(maxu64, maxu64, maxu64, maxu64),
			NewU256(maxu64, maxu64, maxu64, maxu64),
		},
		{
			NewU256(0, maxu64, 0, maxu64),
			NewU256(0, 0, maxu64, maxu64),
			NewU256(0, 0, 0, maxu64),
		},
	}

	for _, test := range tests {
		if want, got := test.want, test.a.And(test.b); !want.Eq(got) {
			t.Errorf("Unexpected result from And, want %v, got %v", want, got)
		}
	}
}

func TestU256_Or(t *testing.T) {
	maxu64 := uint64(0xffffffffffffffff)
	tests := []struct {
		a    U256
		b    U256
		want U256
	}{
		{U256{}, U256{}, U256{}},
		{NewU256(1), NewU256(), NewU256(1)},
		{NewU256(0xff), NewU256(0xff), NewU256(0xff)},
		{NewU256(maxu64), NewU256(0), NewU256(maxu64)},
		{NewU256(0x12345678), NewU256(0x87654321), NewU256(0x12345678 | 0x87654321)},
		{
			NewU256(0, maxu64, 0, maxu64),
			NewU256(0, 0, maxu64, maxu64),
			NewU256(0, maxu64, maxu64, maxu64),
		},
	}

	for _, test := range tests {
		if want, got := test.want, test.a.Or(test.b); !want.Eq(got) {
			t.Errorf("Unexpected result from Or, want %v, got %v", want, got)
		}
	}
}

func TestU256_Xor(t *testing.T) {
	maxu64 := uint64(0xffffffffffffffff)
	tests := []struct {
		a    U256
		b    U256
		want U256
	}{
		{U256{}, U256{}, U256{}},
		{NewU256(1), NewU256(0), NewU256(1)},
		{NewU256(0), NewU256(0xff), NewU256(0xff)},
		{NewU256(maxu64), NewU256(maxu64), U256{}},
		{NewU256(0x12345678), NewU256(0x87654321), NewU256(0x12345678 ^ 0x87654321)},
		{
			NewU256(0, maxu64, 0, maxu64),
			NewU256(0, 0, maxu64, maxu64),
			NewU256(0, maxu64, maxu64, 0),
		},
	}

	for _, test := range tests {
		if want, got := test.want, test.a.Xor(test.b); !want.Eq(got) {
			t.Errorf("Unexpected result from Xor, want %v, got %v", want, got)
		}
	}
}

func TestU256_Srsh(t *testing.T) {
	tests := []struct {
		a    U256
		b    U256
		want U256
	}{
		{NewU256(0x12345678, 0x9abcdef0, 0x12345678, 0x9abcdef0), NewU256(0), NewU256(0x12345678, 0x9abcdef0, 0x12345678, 0x9abcdef0)},
		{NewU256(0x12345678, 0x9abcdef0, 0x12345678, 0x9abcdef0), NewU256(47), NewU256(0, 0x2468acf00000, 0x13579bde00000, 0x2468acf00000)},
		{NewU256(0x12345678, 0x9abcdef0, 0x12345678, 0x9abcdef0), NewU256(64), NewU256(0, 0x12345678, 0x9abcdef0, 0x12345678)},
		{NewU256(0x12345678, 0x9abcdef0, 0x12345678, 0x9abcdef0), NewU256(128), NewU256(0, 0, 0x12345678, 0x9abcdef0)},
		{NewU256(0x12345678, 0x9abcdef0, 0x12345678, 0x9abcdef0), NewU256(256), NewU256(0, 0, 0, 0)},
		{NewU256(0xAAAAAAAA, 0x55555555, 0xAAAAAAAA, 0x55555555), NewU256(1), NewU256(0x55555555, 0x2AAAAAAA, 0x8000000055555555, 0x2AAAAAAA)},
	}

	for _, test := range tests {
		if want, got := test.want, test.a.Srsh(test.b); !want.Eq(got) {
			t.Errorf("Unexpected result from Srsh, want %v, got %v", want, got)
		}
	}
}

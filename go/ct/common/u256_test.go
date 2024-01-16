package common

import (
	"bytes"
	"math/big"
	"testing"
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

func TestU256ToBig(t *testing.T) {
	u256 := NewU256(123456789)
	bigInt := u256.ToBig()
	if want, got := big.NewInt(123456789), bigInt; want.Cmp(got) != 0 {
		t.Errorf("Failed conversion from U256 to bigInt, want: %v, got: %v", want, got)
	}

	maxU256 := MaxU256()
	bigInt = maxU256.ToBig()
	if want, got := new(big.Int).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1)),
		bigInt; want.Cmp(got) != 0 {
		t.Errorf("Failed conversion from maxU256 to bigInt, want: %v, got: %v", want, got)
	}

}

func TestU256U256FromBig(t *testing.T) {
	negBigInt := big.NewInt(-1)
	u256 := U256FromBig(negBigInt)
	if want, got := NewU256(0), u256; !want.Eq(*got) {
		t.Errorf("Failed to convert negative bigInt to U256, want: %v, got: %v", want, got)
	}

	bigInt := big.NewInt(123456789)
	if want, got := NewU256(123456789), U256FromBig(bigInt); !want.Eq(*got) {
		t.Errorf("Failed conversion from big int to U256, want %v, got %v", want, got)
	}

	tooBigInt := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	defer func() {
		r := recover()
		if r == nil || (r != nil && r != "big.Int has more than 256-bits.") {
			t.Error("Failed to panic on conversion overflow.")
		}
	}()
	notEnoughU256 := U256FromBig(tooBigInt)

	if want, got := U256FromBig(new(big.Int).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))),
		notEnoughU256; want.Eq(*got) {
		t.Errorf("Unexpected bigInt to u256 conversion, want: %v, got: %v", want, got)
	}

}

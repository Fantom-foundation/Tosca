package ct

import (
	"bytes"
	"testing"
)

func TestU256New(t *testing.T) {
	x := U256{42}
	if x[0] != 42 || x[1] != 0 || x[2] != 0 || x[3] != 0 {
		t.Fail()
	}
}

func TestU256IsZero(t *testing.T) {
	zero := U256{0}
	if !zero.IsZero() {
		t.Fail()
	}
	one := U256{1}
	if one.IsZero() {
		t.Fail()
	}
}

func TestU256Bytes32be(t *testing.T) {
	x := U256{1, 2, 3, 4}
	xBytes := x.Bytes32be()
	if !bytes.Equal(xBytes[:], []byte{0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 1}) {
		t.Fail()
	}
}

func TestU256Bytes20be(t *testing.T) {
	x := U256{1, 2, 3, 4}
	xBytes := x.Bytes20be()
	if !bytes.Equal(xBytes[:], []byte{0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 1}) {
		t.Fail()
	}
}

func TestU256Eq(t *testing.T) {
	a := U256{1, 2, 3, 4}
	b := U256{0, 0, 0, 4}
	if !a.Eq(a) {
		t.Fail()
	}
	if a.Eq(b) {
		t.Fail()
	}
}

func TestU256Ne(t *testing.T) {
	a := U256{1, 2, 3, 4}
	b := U256{0, 0, 0, 4}
	if a.Ne(a) {
		t.Fail()
	}
	if !a.Ne(b) {
		t.Fail()
	}
}

func TestU256Lt(t *testing.T) {
	a := U256{1, 2, 3, 4}
	b := U256{0, 0, 0, 4}
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

func TestU256Le(t *testing.T) {
	a := U256{1, 2, 3, 4}
	b := U256{0, 0, 0, 4}
	if !a.Le(a) {
		t.Fail()
	}
	if a.Le(b) {
		t.Fail()
	}
	if !b.Le(a) {
		t.Fail()
	}
}

func TestU256Slt(t *testing.T) {
	if !MaxU256().Slt(U256{0}) {
		t.Fail()
	}
}

func TestU256Gt(t *testing.T) {
	a := U256{1, 2, 3, 4}
	b := U256{0, 0, 0, 4}
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

func TestU256Ge(t *testing.T) {
	a := U256{1, 2, 3, 4}
	b := U256{0, 0, 0, 4}
	if !a.Ge(a) {
		t.Fail()
	}
	if !a.Ge(b) {
		t.Fail()
	}
	if b.Ge(a) {
		t.Fail()
	}
}

func TestU256Sgt(t *testing.T) {
	zero := U256{0}
	if !zero.Sgt(MaxU256()) {
		t.Fail()
	}
}

func TestU256Add(t *testing.T) {
	a := U256{17}
	b := U256{13}
	if a.Add(b).Ne(U256{17 + 13}) {
		t.Fail()
	}
	if MaxU256().Add(U256{1}).Ne(U256{0}) {
		t.Fail()
	}
}

func TestU256AddMod(t *testing.T) {
	a := U256{10}
	if a.AddMod(U256{10}, U256{8}).Ne(U256{4}) {
		t.Fail()
	}
	if MaxU256().AddMod(U256{2}, U256{2}).Ne(U256{1}) {
		t.Fail()
	}
}

func TestU256Sub(t *testing.T) {
	a := U256{17}
	b := U256{13}
	if a.Sub(b).Ne(U256{17 - 13}) {
		t.Fail()
	}
	zero := U256{0}
	if zero.Sub(U256{1}).Ne(MaxU256()) {
		t.Fail()
	}
}

func TestU256Mul(t *testing.T) {
	a := U256{17}
	b := U256{13}
	if a.Mul(b).Ne(U256{17 * 13}) {
		t.Fail()
	}
}

func TestU256MulMod(t *testing.T) {
	a := U256{10}
	if a.MulMod(U256{10}, U256{8}).Ne(U256{4}) {
		t.Fail()
	}
	if MaxU256().MulMod(MaxU256(), U256{12}).Ne(U256{9}) {
		t.Fail()
	}
}

func TestU256Div(t *testing.T) {
	a := U256{24}
	b := U256{8}
	if a.Div(b).Ne(U256{24 / 8}) {
		t.Fail()
	}
}

func TestU256Mod(t *testing.T) {
	a := U256{25}
	b := U256{8}
	if a.Mod(b).Ne(U256{25 % 8}) {
		t.Fail()
	}
}

func TestU256SDiv(t *testing.T) {
	a := MaxU256().Sub(U256{1})
	b := MaxU256()
	if a.SDiv(b).Ne(U256{2}) {
		t.Fail()
	}
}

func TestU256SMod(t *testing.T) {
	a := MaxU256().Sub(U256{7})
	b := MaxU256().Sub(U256{2})
	if a.SMod(b).Ne(MaxU256().Sub(U256{1})) {
		t.Fail()
	}
}

func TestU256Exp(t *testing.T) {
	a := U256{7}
	b := U256{5}
	if a.Exp(b).Ne(U256{16807}) {
		t.Fail()
	}
}

func TestU256Not(t *testing.T) {
	zero := U256{0}
	if zero.Not().Ne(MaxU256()) {
		t.Fail()
	}
}

func TestU256Lsh(t *testing.T) {
	x := U256{42}
	if x.Lsh(64).Ne(U256{0, 42}) {
		t.Fail()
	}
}
func TestU256Rsh(t *testing.T) {
	x := U256{0, 42}
	if x.Rsh(64).Ne(U256{42}) {
		t.Fail()
	}
}

package common

import (
	"fmt"
	"math/big"

	"pgregory.net/rand"

	"github.com/holiman/uint256"
)

// U256 is a 256-bit integer type. Contrary to holiman/uint256.Int the API
// operates on values rather than pointers.
type U256 struct {
	internal uint256.Int
}

// NewU256 creates a new U256 instance from up to 4 uint64 arguments. The
// arguments are given in the order from most significant to least significant
// by padding leading zeros as needed. No argument results in a value of zero.
func NewU256(args ...uint64) (result U256) {
	if len(args) > 4 {
		panic("Too many arguments")
	}
	offset := 4 - len(args)
	for i := 0; i < len(args) && i < len(result.internal); i++ {
		result.internal[3-i-offset] = args[i]
	}
	return
}

// NewU256FromBytes creates a new U256 instance from up to 32 byte arguments.
// The arguments are given in the order from most significant to least
// significant by padding leading zeros as needed. No argument results in a
// value of zero.
func NewU256FromBytes(bytes ...byte) (result U256) {
	if len(bytes) > 32 {
		panic("Too many arguments")
	}
	result.internal.SetBytes(bytes)
	return
}

func RandU256(rnd *rand.Rand) U256 {
	var value U256
	value.internal[0] = rnd.Uint64()
	value.internal[1] = rnd.Uint64()
	value.internal[2] = rnd.Uint64()
	value.internal[3] = rnd.Uint64()
	return value
}

func MaxU256() (result U256) {
	result.internal.SetAllOne()
	return
}

func (i U256) IsZero() bool {
	return i.internal.IsZero()
}

func (i U256) IsUint64() bool {
	return i.internal.IsUint64()
}

func (i U256) Uint64() uint64 {
	return i.internal.Uint64()
}

func (i U256) Uint256() uint256.Int {
	return i.internal
}

func (i U256) Bytes32be() [32]byte {
	return i.internal.Bytes32()
}

func (i U256) Bytes20be() [20]byte {
	return i.internal.Bytes20()
}

func (a U256) Eq(b U256) bool {
	return a.internal.Eq(&b.internal)
}

func (a U256) Ne(b U256) bool {
	return !a.internal.Eq(&b.internal)
}

func (a U256) Lt(b U256) bool {
	return a.internal.Lt(&b.internal)
}

func (a U256) Slt(b U256) bool {
	return a.internal.Slt(&b.internal)
}

func (a U256) Gt(b U256) bool {
	return a.internal.Gt(&b.internal)
}

func (a U256) Sgt(b U256) bool {
	return a.internal.Sgt(&b.internal)
}

func (a U256) Add(b U256) (z U256) {
	z.internal.Add(&a.internal, &b.internal)
	return
}

func (a U256) AddMod(b, m U256) (z U256) {
	z.internal.AddMod(&a.internal, &b.internal, &m.internal)
	return
}

func (a U256) Sub(b U256) (z U256) {
	z.internal.Sub(&a.internal, &b.internal)
	return
}

func (a U256) Mul(b U256) (z U256) {
	z.internal.Mul(&a.internal, &b.internal)
	return
}

func (a U256) MulMod(b, m U256) (z U256) {
	z.internal.MulMod(&a.internal, &b.internal, &m.internal)
	return
}

func (a U256) Div(b U256) (z U256) {
	z.internal.Div(&a.internal, &b.internal)
	return
}

func (a U256) SDiv(b U256) (z U256) {
	z.internal.SDiv(&a.internal, &b.internal)
	return
}

func (a U256) Mod(b U256) (z U256) {
	z.internal.Mod(&a.internal, &b.internal)
	return
}

func (a U256) SMod(b U256) (z U256) {
	z.internal.SMod(&a.internal, &b.internal)
	return
}

func (a U256) Exp(b U256) (z U256) {
	z.internal.Exp(&a.internal, &b.internal)
	return
}

func (a U256) SignExtend(b U256) (z U256) {
	z.internal.ExtendSign(&a.internal, &b.internal)
	return
}

func (a U256) And(b U256) (z U256) {
	z.internal.And(&a.internal, &b.internal)
	return
}

func (a U256) Or(b U256) (z U256) {
	z.internal.Or(&a.internal, &b.internal)
	return
}

func (a U256) Xor(b U256) (z U256) {
	z.internal.Xor(&a.internal, &b.internal)
	return
}

func (a U256) Not() (z U256) {
	z.internal.Not(&a.internal)
	return
}

func (a U256) Shl(b U256) (z U256) {
	if b.internal.LtUint64(256) {
		z.internal.Lsh(&a.internal, uint(b.internal.Uint64()))
	}
	return
}

func (a U256) Shr(b U256) (z U256) {
	if b.internal.LtUint64(256) {
		z.internal.Rsh(&a.internal, uint(b.internal.Uint64()))
	}
	return
}

func (a U256) Srsh(b U256) (z U256) {
	if b.internal.GtUint64(256) {
		if a.internal.IsZero() || a.internal.Sign() >= 0 {
			return NewU256(0)
		}
		return MaxU256()
	}
	z.internal.SRsh(&a.internal, uint(b.internal.Uint64()))
	return
}

func (i U256) String() string {
	return fmt.Sprintf("%016x %016x %016x %016x", i.internal[3], i.internal[2], i.internal[1], i.internal[0])
}

// ToBig returns a bigInt version of i
func (i U256) ToBig() *big.Int {
	return i.internal.ToBig()
}

// U256FromBig returns a U256 version of b.
// This conversion can panic on overflow conversion.
func U256FromBig(b *big.Int) *U256 {
	ret := NewU256()
	// if big int contains a negative number, we return zero
	if b.Cmp(big.NewInt(0)) == -1 {
		return &ret
	}
	newInternal, overflow := uint256.FromBig(b)
	if overflow {
		// since EVMs handle at most 256-bit values, this case should never happen.
		// IF it ever did, it would most certainly be an error, and execution halted.
		panic("big.Int has more than 256-bits.")
	}
	ret.internal = *newInternal
	return &ret
}

package ct

import (
	"encoding/binary"
	"math"
	"math/bits"
)

// Implementation based on github.com/holiman/uint256
type U256 [4]uint64

func MaxU256() U256 {
	return U256{math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64}
}

func (u U256) IsZero() bool {
	return u[0] == 0 && u[1] == 0 && u[2] == 0 && u[3] == 0
}

func (u U256) IsUint64() bool {
	return (u[1] | u[2] | u[3]) == 0
}

func (u U256) Bytes32be() [32]byte {
	var b [32]byte
	binary.BigEndian.PutUint64(b[0:8], u[3])
	binary.BigEndian.PutUint64(b[8:16], u[2])
	binary.BigEndian.PutUint64(b[16:24], u[1])
	binary.BigEndian.PutUint64(b[24:32], u[0])
	return b
}

func (u U256) Bytes20be() [20]byte {
	var b [20]byte
	binary.BigEndian.PutUint32(b[0:4], uint32(u[2]))
	binary.BigEndian.PutUint64(b[4:12], u[1])
	binary.BigEndian.PutUint64(b[12:20], u[0])
	return b
}

func (a U256) Eq(b U256) bool {
	return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3]
}

func (a U256) Ne(b U256) bool {
	return !a.Eq(b)
}

func (a U256) Cmp(b U256) int {
	if a.Gt(b) {
		return 1
	}
	if a.Lt(b) {
		return -1
	}
	return 0
}

func (a U256) Lt(b U256) bool {
	_, carry := bits.Sub64(a[0], b[0], 0)
	_, carry = bits.Sub64(a[1], b[1], carry)
	_, carry = bits.Sub64(a[2], b[2], carry)
	_, carry = bits.Sub64(a[3], b[3], carry)
	return carry != 0
}

func (a U256) Le(b U256) bool {
	return a.Eq(b) || a.Lt(b)
}

func (a U256) Slt(b U256) bool {
	aSign := a.Sign()
	bSign := b.Sign()
	switch {
	case aSign >= 0 && bSign < 0:
		return false
	case aSign < 0 && bSign >= 0:
		return true
	default:
		return a.Lt(b)
	}
}

func (a U256) Gt(b U256) bool {
	return b.Lt(a)
}

func (a U256) Ge(b U256) bool {
	return a.Eq(b) || a.Gt(b)
}

func (a U256) Sgt(b U256) bool {
	aSign := a.Sign()
	bSign := b.Sign()

	switch {
	case aSign >= 0 && bSign < 0:
		return true
	case aSign < 0 && bSign >= 0:
		return false
	default:
		return a.Gt(b)
	}
}

func (u U256) Neg() U256 {
	zero := U256{0}
	return zero.Sub(u)
}

func (a U256) Add(b U256) U256 {
	z, _ := a.addOverflow(b)
	return z
}

func (a U256) addOverflow(b U256) (U256, bool) {
	var (
		z     U256
		carry uint64
	)
	z[0], carry = bits.Add64(a[0], b[0], 0)
	z[1], carry = bits.Add64(a[1], b[1], carry)
	z[2], carry = bits.Add64(a[2], b[2], carry)
	z[3], carry = bits.Add64(a[3], b[3], carry)
	return z, carry != 0
}

func (x U256) AddMod(y, m U256) U256 {
	if m.IsZero() {
		return U256{0}
	}
	if z, overflow := x.addOverflow(y); overflow {
		sum := [5]uint64{z[0], z[1], z[2], z[3], 1}
		var quot [5]uint64
		return udivrem(quot[:], sum[:], &m)
	} else {
		return z.Mod(m)
	}
}

func (a U256) Sub(b U256) U256 {
	var (
		z     U256
		carry uint64
	)
	z[0], carry = bits.Sub64(a[0], b[0], 0)
	z[1], carry = bits.Sub64(a[1], b[1], carry)
	z[2], carry = bits.Sub64(a[2], b[2], carry)
	z[3], _ = bits.Sub64(a[3], b[3], carry)
	return z
}

func (a U256) Mul(b U256) U256 {
	var (
		res              U256
		carry            uint64
		res1, res2, res3 uint64
	)

	carry, res[0] = bits.Mul64(a[0], b[0])
	carry, res1 = umulHop(carry, a[1], b[0])
	carry, res2 = umulHop(carry, a[2], b[0])
	res3 = a[3]*b[0] + carry

	carry, res[1] = umulHop(res1, a[0], b[1])
	carry, res2 = umulStep(res2, a[1], b[1], carry)
	res3 = res3 + a[2]*b[1] + carry

	carry, res[2] = umulHop(res2, a[0], b[2])
	res3 = res3 + a[1]*b[2] + carry

	res[3] = res3 + a[0]*b[3]

	return res
}

func umulStep(z, x, y, carry uint64) (hi, lo uint64) {
	hi, lo = bits.Mul64(x, y)
	lo, carry = bits.Add64(lo, carry, 0)
	hi, _ = bits.Add64(hi, 0, carry)
	lo, carry = bits.Add64(lo, z, 0)
	hi, _ = bits.Add64(hi, 0, carry)
	return hi, lo
}

func umulHop(z, x, y uint64) (hi, lo uint64) {
	hi, lo = bits.Mul64(x, y)
	lo, carry := bits.Add64(lo, z, 0)
	hi, _ = bits.Add64(hi, 0, carry)
	return hi, lo
}

func (x U256) MulMod(y, m U256) U256 {
	if x.IsZero() || y.IsZero() || m.IsZero() {
		return U256{0}
	}
	p := umul(x, y)
	var (
		pl U256
		ph U256
	)
	copy(pl[:], p[:4])
	copy(ph[:], p[4:])

	// If the multiplication is within 256 bits use Mod().
	if ph.IsZero() {
		return pl.Mod(m)
	}

	var quot [8]uint64
	return udivrem(quot[:], p[:], &m)
}

func umul(x, y U256) [8]uint64 {
	var (
		res                           [8]uint64
		carry, carry4, carry5, carry6 uint64
		res1, res2, res3, res4, res5  uint64
	)

	carry, res[0] = bits.Mul64(x[0], y[0])
	carry, res1 = umulHop(carry, x[1], y[0])
	carry, res2 = umulHop(carry, x[2], y[0])
	carry4, res3 = umulHop(carry, x[3], y[0])

	carry, res[1] = umulHop(res1, x[0], y[1])
	carry, res2 = umulStep(res2, x[1], y[1], carry)
	carry, res3 = umulStep(res3, x[2], y[1], carry)
	carry5, res4 = umulStep(carry4, x[3], y[1], carry)

	carry, res[2] = umulHop(res2, x[0], y[2])
	carry, res3 = umulStep(res3, x[1], y[2], carry)
	carry, res4 = umulStep(res4, x[2], y[2], carry)
	carry6, res5 = umulStep(carry5, x[3], y[2], carry)

	carry, res[3] = umulHop(res3, x[0], y[3])
	carry, res[4] = umulStep(res4, x[1], y[3], carry)
	carry, res[5] = umulStep(res5, x[2], y[3], carry)
	res[7], res[6] = umulStep(carry6, x[3], y[3], carry)

	return res
}

func (a U256) Div(b U256) U256 {
	if b.IsZero() || b.Gt(a) {
		return U256{0}
	}
	if a.Eq(b) {
		return U256{1}
	}
	if a.IsUint64() {
		return U256{a[0] / b[0]}
	}
	var quot U256
	udivrem(quot[:], a[:], &b)
	return quot
}

func udivrem(quot, u []uint64, d *U256) (rem U256) {
	var dLen int
	for i := len(d) - 1; i >= 0; i-- {
		if d[i] != 0 {
			dLen = i + 1
			break
		}
	}

	shift := uint(bits.LeadingZeros64(d[dLen-1]))

	var dnStorage U256
	dn := dnStorage[:dLen]
	for i := dLen - 1; i > 0; i-- {
		dn[i] = (d[i] << shift) | (d[i-1] >> (64 - shift))
	}
	dn[0] = d[0] << shift

	var uLen int
	for i := len(u) - 1; i >= 0; i-- {
		if u[i] != 0 {
			uLen = i + 1
			break
		}
	}

	var unStorage [9]uint64
	un := unStorage[:uLen+1]
	un[uLen] = u[uLen-1] >> (64 - shift)
	for i := uLen - 1; i > 0; i-- {
		un[i] = (u[i] << shift) | (u[i-1] >> (64 - shift))
	}
	un[0] = u[0] << shift

	// TODO: Skip the highest word of numerator if not significant.

	if dLen == 1 {
		r := udivremBy1(quot, un, dn[0])
		rem = U256{r >> shift}
		return rem
	}

	udivremKnuth(quot, un, dn)

	for i := 0; i < dLen-1; i++ {
		rem[i] = (un[i] >> shift) | (un[i+1] << (64 - shift))
	}
	rem[dLen-1] = un[dLen-1] >> shift

	return rem
}

func udivremBy1(quot, u []uint64, d uint64) (rem uint64) {
	reciprocal := reciprocal2by1(d)
	rem = u[len(u)-1] // Set the top word as remainder.
	for j := len(u) - 2; j >= 0; j-- {
		quot[j], rem = udivrem2by1(rem, u[j], d, reciprocal)
	}
	return rem
}

func reciprocal2by1(d uint64) uint64 {
	reciprocal, _ := bits.Div64(^d, ^uint64(0), d)
	return reciprocal
}

func udivrem2by1(uh, ul, d, reciprocal uint64) (quot, rem uint64) {
	qh, ql := bits.Mul64(reciprocal, uh)
	ql, carry := bits.Add64(ql, ul, 0)
	qh, _ = bits.Add64(qh, uh, carry)
	qh++

	r := ul - qh*d

	if r > ql {
		qh--
		r += d
	}

	if r >= d {
		qh++
		r -= d
	}

	return qh, r
}

func udivremKnuth(quot, u, d []uint64) {
	dh := d[len(d)-1]
	dl := d[len(d)-2]
	reciprocal := reciprocal2by1(dh)

	for j := len(u) - len(d) - 1; j >= 0; j-- {
		u2 := u[j+len(d)]
		u1 := u[j+len(d)-1]
		u0 := u[j+len(d)-2]

		var qhat, rhat uint64
		if u2 >= dh { // Division overflows.
			qhat = ^uint64(0)
		} else {
			qhat, rhat = udivrem2by1(u2, u1, dh, reciprocal)
			ph, pl := bits.Mul64(qhat, dl)
			if ph > rhat || (ph == rhat && pl > u0) {
				qhat--
			}
		}

		// Multiply and subtract.
		borrow := subMulTo(u[j:], d, qhat)
		u[j+len(d)] = u2 - borrow
		if u2 < borrow { // Too much subtracted, add back.
			qhat--
			u[j+len(d)] += addTo(u[j:], d)
		}

		quot[j] = qhat // Store quotient digit.
	}
}

func subMulTo(x, y []uint64, multiplier uint64) uint64 {

	var borrow uint64
	for i := 0; i < len(y); i++ {
		s, carry1 := bits.Sub64(x[i], borrow, 0)
		ph, pl := bits.Mul64(y[i], multiplier)
		t, carry2 := bits.Sub64(s, pl, 0)
		x[i] = t
		borrow = ph + carry1 + carry2
	}
	return borrow
}

func addTo(x, y []uint64) uint64 {
	var carry uint64
	for i := 0; i < len(y); i++ {
		x[i], carry = bits.Add64(x[i], y[i], carry)
	}
	return carry
}

func (x U256) Mod(y U256) U256 {
	if x.IsZero() || y.IsZero() {
		return U256{0}
	}
	switch x.Cmp(y) {
	case -1:
		// x < y
		var z U256
		copy(z[:], x[:])
		return z
	case 0:
		// x == y
		return U256{0}
	}

	// At this point:
	// x != 0
	// y != 0
	// x > y

	// Shortcut trivial case
	if x.IsUint64() {
		return U256{x[0] % y[0]}
	}

	var quot U256
	return udivrem(quot[:], x[:], &y)
}

func (z U256) Sign() int {
	if z.IsZero() {
		return 0
	}
	if z[3] < 0x8000000000000000 {
		return 1
	}
	return -1
}

func (n U256) SDiv(d U256) U256 {
	if n.Sign() > 0 {
		if d.Sign() > 0 {
			// pos / pos
			return n.Div(d)
		} else {
			// pos / neg
			return n.Div(d.Neg()).Neg()
		}
	}

	if d.Sign() < 0 {
		// neg / neg
		return n.Neg().Div(d.Neg())
	}
	// neg / pos
	return n.Neg().Div(d).Neg()
}

func (x U256) SMod(y U256) U256 {
	ys := y.Sign()
	xs := x.Sign()

	// abs x
	if xs == -1 {
		x = x.Neg()
	}
	// abs y
	if ys == -1 {
		y = y.Neg()
	}
	z := x.Mod(y)
	if xs == -1 {
		z = z.Neg()
	}
	return z
}

func (base U256) Exp(exponent U256) U256 {
	res := U256{1, 0, 0, 0}
	multiplier := base
	expBitLen := exponent.BitLen()

	curBit := 0
	word := exponent[0]
	for ; curBit < expBitLen && curBit < 64; curBit++ {
		if word&1 == 1 {
			res = res.Mul(multiplier)
		}
		multiplier.squared()
		word >>= 1
	}

	word = exponent[1]
	for ; curBit < expBitLen && curBit < 128; curBit++ {
		if word&1 == 1 {
			res = res.Mul(multiplier)
		}
		multiplier.squared()
		word >>= 1
	}

	word = exponent[2]
	for ; curBit < expBitLen && curBit < 192; curBit++ {
		if word&1 == 1 {
			res = res.Mul(multiplier)
		}
		multiplier.squared()
		word >>= 1
	}

	word = exponent[3]
	for ; curBit < expBitLen && curBit < 256; curBit++ {
		if word&1 == 1 {
			res = res.Mul(multiplier)
		}
		multiplier.squared()
		word >>= 1
	}
	return res
}

func (z U256) BitLen() int {
	switch {
	case z[3] != 0:
		return 192 + bits.Len64(z[3])
	case z[2] != 0:
		return 128 + bits.Len64(z[2])
	case z[1] != 0:
		return 64 + bits.Len64(z[1])
	default:
		return bits.Len64(z[0])
	}
}

func (z *U256) squared() {
	var (
		res                    U256
		carry0, carry1, carry2 uint64
		res1, res2             uint64
	)

	carry0, res[0] = bits.Mul64(z[0], z[0])
	carry0, res1 = umulHop(carry0, z[0], z[1])
	carry0, res2 = umulHop(carry0, z[0], z[2])

	carry1, res[1] = umulHop(res1, z[0], z[1])
	carry1, res2 = umulStep(res2, z[1], z[1], carry1)

	carry2, res[2] = umulHop(res2, z[0], z[2])

	res[3] = 2*(z[0]*z[3]+z[1]*z[2]) + carry0 + carry1 + carry2

	*z = res
}

func (u U256) Not() U256 {
	return U256{^u[0], ^u[1], ^u[2], ^u[3]}
}

func (x U256) And(y U256) U256 {
	return U256{x[0] & y[0], x[1] & y[1], x[2] & y[2], x[3] & y[3]}
}

func (x U256) Or(y U256) U256 {
	return U256{x[0] | y[0], x[1] | y[1], x[2] | y[2], x[3] | y[3]}
}

func (x U256) Xor(y U256) U256 {
	return U256{x[0] ^ y[0], x[1] ^ y[1], x[2] ^ y[2], x[3] ^ y[3]}
}

func (x U256) Lsh(n uint) U256 {
	// n % 64 == 0
	if n&0x3f == 0 {
		switch n {
		case 0:
			return x
		case 64:
			x.lsh64()
			return x
		case 128:
			x.lsh128()
			return x
		case 192:
			x.lsh192()
			return x
		default:
			return U256{0}
		}
	}
	var (
		a, b uint64
	)
	// Big swaps first
	switch {
	case n > 192:
		if n > 256 {
			return U256{0}
		}
		x.lsh192()
		n -= 192
		goto sh192
	case n > 128:
		x.lsh128()
		n -= 128
		goto sh128
	case n > 64:
		x.lsh64()
		n -= 64
		goto sh64
	}

	// remaining shifts
	a = x[0] >> (64 - n)
	x[0] = x[0] << n

sh64:
	b = x[1] >> (64 - n)
	x[1] = (x[1] << n) | a

sh128:
	a = x[2] >> (64 - n)
	x[2] = (x[2] << n) | b

sh192:
	x[3] = (x[3] << n) | a

	return x
}

func (z *U256) lsh64() {
	z[3], z[2], z[1], z[0] = z[2], z[1], z[0], 0
}
func (z *U256) lsh128() {
	z[3], z[2], z[1], z[0] = z[1], z[0], 0, 0
}
func (z *U256) lsh192() {
	z[3], z[2], z[1], z[0] = z[0], 0, 0, 0
}

func (x U256) Rsh(n uint) U256 {
	// n % 64 == 0
	if n&0x3f == 0 {
		switch n {
		case 0:
			return x
		case 64:
			x.rsh64()
			return x
		case 128:
			x.rsh128()
			return x
		case 192:
			x.rsh192()
			return x
		default:
			return U256{0}
		}
	}
	var (
		a, b uint64
	)
	// Big swaps first
	switch {
	case n > 192:
		if n > 256 {
			return U256{0}
		}
		x.rsh192()
		n -= 192
		goto sh192
	case n > 128:
		x.rsh128()
		n -= 128
		goto sh128
	case n > 64:
		x.rsh64()
		n -= 64
		goto sh64
	}

	// remaining shifts
	a = x[3] << (64 - n)
	x[3] = x[3] >> n

sh64:
	b = x[2] << (64 - n)
	x[2] = (x[2] >> n) | a

sh128:
	a = x[1] << (64 - n)
	x[1] = (x[1] >> n) | b

sh192:
	x[0] = (x[0] >> n) | a

	return x
}

func (z *U256) rsh64() {
	z[3], z[2], z[1], z[0] = 0, z[3], z[2], z[1]
}
func (z *U256) rsh128() {
	z[3], z[2], z[1], z[0] = 0, 0, z[3], z[2]
}
func (z *U256) rsh192() {
	z[3], z[2], z[1], z[0] = 0, 0, 0, z[3]
}

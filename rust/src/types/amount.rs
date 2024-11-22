use std::{
    cmp::Ordering,
    fmt::{Debug, Display, LowerHex},
    ops::{
        Add, AddAssign, BitAnd, BitOr, BitXor, Deref, DerefMut, Div, DivAssign, Mul, MulAssign,
        Not, Rem, RemAssign, Shl, Shr, Sub, SubAssign,
    },
};

#[cfg(feature = "fuzzing")]
use arbitrary::Arbitrary;
use bnum::{
    cast::CastFrom,
    types::{I256, U256, U512},
};
use evmc_vm::{Address, Uint256};
use zerocopy::{transmute, transmute_ref, FromBytes, Immutable, IntoBytes};

/// This represents a 256-bit integer. Internally it is a 32 byte array of [`u8`] in big endian.
#[allow(non_camel_case_types)]
#[derive(Clone, Copy, FromBytes, IntoBytes, Immutable)]
#[cfg_attr(feature = "fuzzing", derive(Arbitrary))]
#[repr(align(8))]
pub struct u256([u8; 32]);

impl Deref for u256 {
    type Target = [u8; 32];

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for u256 {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl LowerHex for u256 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        if f.alternate() {
            f.write_str("0x")?;
        }
        for (i, byte) in self.into_iter().enumerate() {
            f.write_fmt(format_args!("{byte:02x}"))?;
            if i % 8 == 7 && i < 31 {
                f.write_str("_")?;
            }
        }

        Ok(())
    }
}

impl Display for u256 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let x = U256::from(*self);
        f.write_fmt(format_args!("{x}"))?;

        Ok(())
    }
}

impl Debug for u256 {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{self}")
    }
}

impl From<Uint256> for u256 {
    fn from(value: Uint256) -> Self {
        Self(value.bytes)
    }
}

impl From<u256> for Uint256 {
    fn from(value: u256) -> Self {
        Uint256 { bytes: value.0 }
    }
}

impl From<U256> for u256 {
    fn from(value: U256) -> Self {
        transmute!(*value.to_be().digits())
    }
}

impl From<u256> for U256 {
    fn from(value: u256) -> Self {
        U256::from_digits(transmute!(value.0)).to_be()
    }
}

impl From<I256> for u256 {
    fn from(value: I256) -> Self {
        u256::from(value.cast_unsigned())
    }
}

impl From<u256> for I256 {
    fn from(value: u256) -> Self {
        U256::from(value).cast_signed()
    }
}

impl From<U512> for u256 {
    fn from(value: U512) -> Self {
        u256::from(U256::cast_from(value))
    }
}

impl From<u256> for U512 {
    fn from(value: u256) -> Self {
        U512::cast_from(U256::from(value))
    }
}

impl From<[u8; 32]> for u256 {
    fn from(value: [u8; 32]) -> Self {
        Self(value)
    }
}

impl From<bool> for u256 {
    fn from(value: bool) -> Self {
        (value as u8).into()
    }
}

impl From<u8> for u256 {
    fn from(value: u8) -> Self {
        Self([
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, value,
        ])
    }
}

impl From<u64> for u256 {
    fn from(value: u64) -> Self {
        let bytes = value.to_be_bytes();
        Self([
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, bytes[0],
            bytes[1], bytes[2], bytes[3], bytes[4], bytes[5], bytes[6], bytes[7],
        ])
    }
}

impl From<usize> for u256 {
    fn from(value: usize) -> Self {
        u256::from(value as u64)
    }
}

impl From<Address> for u256 {
    fn from(value: Address) -> Self {
        let mut bytes = Self::ZERO;
        bytes[32 - 20..].copy_from_slice(&value.bytes);
        bytes
    }
}

impl From<&Address> for u256 {
    fn from(value: &Address) -> Self {
        let mut bytes = Self::ZERO;
        bytes[32 - 20..].copy_from_slice(&value.bytes);
        bytes
    }
}

impl From<u256> for Address {
    fn from(value: u256) -> Self {
        let mut addr = Address { bytes: [0; 20] };
        addr.bytes.copy_from_slice(&value[32 - 20..]);
        addr
    }
}

#[derive(Debug, PartialEq)]
pub struct U64Overflow;

impl TryFrom<u256> for u64 {
    type Error = U64Overflow;

    fn try_from(value: u256) -> Result<Self, Self::Error> {
        let (prefix, u64_bytes) = split_into_most_significant_24_and_least_significant_8(&value);
        if prefix != &[0; 24] {
            Err(U64Overflow)
        } else {
            Ok(u64::from_be_bytes(*u64_bytes))
        }
    }
}

impl Add for u256 {
    type Output = Self;

    fn add(self, rhs: Self) -> Self::Output {
        let lhs: U256 = self.into();
        let rhs: U256 = rhs.into();

        lhs.wrapping_add(rhs).into()
    }
}

impl AddAssign for u256 {
    fn add_assign(&mut self, rhs: Self) {
        *self = *self + rhs;
    }
}

impl Sub for u256 {
    type Output = Self;

    fn sub(self, rhs: Self) -> Self::Output {
        let lhs: U256 = self.into();
        let rhs: U256 = rhs.into();

        lhs.wrapping_sub(rhs).into()
    }
}

impl SubAssign for u256 {
    fn sub_assign(&mut self, rhs: Self) {
        *self = *self - rhs;
    }
}

impl Mul for u256 {
    type Output = Self;

    fn mul(self, rhs: Self) -> Self::Output {
        let lhs: U256 = self.into();
        let rhs: U256 = rhs.into();

        lhs.wrapping_mul(rhs).into()
    }
}

impl MulAssign for u256 {
    fn mul_assign(&mut self, rhs: Self) {
        *self = *self * rhs;
    }
}

impl Div for u256 {
    type Output = Self;

    fn div(self, rhs: Self) -> Self::Output {
        if rhs == u256::ZERO {
            return u256::ZERO;
        }
        let lhs: U256 = self.into();
        let rhs: U256 = rhs.into();

        lhs.wrapping_div(rhs).into()
    }
}

impl DivAssign for u256 {
    fn div_assign(&mut self, rhs: Self) {
        *self = *self / rhs;
    }
}

impl Rem for u256 {
    type Output = Self;

    fn rem(self, rhs: Self) -> Self::Output {
        if rhs == u256::ZERO {
            return u256::ZERO;
        }
        let lhs: U256 = self.into();
        let rhs: U256 = rhs.into();

        lhs.wrapping_rem(rhs).into()
    }
}

impl RemAssign for u256 {
    fn rem_assign(&mut self, rhs: Self) {
        *self = *self % rhs;
    }
}

impl PartialEq for u256 {
    fn eq(&self, other: &Self) -> bool {
        **self == **other
    }
}

impl Eq for u256 {}

impl PartialOrd for u256 {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for u256 {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        let lhs: U256 = (*self).into();
        let rhs: U256 = (*other).into();
        lhs.cmp(&rhs)
    }
}

impl BitAnd for u256 {
    type Output = Self;

    fn bitand(mut self, rhs: Self) -> Self::Output {
        for bit in 0..32 {
            self[bit] &= rhs[bit];
        }
        self
    }
}

impl BitOr for u256 {
    type Output = Self;

    fn bitor(mut self, rhs: Self) -> Self::Output {
        for bit in 0..32 {
            self[bit] |= rhs[bit];
        }
        self
    }
}

impl BitXor for u256 {
    type Output = Self;

    fn bitxor(mut self, rhs: Self) -> Self::Output {
        for bit in 0..32 {
            self[bit] ^= rhs[bit];
        }
        self
    }
}

impl Not for u256 {
    type Output = Self;

    fn not(mut self) -> Self::Output {
        for bit in 0..32 {
            self[bit] = !self[bit];
        }

        self
    }
}

impl Shl for u256 {
    type Output = Self;

    fn shl(self, rhs: Self) -> Self::Output {
        // rhs > 255
        if rhs[..31] != [0; 31] {
            return u256::ZERO;
        }
        let value: U256 = self.into();
        let shift = rhs[31] as u32;
        (value.wrapping_shl(shift)).into()
    }
}

impl Shr for u256 {
    type Output = Self;

    fn shr(self, rhs: Self) -> Self::Output {
        // rhs > 255
        if rhs[..31] != [0; 31] {
            return u256::ZERO;
        }
        let value: U256 = self.into();
        let shift = rhs[31] as u32;
        (value.wrapping_shr(shift)).into()
    }
}

impl u256 {
    pub const ZERO: Self = Self([0; 32]);
    pub const ONE: Self = Self([
        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
        0, 1,
    ]);
    pub const MAX: Self = Self([0xff; 32]);

    pub fn into_u64_with_overflow(self) -> (u64, bool) {
        let (prefix, u64_bytes) = split_into_most_significant_24_and_least_significant_8(&self);
        let overflow = prefix != &[0; 24];
        let num = u64::from_be_bytes(*u64_bytes);
        (num, overflow)
    }

    pub fn into_u64_saturating(self) -> u64 {
        let (prefix, u64_bytes) = split_into_most_significant_24_and_least_significant_8(&self);
        if prefix != &[0; 24] {
            u64::MAX
        } else {
            u64::from_be_bytes(*u64_bytes)
        }
    }

    pub fn sdiv(self, rhs: Self) -> Self {
        if rhs == u256::ZERO {
            return u256::ZERO;
        }
        let lhs: I256 = self.into();
        let rhs: I256 = rhs.into();

        lhs.wrapping_div(rhs).into()
    }

    pub fn srem(self, rhs: Self) -> Self {
        if rhs == u256::ZERO {
            return u256::ZERO;
        }
        let lhs: I256 = self.into();
        let rhs: I256 = rhs.into();

        lhs.wrapping_rem(rhs).into()
    }

    pub fn addmod(s1: Self, s2: Self, m: Self) -> Self {
        if m == u256::ZERO {
            return u256::ZERO;
        }
        let s1: U512 = s1.into();
        let s2: U512 = s2.into();
        let m: U512 = m.into();

        (s1 + s2).rem(m).into()
    }

    pub fn mulmod(s1: Self, s2: Self, m: Self) -> Self {
        if m == u256::ZERO {
            return u256::ZERO;
        }
        let f1: U512 = s1.into();
        let f2: U512 = s2.into();
        let m: U512 = m.into();

        (f1 * f2).rem(m).into()
    }

    pub fn pow(self, exp: Self) -> Self {
        let base: U256 = self.into();
        let exp: U256 = exp.into();
        let mut res = U256::ONE;

        for bit in (0..U256::BITS).rev().map(|bit| exp.bit(bit)) {
            res = res.wrapping_mul(res);
            if bit {
                res = res.wrapping_mul(base);
            }
        }

        res.into()
    }

    pub fn signextend(self, rhs: Self) -> Self {
        let (lhs, lhs_overflow) = self.into_u64_with_overflow();
        let lhs = lhs as usize;
        if lhs_overflow || lhs > 31 {
            return rhs;
        }

        let byte = 31 - lhs; // lhs <= 31 so this does not underflow
        let negative = (rhs[byte] & 0x80) > 0;

        let rhs: U256 = rhs.into();

        let res = if negative {
            rhs | (U256::MAX << ((32 - byte) * 8))
        } else {
            rhs & (U256::MAX >> (byte * 8))
        };

        res.into()
    }

    pub fn slt(&self, rhs: &Self) -> bool {
        let lhs: I256 = (*self).into();
        let rhs: I256 = (*rhs).into();
        lhs.cmp(&rhs) == Ordering::Less
    }

    pub fn sgt(&self, rhs: &Self) -> bool {
        let lhs: I256 = (*self).into();
        let rhs: I256 = (*rhs).into();
        lhs.cmp(&rhs) == Ordering::Greater
    }

    pub fn byte(&self, index: Self) -> Self {
        if index >= 32u8.into() {
            return u256::ZERO;
        }
        let idx = index[31];
        self[idx as usize].into()
    }

    pub fn sar(self, rhs: Self) -> Self {
        let negative = self[0] & 0x80 > 0;
        // rhs > 255
        if rhs[..31] != [0; 31] {
            if negative {
                return u256::MAX;
            } else {
                return u256::ZERO;
            }
        }
        let value: U256 = self.into();
        let shift = rhs[31] as u32;
        let mut shr = value.wrapping_shr(shift);
        if negative {
            shr |= U256::MAX.wrapping_shl(255 - shift);
        }
        shr.into()
    }
}

fn split_into_most_significant_24_and_least_significant_8(input: &u256) -> (&[u8; 24], &[u8; 8]) {
    #[derive(FromBytes, Immutable)]
    #[repr(C)]
    struct Split {
        smb24: [u8; 24],
        lsb8: [u8; 8],
    }
    let s: &Split = transmute_ref!(input);
    (&s.smb24, &s.lsb8)
}

#[cfg(test)]
mod tests {
    use bnum::{
        cast::CastFrom,
        types::{I256, U256, U512},
    };

    use crate::types::amount::{u256, U64Overflow};

    #[test]
    fn display() {
        let x = [
            (
                u256::from(0u8),
                [
                    "0",
                    "0000000000000000_0000000000000000_0000000000000000_0000000000000000",
                    "0x0000000000000000_0000000000000000_0000000000000000_0000000000000000",
                ],
            ),
            (
                u256::from(0xfeu8),
                [
                    "254",
                    "0000000000000000_0000000000000000_0000000000000000_00000000000000fe",
                    "0x0000000000000000_0000000000000000_0000000000000000_00000000000000fe",
                ],
            ),
            (
                u256::from(0xfeu8) << u256::from(8 * 31u8),
                [
                    "114887463540149662646824336688307533573166312910440247132899321632851308314624",
                    "fe00000000000000_0000000000000000_0000000000000000_0000000000000000",
                    "0xfe00000000000000_0000000000000000_0000000000000000_0000000000000000",
                ],
            ),
        ];
        for (value, fmt_strings) in x {
            assert_eq!(format!("{value}",), fmt_strings[0]);
            assert_eq!(format!("{value:x}",), fmt_strings[1]);
            assert_eq!(format!("{value:#x}",), fmt_strings[2]);
        }
    }

    #[test]
    fn conversions() {
        let v1 = u256::from([
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 1,
        ]);
        let v255 = u256::from([
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 255,
        ]);

        assert_eq!(u256::from(false), u256::ZERO);
        assert_eq!(u256::from(true), v1);

        assert_eq!(u256::from(0u8), u256::ZERO);
        assert_eq!(u256::from(1u8), v1);
        assert_eq!(u256::from(255u8), v255);

        assert_eq!(u256::from(0u64), u256::ZERO);
        assert_eq!(u256::from(1u64), v1);
        assert_eq!(u256::from(255u64), v255);
        for num in [0, 1, u64::MAX - 1, u64::MAX] {
            assert_eq!(u256::from(num).try_into(), Ok(num));
        }
        for num in [0, 1, u64::MAX - 1, u64::MAX] {
            assert_eq!(u256::from(num).into_u64_with_overflow(), (num, false));
        }
        for num in [0, 1, u64::MAX - 1, u64::MAX] {
            assert_eq!(u256::from(num).into_u64_saturating(), num);
        }
        assert_eq!(u256::MAX.try_into(), Result::<u64, _>::Err(U64Overflow));
        assert_eq!(u256::MAX.into_u64_with_overflow(), (u64::MAX, true));
        assert_eq!(u256::MAX.into_u64_saturating(), u64::MAX);

        assert_eq!(U256::from(u256::ZERO), U256::ZERO);
        assert_eq!(U256::from(v1), U256::ONE);
        assert_eq!(U256::from(u256::MAX), U256::MAX);
        assert_eq!(u256::from(U256::ZERO), u256::ZERO);
        assert_eq!(u256::from(U256::ONE), v1);
        assert_eq!(u256::from(U256::MAX), u256::MAX);

        assert_eq!(I256::from(u256::ZERO), I256::ZERO);
        assert_eq!(I256::from(v1), I256::ONE);
        assert_eq!(I256::from(u256::MAX), I256::NEG_ONE);
        assert_eq!(u256::from(I256::ZERO), u256::ZERO);
        assert_eq!(u256::from(I256::ONE), v1);
        assert_eq!(u256::from(I256::MAX), u256::MAX >> v1);

        assert_eq!(U512::from(u256::ZERO), U512::ZERO);
        assert_eq!(U512::from(v1), U512::ONE);
        assert_eq!(U512::from(u256::MAX), U512::cast_from(U256::MAX));
        assert_eq!(u256::from(U512::ZERO), u256::ZERO);
        assert_eq!(u256::from(U512::ONE), v1);
        assert_eq!(u256::from(U512::MAX), u256::MAX);
    }
}

use std::{
    cmp::Ordering,
    mem,
    ops::{
        Add, AddAssign, BitAnd, BitOr, BitXor, Deref, DerefMut, Div, DivAssign, Mul, MulAssign,
        Not, Rem, RemAssign, Shl, Shr, Sub, SubAssign,
    },
};

use bnum::types::{I256, U256, U512};
use evmc_vm::{Address, Uint256};

#[allow(non_camel_case_types)]
#[derive(Debug, Clone, Copy)]
#[repr(transparent)]
pub struct u256(Uint256);

impl Deref for u256 {
    type Target = [u8; 32];

    fn deref(&self) -> &Self::Target {
        &self.0.bytes
    }
}

impl DerefMut for u256 {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0.bytes
    }
}

impl u256 {
    pub const ZERO: Self = Self(Uint256 { bytes: [0; 32] });
    pub const MAX: Self = Self(Uint256 { bytes: [0xff; 32] });
}

impl From<Uint256> for u256 {
    fn from(value: Uint256) -> Self {
        Self(value)
    }
}

impl From<u256> for Uint256 {
    fn from(value: u256) -> Self {
        value.0
    }
}

impl From<U256> for u256 {
    fn from(value: U256) -> Self {
        let be_value = value.to_be();
        let bytes: [u8; 32] = unsafe { mem::transmute(be_value) };
        bytes.into()
    }
}

impl From<u256> for U256 {
    fn from(value: u256) -> Self {
        let mut bytes = value.0.bytes;
        //let lhs = bnum::BUint::from_be_bytes(lhs); // TODO required nightly
        //let lhs: U256 = bnum::BUint::from_be_slice(&lhs).unwrap(); // works but has overhead
        bytes.reverse();
        unsafe { mem::transmute(bytes) }
    }
}

impl From<I256> for u256 {
    fn from(value: I256) -> Self {
        let be_value = value.to_be();
        let bytes: [u8; 32] = unsafe { mem::transmute(be_value) };
        bytes.into()
    }
}

impl From<u256> for I256 {
    fn from(value: u256) -> Self {
        let mut bytes = value.0.bytes;
        //let lhs = bnum::BUint::from_be_bytes(lhs); // TODO required nightly
        //let lhs: U256 = bnum::BUint::from_be_slice(&lhs).unwrap(); // works but has overhead
        bytes.reverse();
        unsafe { mem::transmute(bytes) }
    }
}

impl From<U512> for u256 {
    fn from(value: U512) -> Self {
        let be_value = value.to_be();
        let bytes: [u8; 64] = unsafe { mem::transmute(be_value) };
        (&bytes[32..]).try_into().unwrap()
    }
}

impl From<u256> for U512 {
    fn from(value: u256) -> Self {
        U512::from_be_slice(&value.0.bytes).unwrap()
    }
}

impl From<u256> for [u8; 32] {
    fn from(value: u256) -> Self {
        value.0.bytes
    }
}

impl From<[u8; 32]> for u256 {
    fn from(value: [u8; 32]) -> Self {
        Self(Uint256 { bytes: value })
    }
}

impl From<u8> for u256 {
    fn from(value: u8) -> Self {
        [
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, value,
        ]
        .into()
    }
}

impl From<&Address> for u256 {
    fn from(value: &Address) -> Self {
        let mut bytes = [0; 32];
        bytes[32 - 20..].copy_from_slice(&value.bytes);
        bytes.into()
    }
}

impl From<u256> for Address {
    fn from(value: u256) -> Self {
        let mut bytes = [0; 20];
        bytes.copy_from_slice(&value[32 - 20..]);
        Address { bytes }
    }
}

#[derive(Debug)]
pub struct SliceTooLarge;

impl TryFrom<&[u8]> for u256 {
    type Error = SliceTooLarge;

    fn try_from(value: &[u8]) -> Result<Self, Self::Error> {
        let len = value.len();
        if len > 32 {
            return Err(SliceTooLarge);
        }
        let mut bytes = [0; 32];
        bytes[32 - len..].copy_from_slice(value);
        Ok(bytes.into())
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
        let lhs: U256 = self.into();
        let rhs: U256 = rhs.into();
        if rhs == U256::ZERO {
            return U256::ZERO.into();
        }

        lhs.wrapping_div(rhs).into()
    }
}

impl DivAssign for u256 {
    fn div_assign(&mut self, rhs: Self) {
        *self = *self / rhs;
    }
}

impl u256 {
    pub fn sdiv(self, rhs: Self) -> Self {
        let lhs: I256 = self.into();
        let rhs: I256 = rhs.into();
        if rhs == I256::ZERO {
            return u256::ZERO;
        }

        lhs.wrapping_div(rhs).into()
    }
}

impl Rem for u256 {
    type Output = Self;

    fn rem(self, rhs: Self) -> Self::Output {
        let lhs: U256 = self.into();
        let rhs: U256 = rhs.into();
        if rhs == U256::ZERO {
            return u256::ZERO;
        }

        lhs.wrapping_rem(rhs).into()
    }
}

impl RemAssign for u256 {
    fn rem_assign(&mut self, rhs: Self) {
        *self = *self % rhs;
    }
}

impl u256 {
    pub fn srem(self, rhs: Self) -> Self {
        let lhs: I256 = self.into();
        let rhs: I256 = rhs.into();
        if rhs == I256::ZERO {
            return u256::ZERO;
        }

        lhs.wrapping_rem(rhs).into()
    }

    pub fn addmod(s1: Self, s2: Self, m: Self) -> Self {
        let s1: U512 = s1.into();
        let s2: U512 = s2.into();
        let m: U512 = m.into();
        if m == U512::ZERO {
            return u256::ZERO;
        }

        (s1 + s2).rem(m).into()
    }

    pub fn mulmod(s1: Self, s2: Self, m: Self) -> Self {
        let f1: U512 = s1.into();
        let f2: U512 = s2.into();
        let m: U512 = m.into();
        if m == U512::ZERO {
            return u256::ZERO;
        }

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
        if U256::from(self) > U256::from_digit(31) {
            return rhs;
        }

        let byte = 31 - self[31]; // self <= 31 so it fits into the lower significant bit
        let negative = (rhs[byte as usize] & 0x80) > 0;

        let rhs: U256 = rhs.into();

        let res = if negative {
            rhs | (U256::MAX << ((32 - byte) * 8))
        } else {
            rhs & (U256::MAX >> (byte * 8))
        };

        res.into()
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

impl u256 {
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

impl u256 {
    pub fn byte(&self, index: Self) -> Self {
        if index >= 32.into() {
            return u256::ZERO;
        }
        let idx = index[31];
        self[idx as usize].into()
    }
}

impl Shl for u256 {
    type Output = Self;

    fn shl(self, rhs: Self) -> Self::Output {
        let value: U256 = self.into();
        if u256::from(rhs) > u256::from(255) {
            return u256::ZERO;
        }
        let shift = rhs[31] as u32;
        (value.wrapping_shl(shift)).into()
    }
}

impl Shr for u256 {
    type Output = Self;

    fn shr(self, rhs: Self) -> Self::Output {
        let value: U256 = self.into();
        if u256::from(rhs) > u256::from(255) {
            return u256::ZERO;
        }
        let shift = rhs[31] as u32;
        (value.wrapping_shr(shift)).into()
    }
}

impl u256 {
    pub fn sar(self, rhs: Self) -> Self {
        let value: U256 = self.into();
        let negative = self[0] & 0x80 > 0;
        if u256::from(rhs) > u256::from(255) {
            if negative {
                return u256::MAX;
            } else {
                return u256::ZERO;
            }
        }
        let shift = rhs[31] as u32;
        let mut shr = value.wrapping_shr(shift);
        if negative {
            shr |= U256::MAX.wrapping_shl(255 - shift);
        }
        shr.into()
    }
}

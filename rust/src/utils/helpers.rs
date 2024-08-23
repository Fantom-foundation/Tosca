use std::cmp::min;

use evmc_vm::{MessageFlags, Revision, StatusCode};

use crate::{
    interpreter::Interpreter,
    types::{u256, ExecutionContextTrait},
    utils::gas::consume_copy_cost,
};

pub trait SliceExt {
    fn get_within_bounds(&self, offset: u256, len: u64) -> &[u8];

    fn set_to_zero(&mut self);

    fn copy_padded(&mut self, src: &[u8], gas_left: &mut u64) -> Result<(), StatusCode>;
}

impl SliceExt for [u8] {
    #[inline(always)]
    fn get_within_bounds(&self, offset: u256, len: u64) -> &[u8] {
        if len == 0 {
            return &[];
        }
        let (offset, offset_overflow) = offset.into_u64_with_overflow();
        if offset_overflow {
            return &[];
        }
        let offset = offset as usize;
        let len = len as usize;
        let (end, end_overflow) = offset.overflowing_add(len);
        if end_overflow || offset >= self.len() {
            &[]
        } else {
            &self[offset..min(end, self.len())]
        }
    }

    #[inline(always)]
    fn set_to_zero(&mut self) {
        for byte in self {
            *byte = 0;
        }
    }

    #[inline(always)]
    fn copy_padded(&mut self, src: &[u8], gas_left: &mut u64) -> Result<(), StatusCode> {
        consume_copy_cost(gas_left, self.len() as u64)?;
        self[..src.len()].copy_from_slice(src);
        self[src.len()..].set_to_zero();
        Ok(())
    }
}

#[inline(always)]
pub fn word_size(byte_len: u64) -> Result<u64, StatusCode> {
    let (end, overflow) = byte_len.overflowing_add(31);
    if overflow {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    Ok(end / 32)
}

#[inline(always)]
pub fn check_min_revision(min_revision: Revision, revision: Revision) -> Result<(), StatusCode> {
    if revision < min_revision {
        return Err(StatusCode::EVMC_UNDEFINED_INSTRUCTION);
    }
    Ok(())
}

#[inline(always)]
pub fn check_not_read_only<E>(state: &Interpreter<E>) -> Result<(), StatusCode>
where
    E: ExecutionContextTrait,
{
    if state.revision >= Revision::EVMC_BYZANTIUM
        && state.message.flags() == MessageFlags::EVMC_STATIC as u32
    {
        return Err(StatusCode::EVMC_STATIC_MODE_VIOLATION);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::SliceExt;
    use crate::types::u256;

    #[test]
    fn get_slice_within_bounds() {
        assert_eq!([1].get_within_bounds(u256::ZERO, 0), &[]);
        assert_eq!([1].get_within_bounds(u256::ZERO, 1), &[1]);
        assert_eq!([1].get_within_bounds(u256::ZERO, 2), &[1]);
        assert_eq!([1].get_within_bounds(1u8.into(), 1), &[]);
    }

    #[test]
    fn copy_slice_padded() {
        let src = [];
        let mut dest = [];
        assert_eq!(dest.copy_padded(&src, &mut 1_000_000), Ok(()));

        let src = [];
        let mut dest = [1];
        assert_eq!(dest.copy_padded(&src, &mut 1_000_000), Ok(()));
        assert_eq!(dest, [0]);

        let src = [2];
        let mut dest = [1];
        assert_eq!(dest.copy_padded(&src, &mut 1_000_000), Ok(()));
        assert_eq!(dest, [2]);

        let src = [2];
        let mut dest = [1, 3];
        assert_eq!(dest.copy_padded(&src, &mut 1_000_000), Ok(()));
        assert_eq!(dest, [2, 0]);
    }
}

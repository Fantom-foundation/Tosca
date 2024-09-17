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
        consume_copy_cost(self.len() as u64, gas_left)?;
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
    use evmc_vm::{MessageFlags, Revision, StatusCode};

    use crate::{
        interpreter::Interpreter,
        types::{u256, MockExecutionContextTrait, MockExecutionMessage},
        utils::{self, SliceExt},
    };

    #[test]
    fn get_within_bounds() {
        assert_eq!([].get_within_bounds(u256::ZERO, 1), &[]);
        assert_eq!([1].get_within_bounds(u256::ZERO, 0), &[]);
        assert_eq!([1].get_within_bounds(u256::ZERO, 1), &[1]);
        assert_eq!([1].get_within_bounds(u256::ZERO, 2), &[1]);
        assert_eq!([1].get_within_bounds(u256::ONE, 1), &[]);
        assert_eq!([1].get_within_bounds(u256::MAX, 1), &[]);
    }

    #[test]
    fn set_to_zero() {
        let mut data = [];
        data.set_to_zero();
        assert_eq!(&data, &[]);
        let mut data = [1];
        data.set_to_zero();
        assert_eq!(&data, &[0]);
        let mut data = [1, 2];
        data.set_to_zero();
        assert_eq!(&data, &[0, 0]);
    }

    #[test]
    fn copy_padded() {
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

        let src = [3];
        let mut dest = [1, 2];
        assert_eq!(dest.copy_padded(&src, &mut 1_000_000), Ok(()));
        assert_eq!(dest, [3, 0]);

        let src = [2];
        let mut dest = [1];
        assert_eq!(
            dest.copy_padded(&src, &mut 0),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );
    }

    #[test]
    fn word_size() {
        assert_eq!(utils::word_size(0), Ok(0));
        assert_eq!(utils::word_size(1), Ok(1));
        assert_eq!(utils::word_size(32), Ok(1));
        assert_eq!(utils::word_size(33), Ok(2));
        assert_eq!(utils::word_size(u64::MAX), Err(StatusCode::EVMC_OUT_OF_GAS));
    }

    #[test]
    fn check_min_revision() {
        assert_eq!(
            utils::check_min_revision(Revision::EVMC_FRONTIER, Revision::EVMC_FRONTIER),
            Ok(())
        );
        assert_eq!(
            utils::check_min_revision(Revision::EVMC_FRONTIER, Revision::EVMC_CANCUN),
            Ok(())
        );
        assert_eq!(
            utils::check_min_revision(Revision::EVMC_CANCUN, Revision::EVMC_FRONTIER),
            Err(StatusCode::EVMC_UNDEFINED_INSTRUCTION)
        );
    }

    #[test]
    fn check_not_read_only() {
        let message = MockExecutionMessage::default().into();
        let mut context = MockExecutionContextTrait::new();
        let interpreter = Interpreter::new(Revision::EVMC_FRONTIER, &message, &mut context, &[]);
        assert_eq!(utils::check_not_read_only(&interpreter), Ok(()));

        let interpreter = Interpreter::new(Revision::EVMC_BYZANTIUM, &message, &mut context, &[]);
        assert_eq!(utils::check_not_read_only(&interpreter), Ok(()));

        let message = MockExecutionMessage {
            flags: MessageFlags::EVMC_STATIC as u32,
            ..Default::default()
        };
        let message = message.into();
        let interpreter = Interpreter::new(Revision::EVMC_BYZANTIUM, &message, &mut context, &[]);
        assert_eq!(
            utils::check_not_read_only(&interpreter),
            Err(StatusCode::EVMC_STATIC_MODE_VIOLATION)
        );
    }
}

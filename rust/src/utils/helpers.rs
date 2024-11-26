use std::cmp::min;

use evmc_vm::{ExecutionMessage, MessageFlags, Revision};

use crate::{
    types::{u256, FailStatus},
    utils::Gas,
};

pub trait SliceExt {
    fn get_within_bounds(&self, offset: u256, len: u64) -> &[u8];

    fn copy_padded(&mut self, src: &[u8], gas_left: &mut Gas) -> Result<(), FailStatus>;
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
    fn copy_padded(&mut self, src: &[u8], gas_left: &mut Gas) -> Result<(), FailStatus> {
        gas_left.consume_copy_cost(self.len() as u64)?;
        self[..src.len()].copy_from_slice(src);
        self[src.len()..].fill(0);
        Ok(())
    }
}

#[inline(always)]
pub fn word_size(byte_len: u64) -> Result<u64, FailStatus> {
    let (end, overflow) = byte_len.overflowing_add(31);
    if overflow {
        return Err(FailStatus::OutOfGas);
    }
    Ok(end / 32)
}

#[inline(always)]
pub fn check_min_revision(min_revision: Revision, revision: Revision) -> Result<(), FailStatus> {
    if revision < min_revision {
        return Err(FailStatus::UndefinedInstruction);
    }
    Ok(())
}

#[inline(always)]
pub fn check_not_read_only(message: &ExecutionMessage) -> Result<(), FailStatus> {
    if message.flags() == MessageFlags::EVMC_STATIC as u32 {
        return Err(FailStatus::StaticModeViolation);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use evmc_vm::{MessageFlags, Revision};

    use crate::{
        interpreter::Interpreter,
        types::{u256, FailStatus, MockExecutionContextTrait, MockExecutionMessage},
        utils::{self, Gas, SliceExt},
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
    fn copy_padded() {
        let src = [];
        let mut dest = [];
        assert_eq!(dest.copy_padded(&src, &mut Gas::new(1_000_000)), Ok(()));

        let src = [];
        let mut dest = [1];
        assert_eq!(dest.copy_padded(&src, &mut Gas::new(1_000_000)), Ok(()));
        assert_eq!(dest, [0]);

        let src = [2];
        let mut dest = [1];
        assert_eq!(dest.copy_padded(&src, &mut Gas::new(1_000_000)), Ok(()));
        assert_eq!(dest, [2]);

        let src = [3];
        let mut dest = [1, 2];
        assert_eq!(dest.copy_padded(&src, &mut Gas::new(1_000_000)), Ok(()));
        assert_eq!(dest, [3, 0]);

        let src = [2];
        let mut dest = [1];
        assert_eq!(
            dest.copy_padded(&src, &mut Gas::new(0)),
            Err(FailStatus::OutOfGas)
        );
    }

    #[test]
    fn word_size() {
        assert_eq!(utils::word_size(0), Ok(0));
        assert_eq!(utils::word_size(1), Ok(1));
        assert_eq!(utils::word_size(32), Ok(1));
        assert_eq!(utils::word_size(33), Ok(2));
        assert_eq!(utils::word_size(u64::MAX), Err(FailStatus::OutOfGas));
    }

    #[test]
    fn check_min_revision() {
        assert_eq!(
            utils::check_min_revision(Revision::EVMC_ISTANBUL, Revision::EVMC_ISTANBUL),
            Ok(())
        );
        assert_eq!(
            utils::check_min_revision(Revision::EVMC_ISTANBUL, Revision::EVMC_CANCUN),
            Ok(())
        );
        assert_eq!(
            utils::check_min_revision(Revision::EVMC_CANCUN, Revision::EVMC_ISTANBUL),
            Err(FailStatus::UndefinedInstruction)
        );
    }

    #[test]
    fn check_not_read_only() {
        let message = MockExecutionMessage::default().into();
        let mut context = MockExecutionContextTrait::new();
        let interpreter = Interpreter::new(Revision::EVMC_CANCUN, &message, &mut context, &[]);
        assert_eq!(utils::check_not_read_only(&interpreter), Ok(()));

        let message = MockExecutionMessage {
            flags: MessageFlags::EVMC_STATIC as u32,
            ..Default::default()
        };
        let message = message.into();
        let interpreter = Interpreter::new(Revision::EVMC_CANCUN, &message, &mut context, &[]);
        assert_eq!(
            utils::check_not_read_only(&interpreter),
            Err(FailStatus::StaticModeViolation)
        );
    }
}

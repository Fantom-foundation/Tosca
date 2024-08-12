use std::{cmp::max, iter};

use evmc_vm::{StatusCode, StepStatusCode};

use crate::{
    interpreter::{consume_copy_cost, consume_dyn_gas, word_size},
    types::u256,
};

pub(super) struct Memory(Vec<u8>);

impl Memory {
    pub fn new(memory: Vec<u8>) -> Self {
        Self(memory)
    }

    pub fn into_inner(self) -> Vec<u8> {
        self.0
    }

    pub fn len(&self) -> u64 {
        self.0.len() as u64
    }

    fn expand(
        &mut self,
        new_len_bytes: u64,
        gas_left: &mut u64,
    ) -> Result<(), (StepStatusCode, StatusCode)> {
        let current_len = self.0.len() as u64;
        let new_len = word_size(new_len_bytes) * 32;
        if new_len > current_len {
            self.consume_expansion_cost(gas_left, new_len)?;
            self.0
                .extend(iter::repeat(0).take((new_len - current_len) as usize))
        }
        Ok(())
    }

    fn consume_expansion_cost(
        &self,
        gas_left: &mut u64,
        new_len: u64,
    ) -> Result<(), (StepStatusCode, StatusCode)> {
        fn memory_cost(size: u64) -> Result<u64, (StepStatusCode, StatusCode)> {
            let memory_size_word = word_size(size);
            let Some(pow2) = memory_size_word.checked_pow(2) else {
                return Err((
                    StepStatusCode::EVMC_STEP_FAILED,
                    StatusCode::EVMC_OUT_OF_GAS,
                ));
            };
            Ok(pow2 / 512 + (3 * memory_size_word))
        }

        let current_len = self.0.len() as u64;

        if new_len > current_len {
            let memory_expansion_cost = memory_cost(new_len)? - memory_cost(current_len)?;
            consume_dyn_gas(gas_left, memory_expansion_cost)?;
        }
        Ok(())
    }

    pub fn get_slice(
        &mut self,
        offset: u256,
        len: u64,
        gas_left: &mut u64,
    ) -> Result<&mut [u8], (StepStatusCode, StatusCode)> {
        if len == 0 {
            return Ok(&mut []);
        }
        let (offset, offset_overflow) = offset.into_u64_with_overflow();
        if offset_overflow {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_INVALID_MEMORY_ACCESS,
            ));
        }
        let end = offset + len;
        self.expand(end, gas_left)?;

        Ok(&mut self.0[offset as usize..end as usize])
    }

    pub fn get_word(
        &mut self,
        offset: u256,
        gas_left: &mut u64,
    ) -> Result<u256, (StepStatusCode, StatusCode)> {
        let slice = self.get_slice(offset, 32u8.into(), gas_left)?;
        let mut bytes = [0; 32];
        bytes.copy_from_slice(slice);
        Ok(bytes.into())
    }

    pub fn get_byte(
        &mut self,
        offset: u256,
        gas_left: &mut u64,
    ) -> Result<&mut u8, (StepStatusCode, StatusCode)> {
        self.get_slice(offset, 1u8.into(), gas_left)
            .map(|slice| &mut slice[0])
    }

    pub fn copy_within(
        &mut self,
        src_offset: u256,
        dest_offset: u256,
        len: u256,
        gas_left: &mut u64,
    ) -> Result<(), (StepStatusCode, StatusCode)> {
        let (src_offset, src_overflow) = src_offset.into_u64_with_overflow();
        let (dest_offset, dest_overflow) = dest_offset.into_u64_with_overflow();
        let (len, len_overflow) = len.into_u64_with_overflow();
        if src_overflow || dest_overflow || len_overflow {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_OUT_OF_GAS,
            ));
        }
        consume_copy_cost(gas_left, len)?;
        self.expand(max(src_offset, dest_offset) + len, gas_left)?;
        let src_offset = src_offset as usize;
        let dest_offset = dest_offset as usize;
        let len = len as usize;
        self.0
            .copy_within(src_offset..src_offset + len, dest_offset);
        Ok(())
    }
}

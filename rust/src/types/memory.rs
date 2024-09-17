use std::{cmp::max, iter};

use evmc_vm::StatusCode;

use crate::{
    types::u256,
    utils::{consume_copy_cost, consume_gas, word_size},
};

#[derive(Debug)]
pub struct Memory(Vec<u8>);

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

    fn expand(&mut self, new_len_bytes: u64, gas_left: &mut u64) -> Result<(), StatusCode> {
        let current_len = self.0.len() as u64;
        let new_len = word_size(new_len_bytes)? * 32; // word_size just did a division by 32 so * will not overflow
        if new_len > current_len {
            self.consume_expansion_cost(gas_left, new_len)?;
            self.0
                .extend(iter::repeat(0).take((new_len - current_len) as usize));
        }
        Ok(())
    }

    fn consume_expansion_cost(&self, gas_left: &mut u64, new_len: u64) -> Result<(), StatusCode> {
        fn memory_cost(size: u64) -> Result<u64, StatusCode> {
            let word_size = word_size(size)?;
            let (pow2, pow2_overflow) = word_size.overflowing_pow(2);
            let (word_size_3, word_size_3_overflow) = word_size.overflowing_mul(3);
            let (cost, cost_overflow) = (pow2 / 512).overflowing_add(word_size_3);
            if pow2_overflow || word_size_3_overflow || cost_overflow {
                return Err(StatusCode::EVMC_OUT_OF_GAS);
            };
            Ok(cost)
        }

        let current_len = self.0.len() as u64;

        if new_len > current_len {
            let memory_expansion_cost = memory_cost(new_len)? - memory_cost(current_len)?;
            consume_gas(memory_expansion_cost, gas_left)?;
        }
        Ok(())
    }

    pub fn get_mut_slice(
        &mut self,
        offset: u256,
        len: u64,
        gas_left: &mut u64,
    ) -> Result<&mut [u8], StatusCode> {
        if len == 0 {
            return Ok(&mut []);
        }
        let (offset, offset_overflow) = offset.into_u64_with_overflow();
        let (end, end_overflow) = offset.overflowing_add(len);
        if offset_overflow || end_overflow {
            return Err(StatusCode::EVMC_OUT_OF_GAS);
        }
        self.expand(end, gas_left)?;

        Ok(&mut self.0[offset as usize..end as usize])
    }

    pub fn get_word(&mut self, offset: u256, gas_left: &mut u64) -> Result<u256, StatusCode> {
        let slice = self.get_mut_slice(offset, 32u8.into(), gas_left)?;
        let mut bytes = [0; 32];
        bytes.copy_from_slice(slice);
        Ok(bytes.into())
    }

    pub fn get_mut_byte(
        &mut self,
        offset: u256,
        gas_left: &mut u64,
    ) -> Result<&mut u8, StatusCode> {
        let slice = self.get_mut_slice(offset, 1, gas_left)?;
        Ok(&mut slice[0])
    }

    pub fn copy_within(
        &mut self,
        src_offset: u256,
        dest_offset: u256,
        len: u256,
        gas_left: &mut u64,
    ) -> Result<(), StatusCode> {
        let (src_offset, src_overflow) = src_offset.into_u64_with_overflow();
        let (dest_offset, dest_overflow) = dest_offset.into_u64_with_overflow();
        let (len, len_overflow) = len.into_u64_with_overflow();
        let (end, end_overflow) = max(src_offset, dest_offset).overflowing_add(len);
        if src_overflow || dest_overflow || len_overflow || end_overflow {
            return Err(StatusCode::EVMC_OUT_OF_GAS);
        }
        consume_copy_cost(len, gas_left)?;
        self.expand(end, gas_left)?;
        let src_offset = src_offset as usize;
        let dest_offset = dest_offset as usize;
        let len = len as usize;
        self.0
            .copy_within(src_offset..src_offset + len, dest_offset); // + does not overflow
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use evmc_vm::StatusCode;

    use crate::types::{memory::Memory, u256};

    #[test]
    fn internals() {
        let mem = Memory::new(vec![0]);
        assert_eq!(mem.len(), 1);
        assert_eq!(mem.into_inner(), vec![0]);
    }

    #[test]
    fn expand() {
        let mut memory = Memory::new(Vec::new());
        assert_eq!(memory.expand(1, &mut 1_000), Ok(()));
        assert_eq!(memory.into_inner(), [0; 32]);

        let mut memory = Memory::new(Vec::new());
        assert_eq!(memory.expand(32, &mut 1_000), Ok(()));
        assert_eq!(memory.into_inner(), [0; 32]);

        let mut memory = Memory::new(vec![1; 32]);
        assert_eq!(memory.expand(64, &mut 1_000), Ok(()));
        assert_eq!(memory.into_inner(), {
            let mut mem = [1; 64];
            mem[32..].copy_from_slice(&[0; 32]);
            mem
        });

        let mut memory = Memory::new(Vec::new());
        assert_eq!(
            memory.expand(u64::MAX, &mut 1_000),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );
    }

    #[test]
    fn consume_expansion_cost() {
        let memory = Memory::new(Vec::new());
        let mut gas_left = 0;
        assert_eq!(memory.consume_expansion_cost(&mut gas_left, 0), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = 3;
        assert_eq!(memory.consume_expansion_cost(&mut gas_left, 1), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = 3;
        assert_eq!(memory.consume_expansion_cost(&mut gas_left, 32), Ok(()));
        assert_eq!(gas_left, 0);

        let memory = Memory::new(vec![0; 32]);
        let mut gas_left = 3;
        assert_eq!(memory.consume_expansion_cost(&mut gas_left, 64), Ok(()));
        assert_eq!(gas_left, 0);

        assert_eq!(
            memory.consume_expansion_cost(&mut 10_000, u64::MAX),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        assert_eq!(
            memory.consume_expansion_cost(&mut 10_000, u64::MAX / 100),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );
    }

    #[test]
    fn get_mut_slice() {
        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 0, &mut gas),
            Ok([].as_mut_slice())
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 1, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 3;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 1, &mut gas),
            Ok([0].as_mut_slice())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(Vec::new());
        let mut gas = 3;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32, &mut gas),
            Ok([0; 32].as_mut_slice())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(Vec::new());
        let mut gas = 6;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32 + 1, &mut gas),
            Ok([0; 32 + 1].as_mut_slice())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 0;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 1, &mut gas),
            Ok([1].as_mut_slice())
        );

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 0;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32, &mut gas),
            Ok([1; 32].as_mut_slice())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 3;
        let mut result = [1; 32 + 1];
        result[32] = 0;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32 + 1, &mut gas),
            Ok(result.as_mut_slice())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32 * 2]);
        let mut gas = 3;
        let mut result = [1; 32 * 2];
        result[32..].copy_from_slice(&[0; 32]);
        assert_eq!(
            mem.get_mut_slice(32u8.into(), 32 * 2, &mut gas),
            Ok(result.as_mut_slice())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(Vec::new());
        let mut gas = 1_000_000;
        assert_eq!(
            mem.get_mut_slice(u256::MAX, 1, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );
    }

    #[test]
    fn get_word() {
        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.get_word(u256::ZERO, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 3;
        assert_eq!(mem.get_word(u256::ZERO, &mut gas), Ok(u256::ZERO));
        assert_eq!(gas, 0);

        let mut mem = Memory::new(Vec::new());
        let mut gas = 6;
        assert_eq!(mem.get_word(u256::ONE, &mut gas), Ok(u256::ZERO));
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![0xff; 32]);
        let mut gas = 0;
        assert_eq!(mem.get_word(u256::ZERO, &mut gas), Ok(u256::MAX));
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![0xff; 32]);
        let mut gas = 3;
        assert_eq!(mem.get_word(32u8.into(), &mut gas), Ok(u256::ZERO));
        assert_eq!(gas, 0);
    }

    #[test]
    fn get_byte() {
        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.get_mut_byte(u256::ZERO, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 3;
        assert_eq!(mem.get_mut_byte(u256::ZERO, &mut gas), Ok(&mut 0));
        assert_eq!(gas, 0);

        let mut mem = Memory::new(Vec::new());
        let mut gas = 6;
        assert_eq!(mem.get_mut_byte(32u8.into(), &mut gas), Ok(&mut 0));
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 0;
        assert_eq!(mem.get_mut_byte(u256::ZERO, &mut gas), Ok(&mut 1));
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 3;
        assert_eq!(mem.get_mut_byte(32u8.into(), &mut gas), Ok(&mut 0));
        assert_eq!(gas, 0);
    }

    #[test]
    fn copy_within() {
        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ZERO, &mut gas),
            Ok(())
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.copy_within(u256::ONE, u256::ZERO, u256::ZERO, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ONE, u256::ZERO, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 0;
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ONE, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 1_000_000;
        assert_eq!(
            mem.copy_within(u256::MAX, u256::ZERO, u256::ZERO, &mut gas),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );

        let mut mem = Memory::new(Vec::new());
        let mut gas = 3 + 3;
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ONE, &mut gas),
            Ok(())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 3;
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ONE, &mut gas),
            Ok(())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 3 + 6;
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, 33u8.into(), &mut gas),
            Ok(())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 3 + 3;
        assert_eq!(
            mem.copy_within(32u8.into(), u256::ZERO, u256::ONE, &mut gas),
            Ok(())
        );
        assert_eq!(gas, 0);

        let mut mem = Memory::new(vec![1; 32]);
        let mut gas = 3 + 3;
        assert_eq!(
            mem.copy_within(u256::ZERO, 32u8.into(), u256::ONE, &mut gas),
            Ok(())
        );
        assert_eq!(gas, 0);
    }
}

#[cfg(feature = "alloc-reuse")]
use std::sync::Mutex;
use std::{cmp::max, iter};

use crate::{
    types::{u256, FailStatus},
    utils::{word_size, Gas},
};

#[cfg(feature = "alloc-reuse")]
static REUSABLE_MEMORY: Mutex<Vec<Vec<u8>>> = Mutex::new(Vec::new());

#[derive(Debug)]
pub struct Memory(Vec<u8>);

#[cfg(feature = "alloc-reuse")]
impl Drop for Memory {
    fn drop(&mut self) {
        let mut memory = Vec::new();
        std::mem::swap(&mut memory, &mut self.0);
        REUSABLE_MEMORY.lock().unwrap().push(memory);
    }
}

impl Memory {
    pub fn new(memory: &[u8]) -> Self {
        #[cfg(not(feature = "alloc-reuse"))]
        let mut m = Vec::new();
        #[cfg(feature = "alloc-reuse")]
        let mut m = REUSABLE_MEMORY.lock().unwrap().pop().unwrap_or_default();
        m.clear();

        m.extend_from_slice(memory);
        Self(m)
    }

    pub fn as_slice(&self) -> &[u8] {
        self.0.as_slice()
    }

    pub fn len(&self) -> u64 {
        self.0.len() as u64
    }

    fn expand(&mut self, new_len_bytes: u64, gas_left: &mut Gas) -> Result<(), FailStatus> {
        #[cold]
        fn expand_raw(m: &mut Memory, new_len: u64, gas_left: &mut Gas) -> Result<(), FailStatus> {
            let current_len = m.0.len() as u64;
            m.consume_expansion_cost(new_len, gas_left)?;
            m.0.extend(iter::repeat(0).take((new_len - current_len) as usize));
            Ok(())
        }

        let current_len = self.0.len() as u64;
        let new_len = word_size(new_len_bytes)? * 32; // word_size just did a division by 32 so * will not overflow
        if new_len > current_len {
            expand_raw(self, new_len, gas_left)?;
        }
        Ok(())
    }

    fn consume_expansion_cost(&self, new_len: u64, gas_left: &mut Gas) -> Result<(), FailStatus> {
        fn memory_cost(size: u64) -> Result<u64, FailStatus> {
            let word_size = word_size(size)?;
            let (pow2, pow2_overflow) = word_size.overflowing_pow(2);
            let (word_size_3, word_size_3_overflow) = word_size.overflowing_mul(3);
            let (cost, cost_overflow) = (pow2 / 512).overflowing_add(word_size_3);
            if pow2_overflow || word_size_3_overflow || cost_overflow {
                return Err(FailStatus::OutOfGas);
            };
            Ok(cost)
        }

        let current_len = self.0.len() as u64;

        if new_len > current_len {
            let memory_expansion_cost = memory_cost(new_len)? - memory_cost(current_len)?;
            gas_left.consume(memory_expansion_cost)?;
        }
        Ok(())
    }

    pub fn get_mut_slice(
        &mut self,
        offset: u256,
        len: u64,
        gas_left: &mut Gas,
    ) -> Result<&mut [u8], FailStatus> {
        if len == 0 {
            return Ok(&mut []);
        }
        let (offset, offset_overflow) = offset.into_u64_with_overflow();
        let (end, end_overflow) = offset.overflowing_add(len);
        if offset_overflow || end_overflow {
            return Err(FailStatus::OutOfGas);
        }
        self.expand(end, gas_left)?;

        let offset = offset as usize;
        let end = end as usize;
        unsafe {
            std::hint::assert_unchecked(offset < end && end <= self.0.len());
        }
        Ok(&mut self.0[offset..end])
    }

    pub fn get_word(&mut self, offset: u256, gas_left: &mut Gas) -> Result<u256, FailStatus> {
        let slice = self.get_mut_slice(offset, 32, gas_left)?;
        // SAFETY:
        // The slice is 32 bytes long.
        let slice = unsafe { &*(slice.as_ptr() as *const [u8; 32]) };
        Ok(u256::from_be_bytes(*slice))
    }

    pub fn get_mut_byte(
        &mut self,
        offset: u256,
        gas_left: &mut Gas,
    ) -> Result<&mut u8, FailStatus> {
        let slice = self.get_mut_slice(offset, 1, gas_left)?;
        Ok(&mut slice[0])
    }

    pub fn copy_within(
        &mut self,
        src_offset: u256,
        dest_offset: u256,
        len: u256,
        gas_left: &mut Gas,
    ) -> Result<(), FailStatus> {
        let (src_offset, src_overflow) = src_offset.into_u64_with_overflow();
        let (dest_offset, dest_overflow) = dest_offset.into_u64_with_overflow();
        let (len, len_overflow) = len.into_u64_with_overflow();
        let (end, end_overflow) = max(src_offset, dest_offset).overflowing_add(len);
        if src_overflow || dest_overflow || len_overflow || end_overflow {
            return Err(FailStatus::OutOfGas);
        }
        gas_left.consume_copy_cost(len)?;
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
    use crate::{
        types::{memory::Memory, u256, FailStatus},
        utils::Gas,
    };

    #[test]
    fn internals() {
        let mem = Memory::new(&[0]);
        assert_eq!(mem.len(), 1);
        assert_eq!(mem.as_slice(), [0]);
    }

    #[test]
    fn expand() {
        let mut memory = Memory::new(&[]);
        assert_eq!(memory.expand(1, &mut Gas::new(1_000)), Ok(()));
        assert_eq!(memory.as_slice(), [0; 32]);

        let mut memory = Memory::new(&[]);
        assert_eq!(memory.expand(32, &mut Gas::new(1_000)), Ok(()));
        assert_eq!(memory.as_slice(), [0; 32]);

        let mut memory = Memory::new(&[1; 32]);
        assert_eq!(memory.expand(64, &mut Gas::new(1_000)), Ok(()));
        assert_eq!(memory.as_slice(), {
            let mut mem = [1; 64];
            mem[32..].copy_from_slice(&[0; 32]);
            mem
        });

        let mut memory = Memory::new(&[]);
        assert_eq!(
            memory.expand(u64::MAX, &mut Gas::new(1_000)),
            Err(FailStatus::OutOfGas)
        );
    }

    #[test]
    fn consume_expansion_cost() {
        let memory = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(memory.consume_expansion_cost(0, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = Gas::new(3);
        assert_eq!(memory.consume_expansion_cost(1, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = Gas::new(3);
        assert_eq!(memory.consume_expansion_cost(32, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 0);

        let memory = Memory::new(&[0; 32]);
        let mut gas_left = Gas::new(3);
        assert_eq!(memory.consume_expansion_cost(64, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 0);

        assert_eq!(
            memory.consume_expansion_cost(u64::MAX, &mut Gas::new(10_000)),
            Err(FailStatus::OutOfGas)
        );

        assert_eq!(
            memory.consume_expansion_cost(u64::MAX / 100, &mut Gas::new(10_000)),
            Err(FailStatus::OutOfGas)
        );
    }

    #[test]
    fn get_mut_slice() {
        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 0, &mut gas_left),
            Ok([].as_mut_slice())
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 1, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(3);
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 1, &mut gas_left),
            Ok([0].as_mut_slice())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(3);
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32, &mut gas_left),
            Ok([0; 32].as_mut_slice())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(6);
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32 + 1, &mut gas_left),
            Ok([0; 32 + 1].as_mut_slice())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 1, &mut gas_left),
            Ok([1].as_mut_slice())
        );

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32, &mut gas_left),
            Ok([1; 32].as_mut_slice())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(3);
        let mut result = [1; 32 + 1];
        result[32] = 0;
        assert_eq!(
            mem.get_mut_slice(u256::ZERO, 32 + 1, &mut gas_left),
            Ok(result.as_mut_slice())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32 * 2]);
        let mut gas_left = Gas::new(3);
        let mut result = [1; 32 * 2];
        result[32..].copy_from_slice(&[0; 32]);
        assert_eq!(
            mem.get_mut_slice(32u8.into(), 32 * 2, &mut gas_left),
            Ok(result.as_mut_slice())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(1_000_000);
        assert_eq!(
            mem.get_mut_slice(u256::MAX, 1, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );
    }

    #[test]
    fn get_word() {
        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.get_word(u256::ZERO, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(3);
        assert_eq!(mem.get_word(u256::ZERO, &mut gas_left), Ok(u256::ZERO));
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(6);
        assert_eq!(mem.get_word(u256::ONE, &mut gas_left), Ok(u256::ZERO));
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[0xff; 32]);
        let mut gas_left = Gas::new(0);
        assert_eq!(mem.get_word(u256::ZERO, &mut gas_left), Ok(u256::MAX));
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[0xff; 32]);
        let mut gas_left = Gas::new(3);
        assert_eq!(mem.get_word(32u8.into(), &mut gas_left), Ok(u256::ZERO));
        assert_eq!(gas_left, 0);
    }

    #[test]
    fn get_byte() {
        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.get_mut_byte(u256::ZERO, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(3);
        assert_eq!(mem.get_mut_byte(u256::ZERO, &mut gas_left), Ok(&mut 0));
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(6);
        assert_eq!(mem.get_mut_byte(32u8.into(), &mut gas_left), Ok(&mut 0));
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(0);
        assert_eq!(mem.get_mut_byte(u256::ZERO, &mut gas_left), Ok(&mut 1));
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(3);
        assert_eq!(mem.get_mut_byte(32u8.into(), &mut gas_left), Ok(&mut 0));
        assert_eq!(gas_left, 0);
    }

    #[test]
    fn copy_within() {
        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ZERO, &mut gas_left),
            Ok(())
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.copy_within(u256::ONE, u256::ZERO, u256::ZERO, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ONE, u256::ZERO, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(0);
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ONE, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(1_000_000);
        assert_eq!(
            mem.copy_within(u256::MAX, u256::ZERO, u256::ZERO, &mut gas_left),
            Err(FailStatus::OutOfGas)
        );

        let mut mem = Memory::new(&[]);
        let mut gas_left = Gas::new(3 + 3);
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ONE, &mut gas_left),
            Ok(())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(3);
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, u256::ONE, &mut gas_left),
            Ok(())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(3 + 6);
        assert_eq!(
            mem.copy_within(u256::ZERO, u256::ZERO, 33u8.into(), &mut gas_left),
            Ok(())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(3 + 3);
        assert_eq!(
            mem.copy_within(32u8.into(), u256::ZERO, u256::ONE, &mut gas_left),
            Ok(())
        );
        assert_eq!(gas_left, 0);

        let mut mem = Memory::new(&[1; 32]);
        let mut gas_left = Gas::new(3 + 3);
        assert_eq!(
            mem.copy_within(u256::ZERO, 32u8.into(), u256::ONE, &mut gas_left),
            Ok(())
        );
        assert_eq!(gas_left, 0);
    }
}

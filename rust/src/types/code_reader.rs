use std::{
    cmp::min,
    mem::{self},
    ops::Deref,
};

use crate::types::{u256, CodeByteType, FailStatus, JumpAnalysis, Opcode};

#[derive(Debug, Clone, Copy)]
pub struct PushLen(usize);

impl PushLen {
    pub const fn new(len: usize) -> Self {
        assert!(len > 0 && len <= 32);
        Self(len)
    }

    pub const fn value(self) -> usize {
        self.0
    }
}

#[derive(Debug)]
pub struct CodeReader<'a> {
    code: &'a [u8],
    jump_analysis: JumpAnalysis,
    pc: usize,
}

impl<'a> Deref for CodeReader<'a> {
    type Target = [u8];

    fn deref(&self) -> &Self::Target {
        self.code
    }
}

#[derive(Debug, PartialEq, Eq)]
pub enum GetOpcodeError {
    OutOfRange,
    Invalid,
}

impl<'a> CodeReader<'a> {
    pub fn new(code: &'a [u8], code_hash: Option<u256>, pc: usize) -> Self {
        Self {
            code,
            jump_analysis: JumpAnalysis::new(code, code_hash),
            pc,
        }
    }

    pub fn get(&self) -> Result<Opcode, GetOpcodeError> {
        if let Some(op) = self.code.get(self.pc) {
            #[cfg(not(feature = "no-bounds-checks"))]
            let analysis = self.jump_analysis[self.pc];
            #[cfg(feature = "no-bounds-checks")]
            // SAFETY:
            // self.code and self.jump_analysis have the same length. Because self.pc <
            // self.code.len() this also holds for self.jump_analysis.
            let analysis = unsafe { *self.jump_analysis.get_unchecked(self.pc) };
            if analysis == CodeByteType::DataOrInvalid {
                Err(GetOpcodeError::Invalid)
            } else {
                // SAFETY:
                // [Opcode] has repr(u8) and therefore the same memory layout as u8.
                // In get_code_byte_types this byte of the code was determined to be a valid opcode.
                // Therefore the value is a valid [Opcode].
                let op = unsafe { mem::transmute::<u8, Opcode>(*op) };
                Ok(op)
            }
        } else {
            Err(GetOpcodeError::OutOfRange)
        }
    }

    pub fn next(&mut self) {
        self.pc += 1;
    }

    pub fn try_jump(&mut self, dest: u256) -> Result<(), FailStatus> {
        let dest = u64::try_from(dest).map_err(|_| FailStatus::BadJumpDestination)? as usize;
        if dest >= self.jump_analysis.len() || self.jump_analysis[dest] != CodeByteType::JumpDest {
            return Err(FailStatus::BadJumpDestination);
        }
        self.pc = dest;

        Ok(())
    }

    pub fn get_push_data(&mut self, push_len: PushLen) -> u256 {
        let len = min(push_len.value(), self.code.len().saturating_sub(self.pc));
        let mut data = u256::ZERO;
        if len > 0 {
            #[cfg(not(feature = "no-bounds-checks"))]
            let dest = &mut data[32 - len..];
            #[cfg(feature = "no-bounds-checks")]
            // SAFETY:
            // Because push_len <= 32 so is len, which means the index is always in bounds.
            let dest = unsafe { data.get_unchecked_mut(32 - len..) };
            #[cfg(not(feature = "no-bounds-checks"))]
            let src = &self.code[self.pc..self.pc + len];
            #[cfg(feature = "no-bounds-checks")]
            // SAFETY:
            // - len > 0
            // - because push_len <= 32 so is len
            // - self.pc + len will not overflow because self.code can never be that large because
            //   we would run out of memory before. Therefore, self.pc < self.pc + len.
            // - len <= self.code.len().saturating_sub(self.pc) which also means self.pc + len <=
            //   self.code.len()
            // Therefore, the index is always withing bounds.
            let src = unsafe { self.code.get_unchecked(self.pc..self.pc + len) };
            dest.copy_from_slice(src);
        }
        self.pc += len;

        data
    }

    pub fn pc(&self) -> usize {
        self.pc
    }
}

#[cfg(test)]
mod tests {
    use crate::types::{
        code_reader::{CodeReader, GetOpcodeError},
        u256, FailStatus, Opcode, PushLen,
    };

    #[test]
    fn code_reader_internals() {
        let code = [Opcode::Add as u8, Opcode::Add as u8, 0xc0];
        let pc = 1;
        let code_reader = CodeReader::new(&code, None, pc);
        assert_eq!(*code_reader, code);
        assert_eq!(code_reader.len(), code.len());
        assert_eq!(code_reader.pc(), pc);
    }

    #[test]
    fn code_reader_get() {
        let mut code_reader =
            CodeReader::new(&[Opcode::Add as u8, Opcode::Add as u8, 0xc0], None, 0);
        assert_eq!(code_reader.get(), Ok(Opcode::Add));
        code_reader.next();
        assert_eq!(code_reader.get(), Ok(Opcode::Add));
        code_reader.next();
        assert_eq!(code_reader.get(), Err(GetOpcodeError::Invalid));
        code_reader.next();
        assert_eq!(code_reader.get(), Err(GetOpcodeError::OutOfRange));
    }

    #[test]
    fn code_reader_try_jump() {
        let mut code_reader = CodeReader::new(
            &[
                Opcode::Push1 as u8,
                Opcode::JumpDest as u8,
                Opcode::JumpDest as u8,
            ],
            None,
            0,
        );
        assert_eq!(
            code_reader.try_jump(1u8.into()),
            Err(FailStatus::BadJumpDestination)
        );
        assert_eq!(code_reader.try_jump(2u8.into()), Ok(()));
        assert_eq!(
            code_reader.try_jump(3u8.into()),
            Err(FailStatus::BadJumpDestination)
        );
        assert_eq!(
            code_reader.try_jump(u256::MAX),
            Err(FailStatus::BadJumpDestination)
        );
    }

    #[test]
    #[should_panic]
    fn push_len_new_0() {
        PushLen::new(0);
    }

    #[test]
    #[should_panic]
    fn push_len_new_33() {
        PushLen::new(33);
    }

    #[test]
    fn push_len_new() {
        for i in 1..=32 {
            assert_eq!(PushLen::new(i).value(), i);
        }
    }

    #[test]
    fn code_reader_get_push_data() {
        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(PushLen::new(1)), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), u256::MAX);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 31);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 32);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), u256::ZERO);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 33);
        assert_eq!(code_reader.get_push_data(PushLen::new(32)), u256::ZERO);
    }
}

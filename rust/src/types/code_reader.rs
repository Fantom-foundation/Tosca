use std::{cmp::min, ops::Deref};

use crate::types::{u256, AnalysisContainer, CodeAnalysis, CodeByteType, FailStatus, Opcode};

#[derive(Debug)]
pub struct CodeReader<'a> {
    code: &'a [u8],
    code_analysis: AnalysisContainer<CodeAnalysis>,
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
            code_analysis: CodeAnalysis::new(code, code_hash),
            pc,
        }
    }

    pub fn get(&self) -> Result<Opcode, GetOpcodeError> {
        if let Some(op) = self.code.get(self.pc) {
            let analysis = self.code_analysis.analysis[self.pc];
            if analysis == CodeByteType::DataOrInvalid {
                Err(GetOpcodeError::Invalid)
            } else {
                // SAFETY:
                // [Opcode] has repr(u8) and therefore the same memory layout as u8.
                // In get_code_byte_types this byte of the code was determined to be a valid opcode.
                // Therefore the value is a valid [Opcode].
                let op = unsafe { std::mem::transmute::<u8, Opcode>(*op) };
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
        if dest >= self.code_analysis.analysis.len()
            || self.code_analysis.analysis[dest] != CodeByteType::JumpDest
        {
            return Err(FailStatus::BadJumpDestination);
        }
        self.pc = dest;

        Ok(())
    }

    pub fn get_push_data(&mut self, len: usize) -> u256 {
        assert!(len <= 32);

        let data_len = min(len, self.code.len().saturating_sub(self.pc));
        let mut data = u256::ZERO;
        data[32 - len..32 - len + data_len]
            .copy_from_slice(&self.code[self.pc..self.pc + data_len]);
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
        u256, FailStatus, Opcode,
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
    fn code_reader_get_push_data() {
        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(0u8.into()), u256::ZERO);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(1u8.into()), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 0);
        assert_eq!(code_reader.get_push_data(32u8.into()), u256::MAX);

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 31);
        assert_eq!(
            code_reader.get_push_data(32u8.into()),
            u256::from(0xffu8) << u256::from(248u8)
        );

        let mut code_reader = CodeReader::new(&[0xff; 32], None, 32);
        assert_eq!(code_reader.get_push_data(32u8.into()), u256::ZERO);
    }
}

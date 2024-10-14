use std::{cmp::min, mem, ops::Deref};

use evmc_vm::StatusCode;

use crate::types::{code_byte_type, u256, CodeByteType, Opcode};

#[derive(Debug)]
pub struct CodeReader<'a> {
    code: &'a [u8],
    code_byte_type: Box<[CodeByteType]>,
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
    pub fn new(code: &'a [u8], pc: usize) -> Self {
        Self {
            code,
            code_byte_type: compute_code_byte_types(code),
            pc,
        }
    }

    pub fn get(&self) -> Result<Opcode, GetOpcodeError> {
        if self.pc >= self.code.len() {
            Err(GetOpcodeError::OutOfRange)
        } else if self.code_byte_type[self.pc] == CodeByteType::DataOrInvalid {
            Err(GetOpcodeError::Invalid)
        } else {
            let op = self.code[self.pc];
            // SAFETY:
            // [Opcode] has repr(u8) and therefore the same memory layout as u8.
            // In get_code_byte_types this byte of the code was determined to be a valid opcode.
            // Therefore the value is a valid [Opcode].
            let op = unsafe { mem::transmute::<u8, Opcode>(op) };
            Ok(op)
        }
    }

    pub fn next(&mut self) {
        self.pc += 1;
    }

    pub fn try_jump(&mut self, dest: u256) -> Result<(), StatusCode> {
        let dest = u64::try_from(dest).map_err(|_| StatusCode::EVMC_BAD_JUMP_DESTINATION)? as usize;
        if dest >= self.code_byte_type.len() || self.code_byte_type[dest] != CodeByteType::JumpDest
        {
            return Err(StatusCode::EVMC_BAD_JUMP_DESTINATION);
        }
        self.pc = dest;

        Ok(())
    }

    pub fn get_push_data(&mut self, len: usize) -> u256 {
        assert!(len <= 32);

        let len = min(len, self.code.len() - self.pc);
        let mut data = u256::ZERO;
        data[32 - len..].copy_from_slice(&self.code[self.pc..self.pc + len]);
        self.pc += len;

        data
    }

    pub fn pc(&self) -> usize {
        self.pc
    }
}

fn compute_code_byte_types(code: &[u8]) -> Box<[CodeByteType]> {
    let mut code_byte_types = vec![CodeByteType::DataOrInvalid; code.len()];

    let mut pc = 0;
    while pc < code.len() {
        let (code_byte_type, inc) = code_byte_type(code[pc]);
        code_byte_types[pc] = code_byte_type;
        pc += inc;
    }

    code_byte_types.into_boxed_slice()
}

#[cfg(test)]
mod tests {
    use evmc_vm::StatusCode;

    use crate::types::{
        code_reader::{compute_code_byte_types, CodeReader, GetOpcodeError},
        u256, CodeByteType, Opcode,
    };

    #[test]
    fn compute_code_byte_types_single_byte() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Add as u8]),
            [CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push2 as u8]),
            [CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8]),
            [CodeByteType::JumpDest]
        );
        assert_eq!(
            *compute_code_byte_types(&[0xc0]),
            [CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn compute_byte_types_jumpdest() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8, Opcode::Add as u8]),
            [CodeByteType::JumpDest, CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8, 0xc0]),
            [CodeByteType::JumpDest, CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn compute_code_byte_types_push_with_data() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push1 as u8, Opcode::Add as u8, Opcode::Add as u8]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push1 as u8, Opcode::Add as u8, 0xc0]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                0xc0,
                Opcode::Add as u8
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                0xc0
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
    }

    #[test]
    fn code_reader_internals() {
        let code = [Opcode::Add as u8, Opcode::Add as u8, 0xc0];
        let pc = 1;
        let code_reader = CodeReader::new(&code, pc);
        assert_eq!(*code_reader, code);
        assert_eq!(code_reader.len(), code.len());
        assert_eq!(code_reader.pc(), pc);
    }

    #[test]
    fn code_reader_get() {
        let mut code_reader = CodeReader::new(&[Opcode::Add as u8, Opcode::Add as u8, 0xc0], 0);
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
            0,
        );
        assert_eq!(
            code_reader.try_jump(1u8.into()),
            Err(StatusCode::EVMC_BAD_JUMP_DESTINATION)
        );
        assert_eq!(code_reader.try_jump(2u8.into()), Ok(()));
        assert_eq!(
            code_reader.try_jump(3u8.into()),
            Err(StatusCode::EVMC_BAD_JUMP_DESTINATION)
        );
        assert_eq!(
            code_reader.try_jump(u256::MAX),
            Err(StatusCode::EVMC_BAD_JUMP_DESTINATION)
        );
    }

    #[test]
    fn code_reader_get_push_data() {
        let mut code_reader = CodeReader::new(&[0xff; 32], 0);
        assert_eq!(code_reader.get_push_data(0u8.into()), u256::ZERO);

        let mut code_reader = CodeReader::new(&[0xff; 32], 0);
        assert_eq!(code_reader.get_push_data(1u8.into()), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], 0);
        assert_eq!(code_reader.get_push_data(32u8.into()), u256::MAX);

        let mut code_reader = CodeReader::new(&[0xff; 32], 31);
        assert_eq!(code_reader.get_push_data(32u8.into()), 0xffu8.into());

        let mut code_reader = CodeReader::new(&[0xff; 32], 32);
        assert_eq!(code_reader.get_push_data(32u8.into()), u256::ZERO);
    }
}

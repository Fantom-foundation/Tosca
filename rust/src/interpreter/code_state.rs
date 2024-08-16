use std::{cmp::min, mem, ops::Deref};

use evmc_vm::{StatusCode, StepStatusCode};

use crate::types::{code_byte_type, u256, CodeByteType, Opcode};

#[derive(Debug)]
pub struct CodeState<'a> {
    code: &'a [u8],
    code_byte_type: Box<[CodeByteType]>,
    pc: usize,
}

impl<'a> Deref for CodeState<'a> {
    type Target = [u8];

    fn deref(&self) -> &Self::Target {
        self.code
    }
}

#[derive(Debug)]
pub enum GetOpcodeError {
    OutOfRange,
    Invalid,
}

impl<'a> CodeState<'a> {
    pub fn new(code: &'a [u8], pc: usize) -> Self {
        let code_byte_type = code_byte_types(code);
        Self {
            code,
            code_byte_type,
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
            let op = unsafe {
                // SAFETY:
                // [Opcode] has repr(u8) and therefore the same memory layout as u8.
                // In get_code_byte_types this byte of the code was determined to be a valid opcode.
                // Therefore the value is a valid enum variant.
                mem::transmute::<u8, Opcode>(op)
            };
            Ok(op)
        }
    }

    pub fn next(&mut self) {
        self.pc += 1;
    }

    pub fn try_jump(&mut self, dest: u256) -> Result<(), (StepStatusCode, StatusCode)> {
        let (dest, dest_overflow) = dest.into_u64_with_overflow();
        let dest = dest as usize;
        if dest_overflow
            || dest >= self.code_byte_type.len()
            || self.code_byte_type[dest] != CodeByteType::JumpDest
        {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_BAD_JUMP_DESTINATION,
            ));
        }
        self.pc = dest;

        Ok(())
    }

    pub fn get_push_data(&mut self, len: usize) -> Result<u256, (StepStatusCode, StatusCode)> {
        assert!(len <= 32);

        let len = min(len, self.code.len() - self.pc);
        let mut bytes = [0; 32];
        bytes[32 - len..].copy_from_slice(&self.code[self.pc..self.pc + len]);
        self.pc += len;

        Ok(bytes.into())
    }

    pub fn pc(&self) -> usize {
        self.pc
    }

    pub fn code_len(&self) -> usize {
        self.code.len()
    }
}

fn code_byte_types(code: &[u8]) -> Box<[CodeByteType]> {
    let mut jump_destinations = vec![CodeByteType::DataOrInvalid; code.len()];

    let mut pc = 0;
    while pc < code.len() {
        let (code_byte_type, inc) = code_byte_type(code[pc]);
        jump_destinations[pc] = code_byte_type;
        pc += inc;
    }

    jump_destinations.into_boxed_slice()
}

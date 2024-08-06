use bnum::types::U256;
use evmc_vm::{StatusCode, StepStatusCode};

use crate::types::{opcode, u256};

pub struct CodeState<'a> {
    code: &'a [u8],
    jump_destinations: Box<[JumpFlag]>,
    pc: usize,
}

impl<'a> CodeState<'a> {
    pub fn new(code: &'a [u8], pc: usize) -> Self {
        let jump_destinations = get_jump_destinations(code);
        Self {
            code,
            jump_destinations,
            pc,
        }
    }

    pub fn get(&self) -> Option<u8> {
        if self.pc >= self.code.len() {
            return None;
        }
        Some(self.code[self.pc])
    }

    pub fn next(&mut self) {
        self.pc += 1;
    }

    pub fn try_jump(&mut self, dest: u256) -> Result<(), (StepStatusCode, StatusCode)> {
        let dest_full = U256::from(dest);
        let dest = dest_full.digits()[0] as usize;
        if dest_full > u64::MAX.into() // If the destination does not fit into u64 it is definitely too large
            || dest >= self.jump_destinations.len()
            || self.jump_destinations[dest] != JumpFlag::JumpDest
        {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_BAD_JUMP_DESTINATION,
            ));
        }
        self.pc = dest;

        Ok(())
    }

    pub fn try_get_push_data(&mut self, len: usize) -> Result<&[u8], (StepStatusCode, StatusCode)> {
        if self.pc + len >= self.code.len() {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_INTERNAL_ERROR,
            ));
        }

        self.pc += 1;
        let data = &self.code[self.pc..self.pc + len];
        self.pc += len;

        Ok(data)
    }

    pub fn pc(&self) -> usize {
        self.pc
    }

    pub fn code_len(&self) -> usize {
        self.code.len()
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum JumpFlag {
    JumpDest,
    NonJumpDest,
}

fn get_jump_destinations(code: &[u8]) -> Box<[JumpFlag]> {
    let mut jump_destinations = vec![JumpFlag::NonJumpDest; code.len()];

    let mut pc = 0;
    while pc < code.len() {
        match code[pc] {
            opcode::PUSH1 => pc += 2,
            opcode::PUSH2 => pc += 3,
            opcode::PUSH3 => pc += 4,
            opcode::PUSH4 => pc += 5,
            opcode::PUSH5 => pc += 6,
            opcode::PUSH6 => pc += 7,
            opcode::PUSH7 => pc += 8,
            opcode::PUSH8 => pc += 9,
            opcode::PUSH9 => pc += 10,
            opcode::PUSH10 => pc += 11,
            opcode::PUSH11 => pc += 12,
            opcode::PUSH12 => pc += 13,
            opcode::PUSH13 => pc += 14,
            opcode::PUSH14 => pc += 15,
            opcode::PUSH15 => pc += 16,
            opcode::PUSH16 => pc += 17,
            opcode::PUSH17 => pc += 18,
            opcode::PUSH18 => pc += 19,
            opcode::PUSH19 => pc += 20,
            opcode::PUSH20 => pc += 21,
            opcode::PUSH21 => pc += 22,
            opcode::PUSH22 => pc += 23,
            opcode::PUSH23 => pc += 24,
            opcode::PUSH24 => pc += 25,
            opcode::PUSH25 => pc += 26,
            opcode::PUSH26 => pc += 27,
            opcode::PUSH27 => pc += 28,
            opcode::PUSH28 => pc += 29,
            opcode::PUSH29 => pc += 30,
            opcode::PUSH30 => pc += 31,
            opcode::PUSH31 => pc += 32,
            opcode::PUSH32 => pc += 33,
            opcode::JUMPDEST => {
                jump_destinations[pc] = JumpFlag::JumpDest;
                pc += 1;
            }
            _ => pc += 1,
        }
    }

    jump_destinations.into_boxed_slice()
}

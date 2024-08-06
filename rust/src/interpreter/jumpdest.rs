use crate::types::opcode;

pub fn get_jump_destinations(code: &[u8]) -> Box<[usize]> {
    let mut jump_destinations = Vec::new();

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
                jump_destinations.push(pc);
                pc += 1;
            }
            _ => pc += 1,
        }
    }

    jump_destinations.into_boxed_slice()
}

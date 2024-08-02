use std::mem;

use evmc_vm::{
    ExecutionContext, ExecutionMessage, Revision, StatusCode, StepResult, StepStatusCode,
};

use crate::types::{opcode, u256};

pub fn run(
    revision: Revision,
    code: &[u8],
    message: &ExecutionMessage,
    context: Option<&mut ExecutionContext>,
    mut step_status_code: StepStatusCode,
    mut pc: usize,
    mut gas_refund: i64,
    mut stack: Vec<u256>,
    mut memory: Vec<u8>,
    mut last_call_return_data: Option<Vec<u8>>,
    steps: Option<i32>,
) -> Result<StepResult, (StepStatusCode, StatusCode)> {
    let mut gas_left = message.gas();
    let mut status_code = StatusCode::EVMC_SUCCESS;
    let mut output = None;

    println!("running test");
    for _ in 0..steps.unwrap_or(i32::MAX) {
        if pc >= code.len() {
            return Err((StepStatusCode::EVMC_STEP_FAILED, StatusCode::EVMC_FAILURE));
        }
        match code[pc] {
            //} unsafe { mem::transmute::<u8, Opcode>(code[pc]) } {
            opcode::STOP => {
                step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                status_code = StatusCode::EVMC_SUCCESS;
                break;
            }
            opcode::ADD => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                *stack.last_mut().unwrap() += top;
                pc += 1;
            }
            opcode::MUL => {
                check_out_of_gas::<5>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 5;
                let top = stack.pop().unwrap();
                *stack.last_mut().unwrap() *= top;
                pc += 1;
            }
            opcode::SUB => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = top - *top2;
                pc += 1;
            }
            opcode::DIV => {
                check_out_of_gas::<5>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 5;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = top / *top2;
                pc += 1;
            }
            opcode::SDIV => {
                check_out_of_gas::<5>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 5;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = top.sdiv(*top2);
                pc += 1;
            }
            opcode::MOD => {
                check_out_of_gas::<5>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 5;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = top % *top2;
                pc += 1;
            }
            opcode::SMOD => {
                check_out_of_gas::<5>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 5;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = top.srem(*top2);
                pc += 1;
            }
            opcode::ADDMOD => {
                check_out_of_gas::<8>(&mut gas_left)?;
                check_stack_underflow::<3>(&stack)?;
                gas_left -= 8;
                let top = stack.pop().unwrap();
                let top2 = stack.pop().unwrap();
                let top3 = stack.last_mut().unwrap();
                *top3 = u256::addmod(top, top2, *top3);
                pc += 1;
            }
            opcode::MULMOD => {
                check_out_of_gas::<8>(&mut gas_left)?;
                check_stack_underflow::<3>(&stack)?;
                gas_left -= 8;
                let top = stack.pop().unwrap();
                let top2 = stack.pop().unwrap();
                let top3 = stack.last_mut().unwrap();
                *top3 = u256::mulmod(top, top2, *top3);
                pc += 1;
            }
            opcode::EXP => {
                check_out_of_gas::<10>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 10;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                let top2_bytes: [u8; 32] = (*top2).into();
                let mut cost_multiplier = 32;
                for byte in top2_bytes.into_iter() {
                    if byte == 0 {
                        cost_multiplier -= 1;
                    } else {
                        break;
                    }
                }
                let dyn_gas = 50 * cost_multiplier;
                check_dyn_out_of_gas(&mut gas_left, dyn_gas)?;
                gas_left -= dyn_gas as i64;
                *top2 = top.pow(*top2);
                pc += 1;
            }
            opcode::SIGNEXTEND => {
                check_out_of_gas::<5>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 5;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = u256::signextend(top, *top2);
                pc += 1;
            }
            opcode::LT => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = ((top < *top2) as u8).into();
                pc += 1;
            }
            opcode::GT => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = ((top > *top2) as u8).into();
                pc += 1;
            }
            opcode::SLT => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = ((top.slt(top2)) as u8).into();
                pc += 1;
            }
            opcode::SGT => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = ((top.sgt(top2)) as u8).into();
                pc += 1;
            }
            opcode::EQ => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 = ((top == *top2) as u8).into();
                pc += 1;
            }
            opcode::ISZERO => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<1>(&stack)?;
                gas_left -= 3;
                let top = stack.last_mut().unwrap();
                *top = ((*top == [0; 32].into()) as u8).into();
                pc += 1;
            }
            opcode::AND => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 &= top;
                pc += 1;
            }
            opcode::OR => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 |= top;
                pc += 1;
            }
            opcode::XOR => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<2>(&stack)?;
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let top2 = stack.last_mut().unwrap();
                *top2 ^= top;
                pc += 1;
            }
            opcode::NOT => {
                check_out_of_gas::<3>(&mut gas_left)?;
                check_stack_underflow::<1>(&stack)?;
                gas_left -= 3;
                let top = stack.last_mut().unwrap();
                *top = !*top;
                pc += 1;
            }
            opcode::PUSH0 => {
                check_min_revision(Revision::EVMC_SHANGHAI, revision)?;
                check_out_of_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;

                gas_left -= 2;
                pc += 1;
                stack.push(0.into());
            }
            opcode::PUSH1 => push::<1>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH2 => push::<2>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH3 => push::<3>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH4 => push::<4>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH5 => push::<5>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH6 => push::<6>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH7 => push::<7>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH8 => push::<8>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH9 => push::<9>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH10 => push::<10>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH11 => push::<11>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH12 => push::<12>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH13 => push::<13>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH14 => push::<14>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH15 => push::<15>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH16 => push::<16>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH17 => push::<17>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH18 => push::<18>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH19 => push::<19>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH20 => push::<20>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH21 => push::<21>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH22 => push::<22>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH23 => push::<23>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH24 => push::<24>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH25 => push::<25>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH26 => push::<26>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH27 => push::<27>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH28 => push::<28>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH29 => push::<29>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH30 => push::<30>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH31 => push::<31>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::PUSH32 => push::<32>(code, &mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP1 => dup::<1>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP2 => dup::<2>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP3 => dup::<3>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP4 => dup::<4>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP5 => dup::<5>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP6 => dup::<6>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP7 => dup::<7>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP8 => dup::<8>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP9 => dup::<9>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP10 => dup::<10>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP11 => dup::<11>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP12 => dup::<12>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP13 => dup::<13>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP14 => dup::<14>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP15 => dup::<15>(&mut pc, &mut stack, &mut gas_left)?,
            opcode::DUP16 => dup::<16>(&mut pc, &mut stack, &mut gas_left)?,
            op => {
                println!("invalid opcode 0x{op:x?}");
                step_status_code = StepStatusCode::EVMC_STEP_FAILED;
                status_code = StatusCode::EVMC_BAD_JUMP_DESTINATION;
                break;
            }
        }
    }

    stack.reverse();
    Ok(StepResult::new(
        step_status_code,
        status_code,
        revision,
        pc as u64,
        gas_left,
        gas_refund,
        output,
        // SAFETY
        // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
        // layout.
        unsafe { mem::transmute(stack) },
        memory,
        last_call_return_data,
    ))
}

fn push<const N: usize>(
    code: &[u8],
    pc: &mut usize,
    stack: &mut Vec<u256>,
    gas_left: &mut i64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    check_out_of_gas::<3>(gas_left)?;
    check_stack_overflow::<1>(stack)?;
    // Note: not tested by ct
    if code.len() < *pc + N {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_INTERNAL_ERROR,
        ));
    }

    *gas_left -= 3;
    *pc += 1;
    stack.push(code[*pc..*pc + N].try_into().unwrap());
    *pc += N;

    Ok(())
}

fn dup<const N: usize>(
    pc: &mut usize,
    stack: &mut Vec<u256>,
    gas_left: &mut i64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    check_out_of_gas::<3>(gas_left)?;
    check_stack_overflow::<1>(stack)?;
    check_stack_underflow::<N>(stack)?;

    *gas_left -= 3;
    *pc += 1;
    stack.push(stack[stack.len() - N]);

    Ok(())
}

#[inline(always)]
fn check_min_revision(
    min_revision: Revision,
    revision: Revision,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if revision < min_revision {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_INTERNAL_ERROR,
        ));
    }
    Ok(())
}

#[inline(always)]
fn check_out_of_gas<const NEEDED: u64>(
    gas_left: &mut i64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if *gas_left < (NEEDED as i64) {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    Ok(())
}

#[inline(always)]
fn check_dyn_out_of_gas(
    gas_left: &mut i64,
    needed: u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if *gas_left < (needed as i64) {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    Ok(())
}

#[inline(always)]
fn check_stack_overflow<const NEEDED: usize>(
    stack: &[u256],
) -> Result<(), (StepStatusCode, StatusCode)> {
    if stack.len() + NEEDED > 1024 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_OVERFLOW,
        ));
    }
    Ok(())
}

#[inline(always)]
fn check_stack_underflow<const NEEDED: usize>(
    stack: &[u256],
) -> Result<(), (StepStatusCode, StatusCode)> {
    if stack.len() < NEEDED {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_UNDERFLOW,
        ));
    }
    Ok(())
}

#![allow(dead_code)]
use std::{i32, mem};

use evmc_vm::{
    EvmcVm, ExecutionContext, ExecutionMessage, ExecutionResult, Revision, StatusCode, StepResult,
    StepStatusCode, SteppableEvmcVm, Uint256,
};

use crate::types::{opcode, u256};

mod ffi;
mod types;

#[evmc_declare::evmc_declare_vm("evmrs", "ewasm, evm", "0.1.0")]
pub struct EvmRs;

impl EvmcVm for EvmRs {
    fn init() -> Self {
        EvmRs {}
    }

    fn execute(
        &self,
        revision: Revision,
        code: &[u8],
        message: &ExecutionMessage,
        context: Option<&mut ExecutionContext>,
    ) -> ExecutionResult {
        let step_result = run(
            revision,
            code,
            message,
            context,
            StepStatusCode::EVMC_STEP_RUNNING,
            0,
            0,
            Vec::with_capacity(1024),
            Vec::new(),
            None,
            None,
        );

        step_result
            .map(Into::into)
            .unwrap_or_else(|(_, status_code)| ExecutionResult::new(status_code, 0, 0, None))
    }

    fn set_option(&mut self, _: &str, _: &str) -> Result<(), evmc_vm::SetOptionError> {
        Ok(())
    }
}

impl SteppableEvmcVm for EvmRs {
    fn step_n<'a>(
        &self,
        revision: Revision,
        code: &'a [u8],
        message: &'a ExecutionMessage,
        context: Option<&'a mut ExecutionContext<'a>>,
        step_status: StepStatusCode,
        pc: u64,
        gas_refund: i64,
        stack: &'a mut [Uint256],
        memory: &'a mut [u8],
        last_call_result_data: &'a mut [u8],
        steps: i32,
    ) -> StepResult {
        run(
            revision,
            code,
            message,
            context,
            step_status,
            pc as usize,
            gas_refund,
            // SAFETY
            // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
            // layout.
            unsafe { mem::transmute(stack.to_owned()) },
            memory.to_owned(),
            Some(last_call_result_data.to_owned()),
            Some(steps),
        )
        .unwrap_or_else(|(step_status_code, status_code)| {
            StepResult::new(
                step_status_code,
                status_code,
                revision,
                0,
                0,
                0,
                None,
                Vec::new(),
                Vec::new(),
                None,
            )
        })
    }
}

fn run(
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
                if gas_left < 3 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_OUT_OF_GAS,
                    ));
                }
                if stack.len() < 2 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_STACK_UNDERFLOW,
                    ));
                }
                gas_left -= 3;
                let top = stack.pop().unwrap();
                *stack.last_mut().unwrap() += top;
                pc += 1;
            }
            opcode::LT => {
                if gas_left < 3 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_OUT_OF_GAS,
                    ));
                }
                if stack.len() < 2 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_STACK_UNDERFLOW,
                    ));
                }
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let len = stack.len();
                stack[len - 1] = (top.lt(stack[len - 1]) as u8).into();
                pc += 1;
            }
            opcode::SLT => {
                if gas_left < 3 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_OUT_OF_GAS,
                    ));
                }
                if stack.len() < 2 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_STACK_UNDERFLOW,
                    ));
                }
                gas_left -= 3;
                let top = stack.pop().unwrap();
                let len = stack.len();
                stack[len - 1] = (top.slt(stack[len - 1]) as u8).into();
                pc += 1;
            }
            opcode::PUSH0 => {
                if revision < Revision::EVMC_SHANGHAI {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_INTERNAL_ERROR,
                    ));
                }
                if gas_left < 2 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_OUT_OF_GAS,
                    ));
                }
                if stack.len() + 1 > 1024 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_STACK_OVERFLOW,
                    ));
                }
                //if pc + 1 >= code.len() {
                //return Err((
                //StepStatusCode::EVMC_STEP_FAILED,
                //StatusCode::EVMC_INTERNAL_ERROR,
                //));
                //}

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
    if *gas_left < 3 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    if stack.len() + 1 > 1024 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_OVERFLOW,
        ));
    }
    //if pc + 1 >= code.len() {
    //return Err((
    //StepStatusCode::EVMC_STEP_FAILED,
    //StatusCode::EVMC_INTERNAL_ERROR,
    //));
    //}

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
    if *gas_left < 3 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    if stack.len() + 1 > 1024 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_OVERFLOW,
        ));
    }
    if stack.len() < N {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_UNDERFLOW,
        ));
    }

    *gas_left -= 3;
    *pc += 1;
    stack.push(stack[stack.len() - N]);

    Ok(())
}

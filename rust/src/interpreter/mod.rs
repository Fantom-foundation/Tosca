use std::{cmp::min, mem};

use evmc_vm::{
    AccessStatus, ExecutionContext, ExecutionMessage, MessageFlags, MessageKind, Revision,
    StatusCode, StepStatusCode, StorageStatus,
};
use sha3::{Digest, Keccak256};

pub use crate::interpreter::{memory::Memory, run_result::RunResult, stack::Stack};
use crate::{
    interpreter::{checks::*, code_state::GetOpcodeError, gas::*},
    types::{u256, Opcode},
};

mod checks;
mod code_state;
mod gas;
mod memory;
mod run_result;
mod stack;

pub use code_state::CodeState;

#[allow(clippy::too_many_arguments)]
pub fn run<'a>(
    revision: Revision,
    message: &ExecutionMessage,
    context: &mut ExecutionContext,
    mut step_status_code: StepStatusCode,
    mut code_state: CodeState<'a>,
    mut gas_refund: i64,
    mut stack: Stack,
    mut memory: Memory,
    mut last_call_return_data: Option<Vec<u8>>,
    mut steps: Option<i32>,
) -> Result<RunResult<'a>, (StepStatusCode, StatusCode)> {
    let mut gas_left = message.gas() as u64;
    let mut status_code = StatusCode::EVMC_SUCCESS;
    let mut output = None;

    //println!("##### running test #####");
    loop {
        match &mut steps {
            None => (),
            Some(0) => break,
            Some(steps) => *steps -= 1,
        }
        let op = match code_state.get() {
            Ok(op) => op,
            Err(GetOpcodeError::OutOfRange) => {
                step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                break;
            }
            Err(GetOpcodeError::Invalid) => {
                //println!("invalid Opcode 0x{op:x?}");
                return Err((
                    StepStatusCode::EVMC_STEP_FAILED,
                    StatusCode::EVMC_INVALID_INSTRUCTION,
                ));
            }
        };
        match op {
            Opcode::Stop => {
                step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                status_code = StatusCode::EVMC_SUCCESS;
                break;
            }
            Opcode::Add => {
                consume_gas::<3>(&mut gas_left)?;
                let [value1, value2] = stack.pop()?;
                stack.push(value1 + value2)?;
                code_state.next();
            }
            Opcode::Mul => {
                consume_gas::<5>(&mut gas_left)?;
                let [fac1, fac2] = stack.pop()?;
                stack.push(fac1 * fac2)?;
                code_state.next();
            }
            Opcode::Sub => {
                consume_gas::<3>(&mut gas_left)?;
                let [value1, value2] = stack.pop()?;
                stack.push(value1 - value2)?;
                code_state.next();
            }
            Opcode::Div => {
                consume_gas::<5>(&mut gas_left)?;
                let [value, denominator] = stack.pop()?;
                stack.push(value / denominator)?;
                code_state.next();
            }
            Opcode::SDiv => {
                consume_gas::<5>(&mut gas_left)?;
                let [value, denominator] = stack.pop()?;
                stack.push(value.sdiv(denominator))?;
                code_state.next();
            }
            Opcode::Mod => {
                consume_gas::<5>(&mut gas_left)?;
                let [value, denominator] = stack.pop()?;
                stack.push(value % denominator)?;
                code_state.next();
            }
            Opcode::SMod => {
                consume_gas::<5>(&mut gas_left)?;
                let [value, denominator] = stack.pop()?;
                stack.push(value.srem(denominator))?;
                code_state.next();
            }
            Opcode::AddMod => {
                consume_gas::<8>(&mut gas_left)?;
                let [value1, value2, denominator] = stack.pop()?;
                stack.push(u256::addmod(value1, value2, denominator))?;
                code_state.next();
            }
            Opcode::MulMod => {
                consume_gas::<8>(&mut gas_left)?;
                let [fac1, fac2, denominator] = stack.pop()?;
                stack.push(u256::mulmod(fac1, fac2, denominator))?;
                code_state.next();
            }
            Opcode::Exp => {
                consume_gas::<10>(&mut gas_left)?;
                let [value, exp] = stack.pop()?;
                let byte_size = 32 - exp.into_iter().take_while(|byte| *byte == 0).count() as u64;
                consume_dyn_gas(&mut gas_left, byte_size * 50)?; // * does not overflow
                stack.push(value.pow(exp))?;
                code_state.next();
            }
            Opcode::SignExtend => {
                consume_gas::<5>(&mut gas_left)?;
                let [size, value] = stack.pop()?;
                stack.push(u256::signextend(size, value))?;
                code_state.next();
            }
            Opcode::Lt => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs < rhs)?;
                code_state.next();
            }
            Opcode::Gt => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs > rhs)?;
                code_state.next();
            }
            Opcode::SLt => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs.slt(&rhs))?;
                code_state.next();
            }
            Opcode::SGt => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs.sgt(&rhs))?;
                code_state.next();
            }
            Opcode::Eq => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs == rhs)?;
                code_state.next();
            }
            Opcode::IsZero => {
                consume_gas::<3>(&mut gas_left)?;
                let [value] = stack.pop()?;
                stack.push(value == u256::ZERO)?;
                code_state.next();
            }
            Opcode::And => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs & rhs)?;
                code_state.next();
            }
            Opcode::Or => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs | rhs)?;
                code_state.next();
            }
            Opcode::Xor => {
                consume_gas::<3>(&mut gas_left)?;
                let [lhs, rhs] = stack.pop()?;
                stack.push(lhs ^ rhs)?;
                code_state.next();
            }
            Opcode::Not => {
                consume_gas::<3>(&mut gas_left)?;
                let [value] = stack.pop()?;
                stack.push(!value)?;
                code_state.next();
            }
            Opcode::Byte => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset, value] = stack.pop()?;
                stack.push(value.byte(offset))?;
                code_state.next();
            }
            Opcode::Shl => {
                consume_gas::<3>(&mut gas_left)?;
                let [shift, value] = stack.pop()?;
                stack.push(value << shift)?;
                code_state.next();
            }
            Opcode::Shr => {
                consume_gas::<3>(&mut gas_left)?;
                let [shift, value] = stack.pop()?;
                stack.push(value >> shift)?;
                code_state.next();
            }
            Opcode::Sar => {
                consume_gas::<3>(&mut gas_left)?;
                let [shift, value] = stack.pop()?;
                stack.push(value.sar(shift))?;
                code_state.next();
            }
            Opcode::Sha3 => {
                consume_gas::<30>(&mut gas_left)?;
                let [offset, len] = stack.pop()?;

                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    OUT_OF_GAS_ERR?;
                }
                consume_dyn_gas(&mut gas_left, 6 * word_size(len)?)?; // * does not overflow

                let data = memory.get_slice(offset, len, &mut gas_left)?;
                let mut hasher = Keccak256::new();
                hasher.update(data);
                let mut bytes = [0; 32];
                hasher.finalize_into((&mut bytes).into());
                stack.push(bytes)?;
                code_state.next();
            }
            Opcode::Address => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(message.recipient())?;
                code_state.next();
            }
            Opcode::Balance => {
                if revision < Revision::EVMC_BERLIN {
                    consume_gas::<700>(&mut gas_left)?;
                }
                let [addr] = stack.pop()?;
                let addr = addr.into();
                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                stack.push(context.get_balance(&addr))?;
                code_state.next();
            }
            Opcode::Origin => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().tx_origin)?;
                code_state.next();
            }
            Opcode::Caller => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(message.sender())?;
                code_state.next();
            }
            Opcode::CallValue => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(*message.value())?;
                code_state.next();
            }
            Opcode::CallDataLoad => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset] = stack.pop()?;
                let (offset, overflow) = offset.into_u64_with_overflow();
                let offset = offset as usize;
                let call_data = message.input().map(|v| v.as_slice()).unwrap_or(&[]);
                if overflow || offset >= call_data.len() {
                    stack.push(u256::ZERO)?;
                } else {
                    let end = min(call_data.len(), offset + 32);
                    let mut bytes = [0; 32];
                    bytes[..end - offset].copy_from_slice(&call_data[offset..end]);
                    stack.push(bytes)?;
                }
                code_state.next();
            }
            Opcode::CallDataSize => {
                consume_gas::<2>(&mut gas_left)?;
                let call_data_len = message
                    .input()
                    .map(|call_data| call_data.len())
                    .unwrap_or(0);
                stack.push(call_data_len)?;
                code_state.next();
            }
            Opcode::Push0 => {
                check_min_revision(Revision::EVMC_SHANGHAI, revision)?;
                consume_gas::<2>(&mut gas_left)?;
                stack.push(u256::ZERO)?;
                code_state.next();
            }
            Opcode::CallDataCopy => {
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = stack.pop()?;

                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        return Err((
                            StepStatusCode::EVMC_STEP_FAILED,
                            StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                        ));
                    }

                    let src = message.input().map(|v| v.as_slice()).unwrap_or(&[]);
                    let src = get_slice_within_bounds(src, offset, len);
                    let dest = memory.get_slice(dest_offset, len, &mut gas_left)?;
                    copy_slice_padded(src, dest, &mut gas_left)?;
                }

                code_state.next();
            }
            Opcode::CodeSize => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(code_state.code_len())?;
                code_state.next();
            }
            Opcode::CodeCopy => {
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = stack.pop()?;

                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        OUT_OF_GAS_ERR?;
                    }

                    let src = get_slice_within_bounds(&code_state, offset, len);
                    let dest = memory.get_slice(dest_offset, len, &mut gas_left)?;
                    copy_slice_padded(src, dest, &mut gas_left)?;
                }

                code_state.next();
            }
            Opcode::GasPrice => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().tx_gas_price)?;
                code_state.next();
            }
            Opcode::ExtCodeSize => {
                if revision < Revision::EVMC_BERLIN {
                    consume_gas::<700>(&mut gas_left)?;
                }
                let [addr] = stack.pop()?;
                let addr = addr.into();
                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                stack.push(context.get_code_size(&addr))?;
                code_state.next();
            }
            Opcode::ExtCodeCopy => {
                if revision < Revision::EVMC_BERLIN {
                    consume_gas::<700>(&mut gas_left)?;
                }
                let [addr, dest_offset, offset, len] = stack.pop()?;
                let addr = addr.into();

                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        OUT_OF_GAS_ERR?;
                    }

                    let dest = memory.get_slice(dest_offset, len, &mut gas_left)?;
                    let (offset, offset_overflow) = offset.into_u64_with_overflow();
                    consume_copy_cost(&mut gas_left, len)?;
                    let bytes_written = context.copy_code(&addr, offset as usize, dest);
                    if offset_overflow {
                        zero_slice(dest);
                    } else if (bytes_written as u64) < len {
                        zero_slice(&mut dest[bytes_written..]);
                    }
                }

                code_state.next();
            }
            Opcode::ReturnDataSize => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(
                    last_call_return_data
                        .as_ref()
                        .map(|data| data.len())
                        .unwrap_or_default(),
                )?;
                code_state.next();
            }
            Opcode::ReturnDataCopy => {
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = stack.pop()?;

                let src = last_call_return_data.as_deref().unwrap_or(&[]);
                let (offset, offset_overflow) = offset.into_u64_with_overflow();
                let (len, len_overflow) = len.into_u64_with_overflow();
                let (end, end_overflow) = offset.overflowing_add(len);
                if offset_overflow || len_overflow || end_overflow || end > src.len() as u64 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                    ));
                }

                if len != 0 {
                    let src = get_slice_within_bounds(src, offset.into(), len);
                    let dest = memory.get_slice(dest_offset, len, &mut gas_left)?;
                    copy_slice_padded(src, dest, &mut gas_left)?;
                }

                code_state.next();
            }
            Opcode::ExtCodeHash => {
                if revision < Revision::EVMC_BERLIN {
                    consume_gas::<700>(&mut gas_left)?;
                }
                let [addr] = stack.pop()?;
                let addr = addr.into();
                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                stack.push(context.get_code_hash(&addr))?;
                code_state.next();
            }
            Opcode::BlockHash => {
                consume_gas::<20>(&mut gas_left)?;
                let [block_number] = stack.pop()?;
                let (idx, idx_overflow) = block_number.into_u64_with_overflow();
                if idx_overflow {
                    stack.push(u256::ZERO)?;
                } else {
                    stack.push(context.get_block_hash(idx as i64))?;
                }
                code_state.next();
            }
            Opcode::Coinbase => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().block_coinbase)?;
                code_state.next();
            }
            Opcode::Timestamp => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().block_timestamp as u64)?;
                code_state.next();
            }
            Opcode::Number => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().block_number as u64)?;
                code_state.next();
            }
            Opcode::PrevRandao => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().block_prev_randao)?;
                code_state.next();
            }
            Opcode::GasLimit => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().block_gas_limit as u64)?;
                code_state.next();
            }
            Opcode::ChainId => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().chain_id)?;
                code_state.next();
            }
            Opcode::SelfBalance => {
                check_min_revision(Revision::EVMC_ISTANBUL, revision)?;
                consume_gas::<5>(&mut gas_left)?;
                let addr = message.recipient();
                if u256::from(addr) == u256::ZERO {
                    stack.push(u256::ZERO)?;
                } else {
                    stack.push(context.get_balance(addr))?;
                }
                code_state.next();
            }
            Opcode::BaseFee => {
                check_min_revision(Revision::EVMC_LONDON, revision)?;
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().block_base_fee)?;
                code_state.next();
            }
            Opcode::BlobHash => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<3>(&mut gas_left)?;
                let [idx] = stack.pop()?;
                let (idx, idx_overflow) = idx.into_u64_with_overflow();
                let idx = idx as usize;
                let hashes = context.get_tx_context().blob_hashes;
                if !idx_overflow && idx < hashes.len() {
                    stack.push(hashes[idx])?;
                } else {
                    stack.push(u256::ZERO)?;
                }
                code_state.next();
            }
            Opcode::BlobBaseFee => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<2>(&mut gas_left)?;
                stack.push(context.get_tx_context().blob_base_fee)?;
                code_state.next();
            }
            Opcode::Pop => {
                consume_gas::<2>(&mut gas_left)?;
                let [_] = stack.pop()?;
                code_state.next();
            }
            Opcode::MLoad => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset] = stack.pop()?;

                stack.push(memory.get_word(offset, &mut gas_left)?)?;
                code_state.next();
            }
            Opcode::MStore => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset, value] = stack.pop()?;

                let dest = memory.get_slice(offset, 32, &mut gas_left)?;
                dest.copy_from_slice(value.as_slice());
                code_state.next();
            }
            Opcode::MStore8 => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset, value] = stack.pop()?;

                let dest = memory.get_byte(offset, &mut gas_left)?;
                *dest = value[31];
                code_state.next();
            }
            Opcode::SLoad => {
                if revision < Revision::EVMC_BERLIN {
                    consume_gas::<800>(&mut gas_left)?;
                }
                let [key] = stack.pop()?;
                let key = key.into();
                let addr = message.recipient();
                if revision >= Revision::EVMC_BERLIN {
                    if context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD {
                        consume_gas::<2100>(&mut gas_left)?;
                    } else {
                        consume_gas::<100>(&mut gas_left)?;
                    }
                }
                let value = context.get_storage(addr, &key);
                stack.push(value)?;
                code_state.next();
            }
            Opcode::SStore => {
                check_not_read_only(message, revision)?;
                if revision >= Revision::EVMC_ISTANBUL && gas_left <= 2300 {
                    OUT_OF_GAS_ERR?;
                }
                let [key, value] = stack.pop()?;
                let key = key.into();
                let addr = message.recipient();

                let (dyn_gas_1, dyn_gas_2, dyn_gas_3, refund_1, refund_2, refund_3) =
                    if revision >= Revision::EVMC_LONDON {
                        (100, 2900, 20000, 5000 - 2100 - 100, 4800, 20000 - 100)
                    } else if revision >= Revision::EVMC_BERLIN {
                        (100, 2900, 20000, 5000 - 2100 - 100, 15000, 20000 - 100)
                    } else if revision >= Revision::EVMC_ISTANBUL {
                        (800, 5000, 20000, 4200, 15000, 19200)
                    } else {
                        (5000, 5000, 20000, 0, 0, 0)
                    };

                // dyn gas
                // if Z == Y
                //     dyn_gas_1 =  100                                 800
                // else if Y == X
                //     if X == 0
                //         dyn_gas_3 = 20000
                //     else
                //         dyn_gas_2 = 2900                             5000
                // else
                //     dyn_gas_1 = 100                                  800

                // gas refunds
                //if z != y
                //    if y == x
                //        if x != 0 and z == 0
                //            gas_refunds_2 += 4800                     15000
                //    else
                //        if x != 0
                //            if y == 0
                //                gas_refunds_2 -= 4800                 15000
                //            else if z == 0
                //                gas_refunds_2 += 4800                 15000
                //        if z == x
                //            if x == 0
                //                gas_refunds_3 += 20000 - 100          19200
                //            else
                //                gas_refunds_1 += 5000 - 2100 - 100    4200

                let status = context.set_storage(addr, &key, &value.into());
                let (mut dyn_gas, gas_refund_change) = match status {
                    StorageStatus::EVMC_STORAGE_ASSIGNED => (dyn_gas_1, 0),
                    StorageStatus::EVMC_STORAGE_ADDED => (dyn_gas_3, 0),
                    StorageStatus::EVMC_STORAGE_DELETED => (dyn_gas_2, refund_2),
                    StorageStatus::EVMC_STORAGE_MODIFIED => (dyn_gas_2, 0),
                    StorageStatus::EVMC_STORAGE_DELETED_ADDED => (dyn_gas_1, -refund_2),
                    StorageStatus::EVMC_STORAGE_MODIFIED_DELETED => (dyn_gas_1, refund_2),
                    StorageStatus::EVMC_STORAGE_DELETED_RESTORED => {
                        (dyn_gas_1, -refund_2 + refund_1)
                    }
                    StorageStatus::EVMC_STORAGE_ADDED_DELETED => (dyn_gas_1, refund_3),
                    StorageStatus::EVMC_STORAGE_MODIFIED_RESTORED => (dyn_gas_1, refund_1),
                };
                if revision >= Revision::EVMC_BERLIN
                    && context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD
                {
                    dyn_gas += 2100;
                }
                consume_dyn_gas(&mut gas_left, dyn_gas)?;
                gas_refund += gas_refund_change;
                code_state.next();
            }
            Opcode::Jump => {
                consume_gas::<8>(&mut gas_left)?;
                let [dest] = stack.pop()?;
                code_state.try_jump(dest)?;
            }
            Opcode::JumpI => {
                consume_gas::<10>(&mut gas_left)?;
                let [dest, cond] = stack.pop()?;
                if cond == u256::ZERO {
                    code_state.next();
                } else {
                    code_state.try_jump(dest)?;
                }
            }
            Opcode::Pc => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(code_state.pc())?;
                code_state.next();
            }
            Opcode::MSize => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(memory.len())?;
                code_state.next();
            }
            Opcode::Gas => {
                consume_gas::<2>(&mut gas_left)?;
                stack.push(gas_left)?;
                code_state.next();
            }
            Opcode::JumpDest => {
                consume_gas::<1>(&mut gas_left)?;
                code_state.next();
            }
            Opcode::TLoad => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<100>(&mut gas_left)?;
                let [key] = stack.pop()?;
                let addr = message.recipient();
                let value = context.get_transient_storage(addr, &key.into());
                stack.push(value)?;
                code_state.next();
            }
            Opcode::TStore => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                check_not_read_only(message, revision)?;
                consume_gas::<100>(&mut gas_left)?;
                let [key, value] = stack.pop()?;
                let addr = message.recipient();
                context.set_transient_storage(addr, &key.into(), &value.into());
                code_state.next();
            }
            Opcode::MCopy => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = stack.pop()?;
                if len != u256::ZERO {
                    memory.copy_within(offset, dest_offset, len, &mut gas_left)?;
                }
                code_state.next();
            }
            Opcode::Push1 => push(1, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push2 => push(2, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push3 => push(3, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push4 => push(4, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push5 => push(5, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push6 => push(6, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push7 => push(7, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push8 => push(8, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push9 => push(9, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push10 => push(10, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push11 => push(11, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push12 => push(12, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push13 => push(13, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push14 => push(14, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push15 => push(15, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push16 => push(16, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push17 => push(17, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push18 => push(18, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push19 => push(19, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push20 => push(20, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push21 => push(21, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push22 => push(22, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push23 => push(23, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push24 => push(24, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push25 => push(25, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push26 => push(26, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push27 => push(27, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push28 => push(28, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push29 => push(29, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push30 => push(30, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push31 => push(31, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Push32 => push(32, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup1 => dup(1, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup2 => dup(2, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup3 => dup(3, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup4 => dup(4, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup5 => dup(5, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup6 => dup(6, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup7 => dup(7, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup8 => dup(8, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup9 => dup(9, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup10 => dup(10, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup11 => dup(11, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup12 => dup(12, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup13 => dup(13, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup14 => dup(14, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup15 => dup(15, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Dup16 => dup(16, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap1 => swap(1, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap2 => swap(2, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap3 => swap(3, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap4 => swap(4, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap5 => swap(5, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap6 => swap(6, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap7 => swap(7, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap8 => swap(8, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap9 => swap(9, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap10 => swap(10, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap11 => swap(11, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap12 => swap(12, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap13 => swap(13, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap14 => swap(14, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap15 => swap(15, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Swap16 => swap(16, &mut code_state, &mut stack, &mut gas_left)?,
            Opcode::Log0 => log::<0>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            Opcode::Log1 => log::<1>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            Opcode::Log2 => log::<2>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            Opcode::Log3 => log::<3>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            Opcode::Log4 => log::<4>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            Opcode::Create => create::<false>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                &mut last_call_return_data,
                revision,
                &mut gas_left,
                &mut gas_refund,
            )?,
            Opcode::Call => call::<false>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
                &mut gas_refund,
                &mut last_call_return_data,
            )?,
            Opcode::CallCode => call::<true>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
                &mut gas_refund,
                &mut last_call_return_data,
            )?,
            Opcode::Return => {
                let [offset, len] = stack.pop()?;
                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    OUT_OF_GAS_ERR?;
                }
                let data = memory.get_slice(offset, len, &mut gas_left)?;
                output = Some(data.to_owned());
                step_status_code = StepStatusCode::EVMC_STEP_RETURNED;
                code_state.next();
                break;
            }
            Opcode::DelegateCall => static_delegate_call::<true>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
                &mut gas_refund,
                &mut last_call_return_data,
            )?,
            Opcode::Create2 => create::<true>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                &mut last_call_return_data,
                revision,
                &mut gas_left,
                &mut gas_refund,
            )?,
            Opcode::StaticCall => static_delegate_call::<false>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
                &mut gas_refund,
                &mut last_call_return_data,
            )?,
            Opcode::Revert => {
                let [offset, len] = stack.pop()?;
                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    OUT_OF_GAS_ERR?;
                }
                let data = memory.get_slice(offset, len, &mut gas_left)?;
                // TODO revert state changes
                // gas_refund = original_gas_refund;
                output = Some(data.to_owned());
                step_status_code = StepStatusCode::EVMC_STEP_REVERTED;
                status_code = StatusCode::EVMC_REVERT;
                code_state.next();
                break;
            }
            Opcode::Invalid => {
                check_min_revision(Revision::EVMC_HOMESTEAD, revision)?;
                return Err((
                    StepStatusCode::EVMC_STEP_FAILED,
                    StatusCode::EVMC_INVALID_INSTRUCTION,
                ));
            }
            Opcode::SelfDestruct => {
                check_not_read_only(message, revision)?;
                consume_gas::<5000>(&mut gas_left)?;
                let [addr] = stack.pop()?;
                let addr = addr.into();

                let tx_context = context.get_tx_context();
                if revision >= Revision::EVMC_BERLIN && addr != tx_context.tx_origin
                        //&& addr != tx_context.tx_to // TODO
                        && !(revision >= Revision::EVMC_SHANGHAI && addr == tx_context.block_coinbase) && context.access_account(&addr) == AccessStatus::EVMC_ACCESS_COLD
                {
                    consume_gas::<2600>(&mut gas_left)?;
                }

                if u256::from(context.get_balance(message.recipient())) > u256::ZERO
                    && !context.account_exists(&addr)
                {
                    consume_gas::<25000>(&mut gas_left)?;
                }

                let destructed = context.selfdestruct(message.recipient(), &addr);
                if revision <= Revision::EVMC_BERLIN && destructed {
                    gas_refund += 24000;
                }

                step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                code_state.next();
                break;
            }
        }
    }

    Ok(RunResult::new(
        step_status_code,
        status_code,
        revision,
        code_state,
        gas_left,
        gas_refund,
        output,
        stack,
        memory,
        last_call_return_data,
    ))
}

fn push(
    len: usize,
    code_state: &mut CodeState,
    stack: &mut Stack,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_gas::<3>(gas_left)?;
    code_state.next();
    stack.push(code_state.get_push_data(len)?)?;
    Ok(())
}

fn dup(
    nth: usize,
    code_state: &mut CodeState,
    stack: &mut Stack,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_gas::<3>(gas_left)?;
    stack.push(stack.nth(nth)?)?;
    code_state.next();
    Ok(())
}

fn swap(
    nth: usize,
    code_state: &mut CodeState,
    stack: &mut Stack,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_gas::<3>(gas_left)?;
    stack.swap_with_top(nth)?;
    code_state.next();
    Ok(())
}

fn log<const N: usize>(
    code_state: &mut CodeState,
    stack: &mut Stack,
    memory: &mut Memory,
    context: &mut ExecutionContext,
    message: &ExecutionMessage,
    revision: Revision,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    check_not_read_only(message, revision)?;
    consume_gas::<375>(gas_left)?;
    let [offset, len] = stack.pop()?;
    let topics: [u256; N] = stack.pop()?;
    let (len, len_overflow) = len.into_u64_with_overflow();
    let (len8, len8_overflow) = len.overflowing_mul(8);
    let (cost, cost_overflow) = (375 * N as u64).overflowing_add(len8);
    if len_overflow || len8_overflow || cost_overflow {
        return OUT_OF_GAS_ERR;
    }
    consume_dyn_gas(gas_left, cost)?;

    let data = memory.get_slice(offset, len, gas_left)?;
    let topics: &[_; N] = unsafe { mem::transmute(&topics) };
    context.emit_log(message.recipient(), data, topics.as_slice());
    code_state.next();
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn create<const CREATE2: bool>(
    code_state: &mut CodeState,
    stack: &mut Stack,
    memory: &mut Memory,
    context: &mut ExecutionContext,
    message: &ExecutionMessage,
    last_call_return_data: &mut Option<Vec<u8>>,
    revision: Revision,
    gas_left: &mut u64,
    gas_refund: &mut i64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_gas::<32000>(gas_left)?;
    check_not_read_only(message, revision)?;
    let [value, offset, len] = stack.pop()?;
    let salt = if CREATE2 {
        stack.pop::<1>()?[0]
    } else {
        u256::ZERO // ignored
    };
    let (len, len_overflow) = len.into_u64_with_overflow();
    if len_overflow {
        return OUT_OF_GAS_ERR;
    }

    let init_code_word_size = word_size(len)?;
    if revision >= Revision::EVMC_SHANGHAI {
        const MAX_INIT_CODE_LEN: u64 = 2 * 24576;
        if len > MAX_INIT_CODE_LEN {
            return OUT_OF_GAS_ERR;
        }
        let init_code_cost = 2 * init_code_word_size; // does not overflow
        consume_dyn_gas(gas_left, init_code_cost)?;
    }
    if CREATE2 {
        let hash_cost = 6 * init_code_word_size; // does not overflow
        consume_dyn_gas(gas_left, hash_cost)?;
    }

    let init_code = memory.get_slice(offset, len, gas_left)?;

    if value > context.get_balance(message.recipient()).into() {
        *last_call_return_data = None;
        stack.push(u256::ZERO)?;
        code_state.next();
        return Ok(());
    }

    let gas_limit = *gas_left - *gas_left / 64;
    consume_dyn_gas(gas_left, gas_limit)?;

    let message = ExecutionMessage::new(
        if CREATE2 {
            MessageKind::EVMC_CREATE2
        } else {
            MessageKind::EVMC_CREATE
        },
        message.flags(),
        message.depth() + 1,
        gas_limit as i64,
        u256::ZERO.into(), // ignored
        *message.recipient(),
        Some(init_code),
        value.into(),
        salt.into(),
        u256::ZERO.into(), // ignored
        None,
    );
    let result = context.call(&message);

    *gas_left += result.gas_left() as u64;
    *gas_refund += result.gas_refund();

    if result.status_code() == StatusCode::EVMC_SUCCESS {
        let Some(addr) = result.create_address() else {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_INTERNAL_ERROR,
            ));
        };

        *last_call_return_data = None;
        stack.push(addr)?;
    } else {
        *last_call_return_data = result.output().map(ToOwned::to_owned);
        stack.push(u256::ZERO)?;
    }
    code_state.next();
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn call<const CODE: bool>(
    code_state: &mut CodeState,
    stack: &mut Stack,
    memory: &mut Memory,
    context: &mut ExecutionContext,
    message: &ExecutionMessage,
    revision: Revision,
    gas_left: &mut u64,
    gas_refund: &mut i64,
    last_call_return_data: &mut Option<Vec<u8>>,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if revision < Revision::EVMC_BERLIN {
        consume_gas::<700>(gas_left)?;
    }
    let [gas, addr, value, args_offset, args_len, ret_offset, ret_len] = stack.pop()?;

    if !CODE && value != u256::ZERO {
        check_not_read_only(message, revision)?;
    }

    let addr = addr.into();
    let (args_len, args_len_overflow) = args_len.into_u64_with_overflow();
    let (ret_len, ret_len_overflow) = ret_len.into_u64_with_overflow();
    if args_len_overflow || ret_len_overflow {
        return OUT_OF_GAS_ERR;
    }

    consume_address_access_cost(gas_left, &addr, context, revision)?;
    // access slice to consume potential memory expansion cost but drop it so that we can get
    // another mutable reference into memory for input
    let _dest = memory.get_slice(ret_offset, ret_len, gas_left)?;
    let input = memory.get_slice(args_offset, args_len, gas_left)?;
    consume_positive_value_cost(&value, gas_left)?;
    if !CODE {
        consume_value_to_empty_account_cost(&value, &addr, context, gas_left)?;
    }

    let limit = *gas_left - *gas_left / 64;
    let mut endowment = gas.into_u64_saturating();
    if revision >= Revision::EVMC_TANGERINE_WHISTLE {
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
    }
    let stipend = if value == u256::ZERO { 0 } else { 2300 };
    *gas_left += stipend;

    if value > u256::from(context.get_balance(message.recipient())) {
        *last_call_return_data = None;
        stack.push(u256::ZERO)?;
        code_state.next();
        return Ok(());
    }

    let call_message = if CODE {
        ExecutionMessage::new(
            MessageKind::EVMC_CALLCODE,
            message.flags(),
            message.depth() + 1,
            (endowment + stipend) as i64,
            *message.recipient(),
            *message.recipient(),
            Some(input),
            value.into(),
            u256::ZERO.into(), // ignored
            addr,
            None,
        )
    } else {
        ExecutionMessage::new(
            MessageKind::EVMC_CALL,
            message.flags(),
            message.depth() + 1,
            (endowment + stipend) as i64,
            addr,
            *message.recipient(),
            Some(input),
            value.into(),
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            None,
        )
    };

    let result = context.call(&call_message);
    *last_call_return_data = result.output().map(ToOwned::to_owned);
    let dest = memory.get_slice(ret_offset, ret_len, gas_left)?;
    if let Some(output) = last_call_return_data {
        let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
        dest[..min_len].copy_from_slice(&output[..min_len]);
    }

    *gas_left += result.gas_left() as u64;
    consume_dyn_gas(gas_left, endowment)?;
    consume_dyn_gas(gas_left, stipend)?;
    *gas_refund += result.gas_refund();

    stack.push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
    code_state.next();
    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn static_delegate_call<const DELEGATE: bool>(
    code_state: &mut CodeState,
    stack: &mut Stack,
    memory: &mut Memory,
    context: &mut ExecutionContext,
    message: &ExecutionMessage,
    revision: Revision,
    gas_left: &mut u64,
    gas_refund: &mut i64,
    last_call_return_data: &mut Option<Vec<u8>>,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if revision < Revision::EVMC_BERLIN {
        consume_gas::<700>(gas_left)?;
    }
    let [gas, addr, args_offset, args_len, ret_offset, ret_len] = stack.pop()?;

    let addr = addr.into();
    let (args_len, args_len_overflow) = args_len.into_u64_with_overflow();
    let (ret_len, ret_len_overflow) = ret_len.into_u64_with_overflow();
    if args_len_overflow || ret_len_overflow {
        return OUT_OF_GAS_ERR;
    }

    consume_address_access_cost(gas_left, &addr, context, revision)?;
    // access slice to consume potential memory expansion cost but drop it so that we can get
    // another mutable reference into memory for input
    let _dest = memory.get_slice(ret_offset, ret_len, gas_left)?;
    let input = memory.get_slice(args_offset, args_len, gas_left)?;

    let limit = *gas_left - *gas_left / 64;
    let mut endowment = gas.into_u64_saturating();
    if revision >= Revision::EVMC_TANGERINE_WHISTLE {
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
    }

    let call_message = if DELEGATE {
        ExecutionMessage::new(
            MessageKind::EVMC_DELEGATECALL,
            message.flags(),
            message.depth() + 1,
            endowment as i64,
            *message.recipient(),
            *message.sender(),
            Some(input),
            *message.value(),
            u256::ZERO.into(), // ignored
            addr,
            None,
        )
    } else {
        ExecutionMessage::new(
            MessageKind::EVMC_CALL,
            MessageFlags::EVMC_STATIC as u32,
            message.depth() + 1,
            (endowment) as i64,
            addr,
            *message.recipient(),
            Some(input),
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            None,
        )
    };

    let result = context.call(&call_message);
    *last_call_return_data = result.output().map(ToOwned::to_owned);
    let dest = memory.get_slice(ret_offset, ret_len, gas_left)?;
    if let Some(output) = last_call_return_data {
        let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
        dest[..min_len].copy_from_slice(&output[..min_len]);
    }

    *gas_left += result.gas_left() as u64;
    consume_dyn_gas(gas_left, endowment)?;
    *gas_refund += result.gas_refund();

    stack.push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
    code_state.next();
    Ok(())
}

#[inline(always)]
fn get_slice_within_bounds<T>(data: &[T], offset: u256, len: u64) -> &[T] {
    if len == 0 {
        return &[];
    }
    let (offset, offset_overflow) = offset.into_u64_with_overflow();
    if offset_overflow {
        return &[];
    }
    let offset = offset as usize;
    let len = len as usize;
    let (end, end_overflow) = offset.overflowing_add(len);
    if end_overflow || offset >= data.len() {
        &[]
    } else {
        &data[offset..min(end, data.len())]
    }
}

#[inline(always)]
fn zero_slice(data: &mut [u8]) {
    for byte in data {
        *byte = 0;
    }
}

#[inline(always)]
fn copy_slice_padded(
    src: &[u8],
    dest: &mut [u8],
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_copy_cost(gas_left, dest.len() as u64)?;
    dest[..src.len()].copy_from_slice(src);
    zero_slice(&mut dest[src.len()..]);
    Ok(())
}

#[inline(always)]
fn word_size(bytes: u64) -> Result<u64, (StepStatusCode, StatusCode)> {
    let (end, overflow) = bytes.overflowing_add(31);
    if overflow {
        OUT_OF_GAS_ERR?;
    }
    Ok(end / 32)
}

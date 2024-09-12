mod code_state;
mod gas;
mod memory;
mod run_result;
mod run_state;
mod stack;
mod tx_context;
mod utils;

use std::{cmp::min, mem};

use evmc_vm::{
    AccessStatus, ExecutionMessage, MessageFlags, MessageKind, Revision, StatusCode,
    StepStatusCode, StorageStatus, Uint256,
};
use sha3::{Digest, Keccak256};

pub use crate::interpreter::{code_state::CodeState, memory::Memory, stack::Stack, utils::*};
use crate::{
    interpreter::{
        code_state::GetOpcodeError,
        gas::{
            consume_address_access_cost, consume_copy_cost, consume_gas,
            consume_positive_value_cost, consume_value_to_empty_account_cost,
        },
        run_result::RunResult,
        run_state::RunState,
        tx_context::ExecutionTxContext,
    },
    types::{u256, Opcode},
};

#[allow(dead_code)] // TODO remove once all parts are merged
pub fn run(mut state: RunState) -> Result<RunResult, StatusCode> {
    loop {
        match &mut state.steps {
            None => (),
            Some(0) => break,
            Some(steps) => *steps -= 1,
        }
        let op = match state.code_state.get() {
            Ok(op) => op,
            Err(GetOpcodeError::OutOfRange) => {
                state.step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                break;
            }
            Err(GetOpcodeError::Invalid) => {
                return Err(StatusCode::EVMC_INVALID_INSTRUCTION);
            }
        };
        match op {
            Opcode::Stop => {
                state.step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                state.status_code = StatusCode::EVMC_SUCCESS;
                break;
            }
            Opcode::Add => {
                consume_gas(&mut state.gas_left, 3)?;
                let [value1, value2] = state.stack.pop()?;
                state.stack.push(value1 + value2)?;
                state.code_state.next();
            }
            Opcode::Mul => {
                consume_gas(&mut state.gas_left, 5)?;
                let [fac1, fac2] = state.stack.pop()?;
                state.stack.push(fac1 * fac2)?;
                state.code_state.next();
            }
            Opcode::Sub => {
                consume_gas(&mut state.gas_left, 3)?;
                let [value1, value2] = state.stack.pop()?;
                state.stack.push(value1 - value2)?;
                state.code_state.next();
            }
            Opcode::Div => {
                consume_gas(&mut state.gas_left, 5)?;
                let [value, denominator] = state.stack.pop()?;
                state.stack.push(value / denominator)?;
                state.code_state.next();
            }
            Opcode::SDiv => {
                consume_gas(&mut state.gas_left, 5)?;
                let [value, denominator] = state.stack.pop()?;
                state.stack.push(value.sdiv(denominator))?;
                state.code_state.next();
            }
            Opcode::Mod => {
                consume_gas(&mut state.gas_left, 5)?;
                let [value, denominator] = state.stack.pop()?;
                state.stack.push(value % denominator)?;
                state.code_state.next();
            }
            Opcode::SMod => {
                consume_gas(&mut state.gas_left, 5)?;
                let [value, denominator] = state.stack.pop()?;
                state.stack.push(value.srem(denominator))?;
                state.code_state.next();
            }
            Opcode::AddMod => {
                consume_gas(&mut state.gas_left, 8)?;
                let [value1, value2, denominator] = state.stack.pop()?;
                state
                    .stack
                    .push(u256::addmod(value1, value2, denominator))?;
                state.code_state.next();
            }
            Opcode::MulMod => {
                consume_gas(&mut state.gas_left, 8)?;
                let [fac1, fac2, denominator] = state.stack.pop()?;
                state.stack.push(u256::mulmod(fac1, fac2, denominator))?;
                state.code_state.next();
            }
            Opcode::Exp => {
                consume_gas(&mut state.gas_left, 10)?;
                let [value, exp] = state.stack.pop()?;
                let byte_size = 32 - exp.into_iter().take_while(|byte| *byte == 0).count() as u64;
                consume_gas(&mut state.gas_left, byte_size * 50)?; // * does not overflow
                state.stack.push(value.pow(exp))?;
                state.code_state.next();
            }
            Opcode::SignExtend => {
                consume_gas(&mut state.gas_left, 5)?;
                let [size, value] = state.stack.pop()?;
                state.stack.push(u256::signextend(size, value))?;
                state.code_state.next();
            }
            Opcode::Lt => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs < rhs)?;
                state.code_state.next();
            }
            Opcode::Gt => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs > rhs)?;
                state.code_state.next();
            }
            Opcode::SLt => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs.slt(&rhs))?;
                state.code_state.next();
            }
            Opcode::SGt => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs.sgt(&rhs))?;
                state.code_state.next();
            }
            Opcode::Eq => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs == rhs)?;
                state.code_state.next();
            }
            Opcode::IsZero => {
                consume_gas(&mut state.gas_left, 3)?;
                let [value] = state.stack.pop()?;
                state.stack.push(value == u256::ZERO)?;
                state.code_state.next();
            }
            Opcode::And => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs & rhs)?;
                state.code_state.next();
            }
            Opcode::Or => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs | rhs)?;
                state.code_state.next();
            }
            Opcode::Xor => {
                consume_gas(&mut state.gas_left, 3)?;
                let [lhs, rhs] = state.stack.pop()?;
                state.stack.push(lhs ^ rhs)?;
                state.code_state.next();
            }
            Opcode::Not => {
                consume_gas(&mut state.gas_left, 3)?;
                let [value] = state.stack.pop()?;
                state.stack.push(!value)?;
                state.code_state.next();
            }
            Opcode::Byte => {
                consume_gas(&mut state.gas_left, 3)?;
                let [offset, value] = state.stack.pop()?;
                state.stack.push(value.byte(offset))?;
                state.code_state.next();
            }
            Opcode::Shl => {
                consume_gas(&mut state.gas_left, 3)?;
                let [shift, value] = state.stack.pop()?;
                state.stack.push(value << shift)?;
                state.code_state.next();
            }
            Opcode::Shr => {
                consume_gas(&mut state.gas_left, 3)?;
                let [shift, value] = state.stack.pop()?;
                state.stack.push(value >> shift)?;
                state.code_state.next();
            }
            Opcode::Sar => {
                consume_gas(&mut state.gas_left, 3)?;
                let [shift, value] = state.stack.pop()?;
                state.stack.push(value.sar(shift))?;
                state.code_state.next();
            }
            Opcode::Sha3 => {
                consume_gas(&mut state.gas_left, 30)?;
                let [offset, len] = state.stack.pop()?;

                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    return Err(StatusCode::EVMC_OUT_OF_GAS);
                }
                consume_gas(&mut state.gas_left, 6 * word_size(len)?)?; // * does not overflow

                let data = state
                    .memory
                    .get_mut_slice(offset, len, &mut state.gas_left)?;
                let mut hasher = Keccak256::new();
                hasher.update(data);
                let mut bytes = [0; 32];
                hasher.finalize_into((&mut bytes).into());
                state.stack.push(bytes)?;
                state.code_state.next();
            }
            Opcode::Address => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.message.recipient())?;
                state.code_state.next();
            }
            Opcode::Balance => {
                if state.revision < Revision::EVMC_BERLIN {
                    consume_gas(&mut state.gas_left, 700)?;
                }
                let [addr] = state.stack.pop()?;
                let addr = addr.into();
                consume_address_access_cost(&addr, &mut state)?;
                state.stack.push(state.context.get_balance(&addr))?;
                state.code_state.next();
            }
            Opcode::Origin => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.context.get_tx_context().tx_origin)?;
                state.code_state.next();
            }
            Opcode::Caller => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.message.sender())?;
                state.code_state.next();
            }
            Opcode::CallValue => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(*state.message.value())?;
                state.code_state.next();
            }
            Opcode::CallDataLoad => {
                consume_gas(&mut state.gas_left, 3)?;
                let [offset] = state.stack.pop()?;
                let (offset, overflow) = offset.into_u64_with_overflow();
                let offset = offset as usize;
                let call_data = state.message.input().map(Vec::as_slice).unwrap_or_default();
                if overflow || offset >= call_data.len() {
                    state.stack.push(u256::ZERO)?;
                } else {
                    let end = min(call_data.len(), offset + 32);
                    let mut bytes = [0; 32];
                    bytes[..end - offset].copy_from_slice(&call_data[offset..end]);
                    state.stack.push(bytes)?;
                }
                state.code_state.next();
            }
            Opcode::CallDataSize => {
                consume_gas(&mut state.gas_left, 2)?;
                let call_data_len = state.message.input().map(Vec::len).unwrap_or_default();
                state.stack.push(call_data_len)?;
                state.code_state.next();
            }
            Opcode::Push0 => {
                check_min_revision(Revision::EVMC_SHANGHAI, state.revision)?;
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(u256::ZERO)?;
                state.code_state.next();
            }
            Opcode::CallDataCopy => {
                consume_gas(&mut state.gas_left, 3)?;
                let [dest_offset, offset, len] = state.stack.pop()?;

                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        return Err(StatusCode::EVMC_INVALID_MEMORY_ACCESS);
                    }

                    let src = state.message.input().map(Vec::as_slice).unwrap_or_default();
                    let src = src.get_within_bounds(offset, len);
                    let dest = state
                        .memory
                        .get_mut_slice(dest_offset, len, &mut state.gas_left)?;
                    dest.copy_padded(src, &mut state.gas_left)?;
                }

                state.code_state.next();
            }
            Opcode::CodeSize => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.code_state.len())?;
                state.code_state.next();
            }
            Opcode::CodeCopy => {
                consume_gas(&mut state.gas_left, 3)?;
                let [dest_offset, offset, len] = state.stack.pop()?;

                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        return Err(StatusCode::EVMC_OUT_OF_GAS);
                    }

                    let src = state.code_state.get_within_bounds(offset, len);
                    let dest = state
                        .memory
                        .get_mut_slice(dest_offset, len, &mut state.gas_left)?;
                    dest.copy_padded(src, &mut state.gas_left)?;
                }

                state.code_state.next();
            }
            Opcode::GasPrice => {
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().tx_gas_price)?;
                state.code_state.next();
            }
            Opcode::ExtCodeSize => {
                if state.revision < Revision::EVMC_BERLIN {
                    consume_gas(&mut state.gas_left, 700)?;
                }
                let [addr] = state.stack.pop()?;
                let addr = addr.into();
                consume_address_access_cost(&addr, &mut state)?;
                state.stack.push(state.context.get_code_size(&addr))?;
                state.code_state.next();
            }
            Opcode::ExtCodeCopy => {
                if state.revision < Revision::EVMC_BERLIN {
                    consume_gas(&mut state.gas_left, 700)?;
                }
                let [addr, dest_offset, offset, len] = state.stack.pop()?;
                let addr = addr.into();

                consume_address_access_cost(&addr, &mut state)?;
                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        return Err(StatusCode::EVMC_OUT_OF_GAS);
                    }

                    let dest = state
                        .memory
                        .get_mut_slice(dest_offset, len, &mut state.gas_left)?;
                    let (offset, offset_overflow) = offset.into_u64_with_overflow();
                    consume_copy_cost(&mut state.gas_left, len)?;
                    let bytes_written = state.context.copy_code(&addr, offset as usize, dest);
                    if offset_overflow {
                        dest.set_to_zero();
                    } else if (bytes_written as u64) < len {
                        dest[bytes_written..].set_to_zero();
                    }
                }

                state.code_state.next();
            }
            Opcode::ReturnDataSize => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(
                    state
                        .last_call_return_data
                        .as_ref()
                        .map(Vec::len)
                        .unwrap_or_default(),
                )?;
                state.code_state.next();
            }
            Opcode::ReturnDataCopy => {
                consume_gas(&mut state.gas_left, 3)?;
                let [dest_offset, offset, len] = state.stack.pop()?;

                let src = state.last_call_return_data.as_deref().unwrap_or_default();
                let (offset, offset_overflow) = offset.into_u64_with_overflow();
                let (len, len_overflow) = len.into_u64_with_overflow();
                let (end, end_overflow) = offset.overflowing_add(len);
                if offset_overflow || len_overflow || end_overflow || end > src.len() as u64 {
                    return Err(StatusCode::EVMC_INVALID_MEMORY_ACCESS);
                }

                if len != 0 {
                    let src = src.get_within_bounds(offset.into(), len);
                    let dest = state
                        .memory
                        .get_mut_slice(dest_offset, len, &mut state.gas_left)?;
                    dest.copy_padded(src, &mut state.gas_left)?;
                }

                state.code_state.next();
            }
            Opcode::ExtCodeHash => {
                if state.revision < Revision::EVMC_BERLIN {
                    consume_gas(&mut state.gas_left, 700)?;
                }
                let [addr] = state.stack.pop()?;
                let addr = addr.into();
                consume_address_access_cost(&addr, &mut state)?;
                state.stack.push(state.context.get_code_hash(&addr))?;
                state.code_state.next();
            }
            Opcode::BlockHash => {
                consume_gas(&mut state.gas_left, 20)?;
                let [block_number] = state.stack.pop()?;
                let (idx, idx_overflow) = block_number.into_u64_with_overflow();
                if idx_overflow {
                    state.stack.push(u256::ZERO)?;
                } else {
                    state.stack.push(state.context.get_block_hash(idx as i64))?;
                }
                state.code_state.next();
            }
            Opcode::Coinbase => {
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().block_coinbase)?;
                state.code_state.next();
            }
            Opcode::Timestamp => {
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().block_timestamp as u64)?;
                state.code_state.next();
            }
            Opcode::Number => {
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().block_number as u64)?;
                state.code_state.next();
            }
            Opcode::PrevRandao => {
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().block_prev_randao)?;
                state.code_state.next();
            }
            Opcode::GasLimit => {
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().block_gas_limit as u64)?;
                state.code_state.next();
            }
            Opcode::ChainId => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.context.get_tx_context().chain_id)?;
                state.code_state.next();
            }
            Opcode::SelfBalance => {
                check_min_revision(Revision::EVMC_ISTANBUL, state.revision)?;
                consume_gas(&mut state.gas_left, 5)?;
                let addr = state.message.recipient();
                if u256::from(addr) == u256::ZERO {
                    state.stack.push(u256::ZERO)?;
                } else {
                    state.stack.push(state.context.get_balance(addr))?;
                }
                state.code_state.next();
            }
            Opcode::BaseFee => {
                check_min_revision(Revision::EVMC_LONDON, state.revision)?;
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().block_base_fee)?;
                state.code_state.next();
            }
            Opcode::BlobHash => {
                check_min_revision(Revision::EVMC_CANCUN, state.revision)?;
                consume_gas(&mut state.gas_left, 3)?;
                let [idx] = state.stack.pop()?;
                let (idx, idx_overflow) = idx.into_u64_with_overflow();
                let idx = idx as usize;
                let hashes = ExecutionTxContext::from(state.context.get_tx_context()).blob_hashes;
                if !idx_overflow && idx < hashes.len() {
                    state.stack.push(hashes[idx])?;
                } else {
                    state.stack.push(u256::ZERO)?;
                }
                state.code_state.next();
            }
            Opcode::BlobBaseFee => {
                check_min_revision(Revision::EVMC_CANCUN, state.revision)?;
                consume_gas(&mut state.gas_left, 2)?;
                state
                    .stack
                    .push(state.context.get_tx_context().blob_base_fee)?;
                state.code_state.next();
            }
            Opcode::Pop => {
                consume_gas(&mut state.gas_left, 2)?;
                let [_] = state.stack.pop()?;
                state.code_state.next();
            }
            Opcode::MLoad => {
                consume_gas(&mut state.gas_left, 3)?;
                let [offset] = state.stack.pop()?;

                state
                    .stack
                    .push(state.memory.get_word(offset, &mut state.gas_left)?)?;
                state.code_state.next();
            }
            Opcode::MStore => {
                consume_gas(&mut state.gas_left, 3)?;
                let [offset, value] = state.stack.pop()?;

                let dest = state
                    .memory
                    .get_mut_slice(offset, 32, &mut state.gas_left)?;
                dest.copy_from_slice(value.as_slice());
                state.code_state.next();
            }
            Opcode::MStore8 => {
                consume_gas(&mut state.gas_left, 3)?;
                let [offset, value] = state.stack.pop()?;

                let dest = state.memory.get_mut_byte(offset, &mut state.gas_left)?;
                *dest = value[31];
                state.code_state.next();
            }
            Opcode::SLoad => {
                if state.revision < Revision::EVMC_BERLIN {
                    consume_gas(&mut state.gas_left, 800)?;
                }
                let [key] = state.stack.pop()?;
                let key = key.into();
                let addr = state.message.recipient();
                if state.revision >= Revision::EVMC_BERLIN {
                    if state.context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD {
                        consume_gas(&mut state.gas_left, 2100)?;
                    } else {
                        consume_gas(&mut state.gas_left, 100)?;
                    }
                }
                let value = state.context.get_storage(addr, &key);
                state.stack.push(value)?;
                state.code_state.next();
            }
            Opcode::SStore => sstore(&mut state)?,
            Opcode::Jump => {
                consume_gas(&mut state.gas_left, 8)?;
                let [dest] = state.stack.pop()?;
                state.code_state.try_jump(dest)?;
            }
            Opcode::JumpI => {
                consume_gas(&mut state.gas_left, 10)?;
                let [dest, cond] = state.stack.pop()?;
                if cond == u256::ZERO {
                    state.code_state.next();
                } else {
                    state.code_state.try_jump(dest)?;
                }
            }
            Opcode::Pc => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.code_state.pc())?;
                state.code_state.next();
            }
            Opcode::MSize => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.memory.len())?;
                state.code_state.next();
            }
            Opcode::Gas => {
                consume_gas(&mut state.gas_left, 2)?;
                state.stack.push(state.gas_left)?;
                state.code_state.next();
            }
            Opcode::JumpDest => {
                consume_gas(&mut state.gas_left, 1)?;
                state.code_state.next();
            }
            Opcode::TLoad => {
                check_min_revision(Revision::EVMC_CANCUN, state.revision)?;
                consume_gas(&mut state.gas_left, 100)?;
                let [key] = state.stack.pop()?;
                let addr = state.message.recipient();
                let value = state.context.get_transient_storage(addr, &key.into());
                state.stack.push(value)?;
                state.code_state.next();
            }
            Opcode::TStore => {
                check_min_revision(Revision::EVMC_CANCUN, state.revision)?;
                check_not_read_only(&state)?;
                consume_gas(&mut state.gas_left, 100)?;
                let [key, value] = state.stack.pop()?;
                let addr = state.message.recipient();
                state
                    .context
                    .set_transient_storage(addr, &key.into(), &value.into());
                state.code_state.next();
            }
            Opcode::MCopy => {
                check_min_revision(Revision::EVMC_CANCUN, state.revision)?;
                consume_gas(&mut state.gas_left, 3)?;
                let [dest_offset, offset, len] = state.stack.pop()?;
                if len != u256::ZERO {
                    state
                        .memory
                        .copy_within(offset, dest_offset, len, &mut state.gas_left)?;
                }
                state.code_state.next();
            }
            Opcode::Push1 => push(1, &mut state)?,
            Opcode::Push2 => push(2, &mut state)?,
            Opcode::Push3 => push(3, &mut state)?,
            Opcode::Push4 => push(4, &mut state)?,
            Opcode::Push5 => push(5, &mut state)?,
            Opcode::Push6 => push(6, &mut state)?,
            Opcode::Push7 => push(7, &mut state)?,
            Opcode::Push8 => push(8, &mut state)?,
            Opcode::Push9 => push(9, &mut state)?,
            Opcode::Push10 => push(10, &mut state)?,
            Opcode::Push11 => push(11, &mut state)?,
            Opcode::Push12 => push(12, &mut state)?,
            Opcode::Push13 => push(13, &mut state)?,
            Opcode::Push14 => push(14, &mut state)?,
            Opcode::Push15 => push(15, &mut state)?,
            Opcode::Push16 => push(16, &mut state)?,
            Opcode::Push17 => push(17, &mut state)?,
            Opcode::Push18 => push(18, &mut state)?,
            Opcode::Push19 => push(19, &mut state)?,
            Opcode::Push20 => push(20, &mut state)?,
            Opcode::Push21 => push(21, &mut state)?,
            Opcode::Push22 => push(22, &mut state)?,
            Opcode::Push23 => push(23, &mut state)?,
            Opcode::Push24 => push(24, &mut state)?,
            Opcode::Push25 => push(25, &mut state)?,
            Opcode::Push26 => push(26, &mut state)?,
            Opcode::Push27 => push(27, &mut state)?,
            Opcode::Push28 => push(28, &mut state)?,
            Opcode::Push29 => push(29, &mut state)?,
            Opcode::Push30 => push(30, &mut state)?,
            Opcode::Push31 => push(31, &mut state)?,
            Opcode::Push32 => push(32, &mut state)?,
            Opcode::Dup1 => dup(1, &mut state)?,
            Opcode::Dup2 => dup(2, &mut state)?,
            Opcode::Dup3 => dup(3, &mut state)?,
            Opcode::Dup4 => dup(4, &mut state)?,
            Opcode::Dup5 => dup(5, &mut state)?,
            Opcode::Dup6 => dup(6, &mut state)?,
            Opcode::Dup7 => dup(7, &mut state)?,
            Opcode::Dup8 => dup(8, &mut state)?,
            Opcode::Dup9 => dup(9, &mut state)?,
            Opcode::Dup10 => dup(10, &mut state)?,
            Opcode::Dup11 => dup(11, &mut state)?,
            Opcode::Dup12 => dup(12, &mut state)?,
            Opcode::Dup13 => dup(13, &mut state)?,
            Opcode::Dup14 => dup(14, &mut state)?,
            Opcode::Dup15 => dup(15, &mut state)?,
            Opcode::Dup16 => dup(16, &mut state)?,
            Opcode::Swap1 => swap(1, &mut state)?,
            Opcode::Swap2 => swap(2, &mut state)?,
            Opcode::Swap3 => swap(3, &mut state)?,
            Opcode::Swap4 => swap(4, &mut state)?,
            Opcode::Swap5 => swap(5, &mut state)?,
            Opcode::Swap6 => swap(6, &mut state)?,
            Opcode::Swap7 => swap(7, &mut state)?,
            Opcode::Swap8 => swap(8, &mut state)?,
            Opcode::Swap9 => swap(9, &mut state)?,
            Opcode::Swap10 => swap(10, &mut state)?,
            Opcode::Swap11 => swap(11, &mut state)?,
            Opcode::Swap12 => swap(12, &mut state)?,
            Opcode::Swap13 => swap(13, &mut state)?,
            Opcode::Swap14 => swap(14, &mut state)?,
            Opcode::Swap15 => swap(15, &mut state)?,
            Opcode::Swap16 => swap(16, &mut state)?,
            Opcode::Log0 => log::<0>(&mut state)?,
            Opcode::Log1 => log::<1>(&mut state)?,
            Opcode::Log2 => log::<2>(&mut state)?,
            Opcode::Log3 => log::<3>(&mut state)?,
            Opcode::Log4 => log::<4>(&mut state)?,
            Opcode::Create => create(&mut state)?,
            Opcode::Call => call(&mut state)?,
            Opcode::CallCode => call_code(&mut state)?,
            Opcode::Return => {
                let [offset, len] = state.stack.pop()?;
                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    return Err(StatusCode::EVMC_OUT_OF_GAS);
                }
                let data = state
                    .memory
                    .get_mut_slice(offset, len, &mut state.gas_left)?;
                state.output = Some(data.to_owned());
                state.step_status_code = StepStatusCode::EVMC_STEP_RETURNED;
                state.code_state.next();
                break;
            }
            Opcode::DelegateCall => delegate_call(&mut state)?,
            Opcode::Create2 => create2(&mut state)?,
            Opcode::StaticCall => static_call(&mut state)?,
            Opcode::Revert => {
                let [offset, len] = state.stack.pop()?;
                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    return Err(StatusCode::EVMC_OUT_OF_GAS);
                }
                let data = state
                    .memory
                    .get_mut_slice(offset, len, &mut state.gas_left)?;
                // TODO revert state changes
                // gas_refund = original_gas_refund;
                state.output = Some(data.to_owned());
                state.step_status_code = StepStatusCode::EVMC_STEP_REVERTED;
                state.status_code = StatusCode::EVMC_REVERT;
                state.code_state.next();
                break;
            }
            Opcode::Invalid => {
                check_min_revision(Revision::EVMC_HOMESTEAD, state.revision)?;
                return Err(StatusCode::EVMC_INVALID_INSTRUCTION);
            }
            Opcode::SelfDestruct => {
                check_not_read_only(&state)?;
                consume_gas(&mut state.gas_left, 5000)?;
                let [addr] = state.stack.pop()?;
                let addr = addr.into();

                let tx_context = state.context.get_tx_context();
                if state.revision >= Revision::EVMC_BERLIN && addr != tx_context.tx_origin
                        //&& addr != tx_context.tx_to // TODO
                        && !(state.revision >= Revision::EVMC_SHANGHAI && addr == tx_context.block_coinbase) && state.context.access_account(&addr) == AccessStatus::EVMC_ACCESS_COLD
                {
                    consume_gas(&mut state.gas_left, 2600)?;
                }

                if u256::from(state.context.get_balance(state.message.recipient())) > u256::ZERO
                    && !state.context.account_exists(&addr)
                {
                    consume_gas(&mut state.gas_left, 25000)?;
                }

                let destructed = state.context.selfdestruct(state.message.recipient(), &addr);
                if state.revision <= Revision::EVMC_BERLIN && destructed {
                    state.gas_refund += 24000;
                }

                state.step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                state.code_state.next();
                break;
            }
        }
    }

    Ok(RunResult {
        step_status_code: state.step_status_code,
        status_code: state.status_code,
        revision: state.revision,
        code_state: state.code_state,
        gas_left: state.gas_left,
        gas_refund: state.gas_refund,
        output: state.output,
        stack: state.stack,
        memory: state.memory,
        last_call_return_data: state.last_call_return_data,
    })
}

fn sstore(state: &mut RunState) -> Result<(), StatusCode> {
    check_not_read_only(state)?;

    if state.revision >= Revision::EVMC_ISTANBUL && state.gas_left <= 2300 {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    let [key, value] = state.stack.pop()?;
    let key = key.into();
    let addr = state.message.recipient();

    let (dyn_gas_1, dyn_gas_2, dyn_gas_3, refund_1, refund_2, refund_3) =
        if state.revision >= Revision::EVMC_LONDON {
            (100, 2900, 20000, 5000 - 2100 - 100, 4800, 20000 - 100)
        } else if state.revision >= Revision::EVMC_BERLIN {
            (100, 2900, 20000, 5000 - 2100 - 100, 15000, 20000 - 100)
        } else if state.revision >= Revision::EVMC_ISTANBUL {
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

    let status = state.context.set_storage(addr, &key, &value.into());
    let (mut dyn_gas, gas_refund_change) = match status {
        StorageStatus::EVMC_STORAGE_ASSIGNED => (dyn_gas_1, 0),
        StorageStatus::EVMC_STORAGE_ADDED => (dyn_gas_3, 0),
        StorageStatus::EVMC_STORAGE_DELETED => (dyn_gas_2, refund_2),
        StorageStatus::EVMC_STORAGE_MODIFIED => (dyn_gas_2, 0),
        StorageStatus::EVMC_STORAGE_DELETED_ADDED => (dyn_gas_1, -refund_2),
        StorageStatus::EVMC_STORAGE_MODIFIED_DELETED => (dyn_gas_1, refund_2),
        StorageStatus::EVMC_STORAGE_DELETED_RESTORED => (dyn_gas_1, -refund_2 + refund_1),
        StorageStatus::EVMC_STORAGE_ADDED_DELETED => (dyn_gas_1, refund_3),
        StorageStatus::EVMC_STORAGE_MODIFIED_RESTORED => (dyn_gas_1, refund_1),
    };
    if state.revision >= Revision::EVMC_BERLIN
        && state.context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD
    {
        dyn_gas += 2100;
    }
    consume_gas(&mut state.gas_left, dyn_gas)?;
    state.gas_refund += gas_refund_change;
    state.code_state.next();
    Ok(())
}

fn push(len: usize, state: &mut RunState) -> Result<(), StatusCode> {
    consume_gas(&mut state.gas_left, 3)?;
    state.code_state.next();
    state.stack.push(state.code_state.get_push_data(len))?;
    Ok(())
}

fn dup(nth: usize, state: &mut RunState) -> Result<(), StatusCode> {
    consume_gas(&mut state.gas_left, 3)?;
    state.stack.push(state.stack.nth(nth - 1)?)?;
    state.code_state.next();
    Ok(())
}

fn swap(nth: usize, state: &mut RunState) -> Result<(), StatusCode> {
    consume_gas(&mut state.gas_left, 3)?;
    state.stack.swap_with_top(nth)?;
    state.code_state.next();
    Ok(())
}

fn log<const N: usize>(state: &mut RunState) -> Result<(), StatusCode> {
    check_not_read_only(state)?;
    consume_gas(&mut state.gas_left, 375)?;
    let [offset, len] = state.stack.pop()?;
    let topics: [u256; N] = state.stack.pop()?;
    let (len, len_overflow) = len.into_u64_with_overflow();
    let (len8, len8_overflow) = len.overflowing_mul(8);
    let (cost, cost_overflow) = (375 * N as u64).overflowing_add(len8);
    if len_overflow || len8_overflow || cost_overflow {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    consume_gas(&mut state.gas_left, cost)?;

    let data = state
        .memory
        .get_mut_slice(offset, len, &mut state.gas_left)?;
    // SAFETY:
    // [u256] is a newtype of [Uint256] with repr(transparent) which guarantees the same memory
    // layout.
    let topics = unsafe { mem::transmute::<&[u256], &[Uint256]>(topics.as_slice()) };
    state
        .context
        .emit_log(state.message.recipient(), data, topics);
    state.code_state.next();
    Ok(())
}

fn create(state: &mut RunState) -> Result<(), StatusCode> {
    create_or_create2::<false>(state)
}

fn create2(state: &mut RunState) -> Result<(), StatusCode> {
    create_or_create2::<true>(state)
}

fn create_or_create2<const CREATE2: bool>(state: &mut RunState) -> Result<(), StatusCode> {
    consume_gas(&mut state.gas_left, 32000)?;
    check_not_read_only(state)?;
    let [value, offset, len] = state.stack.pop()?;
    let salt = if CREATE2 {
        state.stack.pop::<1>()?[0]
    } else {
        u256::ZERO // ignored
    };
    let (len, len_overflow) = len.into_u64_with_overflow();
    if len_overflow {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }

    let init_code_word_size = word_size(len)?;
    if state.revision >= Revision::EVMC_SHANGHAI {
        const MAX_INIT_CODE_LEN: u64 = 2 * 24576;
        if len > MAX_INIT_CODE_LEN {
            return Err(StatusCode::EVMC_OUT_OF_GAS);
        }
        let init_code_cost = 2 * init_code_word_size; // does not overflow
        consume_gas(&mut state.gas_left, init_code_cost)?;
    }
    if CREATE2 {
        let hash_cost = 6 * init_code_word_size; // does not overflow
        consume_gas(&mut state.gas_left, hash_cost)?;
    }

    let init_code = state
        .memory
        .get_mut_slice(offset, len, &mut state.gas_left)?;

    if value > state.context.get_balance(state.message.recipient()).into() {
        state.last_call_return_data = None;
        state.stack.push(u256::ZERO)?;
        state.code_state.next();
        return Ok(());
    }

    let gas_limit = state.gas_left - state.gas_left / 64;
    consume_gas(&mut state.gas_left, gas_limit)?;

    let message = ExecutionMessage::new(
        if CREATE2 {
            MessageKind::EVMC_CREATE2
        } else {
            MessageKind::EVMC_CREATE
        },
        state.message.flags(),
        state.message.depth() + 1,
        gas_limit as i64,
        u256::ZERO.into(), // ignored
        *state.message.recipient(),
        Some(init_code),
        value.into(),
        salt.into(),
        u256::ZERO.into(), // ignored
        None,
    );
    let result = state.context.call(&message);

    state.gas_left += result.gas_left() as u64;
    state.gas_refund += result.gas_refund();

    if result.status_code() == StatusCode::EVMC_SUCCESS {
        let Some(addr) = result.create_address() else {
            return Err(StatusCode::EVMC_INTERNAL_ERROR);
        };

        state.last_call_return_data = None;
        state.stack.push(addr)?;
    } else {
        state.last_call_return_data = result.output().map(ToOwned::to_owned);
        state.stack.push(u256::ZERO)?;
    }
    state.code_state.next();
    Ok(())
}

fn call(state: &mut RunState) -> Result<(), StatusCode> {
    call_or_call_code::<false>(state)
}

fn call_code(state: &mut RunState) -> Result<(), StatusCode> {
    call_or_call_code::<true>(state)
}

fn call_or_call_code<const CODE: bool>(state: &mut RunState) -> Result<(), StatusCode> {
    if state.revision < Revision::EVMC_BERLIN {
        consume_gas(&mut state.gas_left, 700)?;
    }
    let [gas, addr, value, args_offset, args_len, ret_offset, ret_len] = state.stack.pop()?;

    if !CODE && value != u256::ZERO {
        check_not_read_only(state)?;
    }

    let addr = addr.into();
    let (args_len, args_len_overflow) = args_len.into_u64_with_overflow();
    let (ret_len, ret_len_overflow) = ret_len.into_u64_with_overflow();
    if args_len_overflow || ret_len_overflow {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }

    consume_address_access_cost(&addr, state)?;
    consume_positive_value_cost(&value, &mut state.gas_left)?;
    if !CODE {
        consume_value_to_empty_account_cost(&value, &addr, state)?;
    }
    // access slice to consume potential memory expansion cost but drop it so that we can get
    // another mutable reference into memory for input
    let _dest = state
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut state.gas_left)?;
    let input = state
        .memory
        .get_mut_slice(args_offset, args_len, &mut state.gas_left)?;

    let limit = state.gas_left - state.gas_left / 64;
    let mut endowment = gas.into_u64_saturating();
    if state.revision >= Revision::EVMC_TANGERINE_WHISTLE {
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
    }
    let stipend = if value == u256::ZERO { 0 } else { 2300 };
    state.gas_left += stipend;

    if value > u256::from(state.context.get_balance(state.message.recipient())) {
        state.last_call_return_data = None;
        state.stack.push(u256::ZERO)?;
        state.code_state.next();
        return Ok(());
    }

    let call_message = if CODE {
        ExecutionMessage::new(
            MessageKind::EVMC_CALLCODE,
            state.message.flags(),
            state.message.depth() + 1,
            (endowment + stipend) as i64,
            *state.message.recipient(),
            *state.message.recipient(),
            Some(input),
            value.into(),
            u256::ZERO.into(), // ignored
            addr,
            None,
        )
    } else {
        ExecutionMessage::new(
            MessageKind::EVMC_CALL,
            state.message.flags(),
            state.message.depth() + 1,
            (endowment + stipend) as i64,
            addr,
            *state.message.recipient(),
            Some(input),
            value.into(),
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            None,
        )
    };

    let result = state.context.call(&call_message);
    state.last_call_return_data = result.output().map(ToOwned::to_owned);
    let dest = state
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut state.gas_left)?;
    if let Some(output) = &state.last_call_return_data {
        let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
        dest[..min_len].copy_from_slice(&output[..min_len]);
    }

    state.gas_left += result.gas_left() as u64;
    consume_gas(&mut state.gas_left, endowment)?;
    consume_gas(&mut state.gas_left, stipend)?;
    state.gas_refund += result.gas_refund();

    state
        .stack
        .push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
    state.code_state.next();
    Ok(())
}

fn static_call(state: &mut RunState) -> Result<(), StatusCode> {
    static_or_delegate_call::<false>(state)
}

fn delegate_call(state: &mut RunState) -> Result<(), StatusCode> {
    static_or_delegate_call::<true>(state)
}

fn static_or_delegate_call<const DELEGATE: bool>(state: &mut RunState) -> Result<(), StatusCode> {
    if state.revision < Revision::EVMC_BERLIN {
        consume_gas(&mut state.gas_left, 700)?;
    }
    let [gas, addr, args_offset, args_len, ret_offset, ret_len] = state.stack.pop()?;

    let addr = addr.into();
    let (args_len, args_len_overflow) = args_len.into_u64_with_overflow();
    let (ret_len, ret_len_overflow) = ret_len.into_u64_with_overflow();
    if args_len_overflow || ret_len_overflow {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }

    consume_address_access_cost(&addr, state)?;
    // access slice to consume potential memory expansion cost but drop it so that we can get
    // another mutable reference into memory for input
    let _dest = state
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut state.gas_left)?;
    let input = state
        .memory
        .get_mut_slice(args_offset, args_len, &mut state.gas_left)?;

    let limit = state.gas_left - state.gas_left / 64;
    let mut endowment = gas.into_u64_saturating();
    if state.revision >= Revision::EVMC_TANGERINE_WHISTLE {
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
    }

    let call_message = if DELEGATE {
        ExecutionMessage::new(
            MessageKind::EVMC_DELEGATECALL,
            state.message.flags(),
            state.message.depth() + 1,
            endowment as i64,
            *state.message.recipient(),
            *state.message.sender(),
            Some(input),
            *state.message.value(),
            u256::ZERO.into(), // ignored
            addr,
            None,
        )
    } else {
        ExecutionMessage::new(
            MessageKind::EVMC_CALL,
            MessageFlags::EVMC_STATIC as u32,
            state.message.depth() + 1,
            endowment as i64,
            addr,
            *state.message.recipient(),
            Some(input),
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            None,
        )
    };

    let result = state.context.call(&call_message);
    state.last_call_return_data = result.output().map(ToOwned::to_owned);
    let dest = state
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut state.gas_left)?;
    if let Some(output) = &state.last_call_return_data {
        let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
        dest[..min_len].copy_from_slice(&output[..min_len]);
    }

    state.gas_left += result.gas_left() as u64;
    consume_gas(&mut state.gas_left, endowment)?;
    state.gas_refund += result.gas_refund();

    state
        .stack
        .push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
    state.code_state.next();
    Ok(())
}

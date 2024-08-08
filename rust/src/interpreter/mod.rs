use std::{cmp::min, iter, mem, slice};

use evmc_vm::{
    AccessStatus, Address, ExecutionContext, ExecutionMessage, MessageFlags, Revision, StatusCode,
    StepResult, StepStatusCode, StorageStatus, Uint256,
};
use sha3::{Digest, Keccak256};

use crate::types::{opcode, u256};

mod code_state;
pub use code_state::CodeState;

#[allow(clippy::too_many_arguments)]
pub fn run(
    revision: Revision,
    message: &ExecutionMessage,
    context: &mut ExecutionContext,
    mut step_status_code: StepStatusCode,
    mut code_state: CodeState,
    mut gas_refund: i64,
    mut stack: Vec<u256>,
    mut memory: Vec<u8>,
    mut last_call_return_data: Option<Vec<u8>>,
    steps: Option<i32>,
) -> Result<StepResult, (StepStatusCode, StatusCode)> {
    let mut gas_left = message.gas() as u64;
    let mut status_code = StatusCode::EVMC_SUCCESS;
    let mut output = None;

    println!("running test");
    for _ in 0..steps.unwrap_or(i32::MAX) {
        let Some(op) = code_state.get() else {
            return Err((StepStatusCode::EVMC_STEP_FAILED, StatusCode::EVMC_FAILURE));
        };
        match op {
            opcode::STOP => {
                step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                status_code = StatusCode::EVMC_SUCCESS;
                break;
            }
            opcode::ADD => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top + top2);
                code_state.next();
            }
            opcode::MUL => {
                consume_gas::<5>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top * top2);
                code_state.next();
            }
            opcode::SUB => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top - top2);
                code_state.next();
            }
            opcode::DIV => {
                consume_gas::<5>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top / top2);
                code_state.next();
            }
            opcode::SDIV => {
                consume_gas::<5>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top.sdiv(top2));
                code_state.next();
            }
            opcode::MOD => {
                consume_gas::<5>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top % top2);
                code_state.next();
            }
            opcode::SMOD => {
                consume_gas::<5>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top.srem(top2));
                code_state.next();
            }
            opcode::ADDMOD => {
                consume_gas::<8>(&mut gas_left)?;
                let [top, top2, top3] = pop_from_stack(&mut stack)?;
                stack.push(u256::addmod(top, top2, top3));
                code_state.next();
            }
            opcode::MULMOD => {
                consume_gas::<8>(&mut gas_left)?;
                let [top, top2, top3] = pop_from_stack(&mut stack)?;
                stack.push(u256::mulmod(top, top2, top3));
                code_state.next();
            }
            opcode::EXP => {
                consume_gas::<10>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                let top2_bytes: [u8; 32] = *top2;
                let mut cost_multiplier = 32;
                for byte in top2_bytes.into_iter() {
                    if byte == 0 {
                        cost_multiplier -= 1;
                    } else {
                        break;
                    }
                }
                let dyn_gas = 50 * cost_multiplier;
                consume_dyn_gas(&mut gas_left, dyn_gas)?;
                stack.push(top.pow(top2));
                code_state.next();
            }
            opcode::SIGNEXTEND => {
                consume_gas::<5>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(u256::signextend(top, top2));
                code_state.next();
            }
            opcode::LT => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(((top < top2) as u8).into());
                code_state.next();
            }
            opcode::GT => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(((top > top2) as u8).into());
                code_state.next();
            }
            opcode::SLT => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(((top.slt(&top2)) as u8).into());
                code_state.next();
            }
            opcode::SGT => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(((top.sgt(&top2)) as u8).into());
                code_state.next();
            }
            opcode::EQ => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(((top == top2) as u8).into());
                code_state.next();
            }
            opcode::ISZERO => {
                consume_gas::<3>(&mut gas_left)?;
                let [top] = pop_from_stack(&mut stack)?;
                stack.push(((top == u256::ZERO) as u8).into());
                code_state.next();
            }
            opcode::AND => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top & top2);
                code_state.next();
            }
            opcode::OR => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top | top2);
                code_state.next();
            }
            opcode::XOR => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top ^ top2);
                code_state.next();
            }
            opcode::NOT => {
                consume_gas::<3>(&mut gas_left)?;
                let [top] = pop_from_stack(&mut stack)?;
                stack.push(!top);
                code_state.next();
            }
            opcode::BYTE => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top2.byte(top));
                code_state.next();
            }
            opcode::SHL => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top2 << top);
                code_state.next();
            }
            opcode::SHR => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top2 >> top);
                code_state.next();
            }
            opcode::SAR => {
                consume_gas::<3>(&mut gas_left)?;
                let [top, top2] = pop_from_stack(&mut stack)?;
                stack.push(top2.sar(top));
                code_state.next();
            }
            opcode::SHA3 => {
                consume_gas::<30>(&mut gas_left)?;
                let [offset, len] = pop_from_stack(&mut stack)?;

                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                    ));
                }
                consume_dyn_gas(&mut gas_left, 6 * word_size(len))?;

                let memory_access = access_memory_slice(&mut memory, offset, len, &mut gas_left)?;
                let mut hasher = Keccak256::new();
                hasher.update(memory_access);
                let result = hasher.finalize();
                stack.push(result.as_slice().try_into().unwrap());
                code_state.next();
            }
            opcode::ADDRESS => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(message.recipient().into());
                code_state.next();
            }
            opcode::BALANCE => {
                let [addr] = pop_from_stack(&mut stack)?;
                let addr = addr.into();
                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                stack.push(context.get_balance(&addr).into());
                code_state.next();
            }
            opcode::ORIGIN => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_tx_context().tx_origin.into());
                code_state.next();
            }
            opcode::CALLER => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(message.sender().into());
                code_state.next();
            }
            opcode::CALLVALUE => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push((*message.value()).into());
                code_state.next();
            }
            opcode::CALLDATALOAD => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset] = pop_from_stack(&mut stack)?;
                let (offset, overflow) = offset.into_u64_with_overflow();
                let offset = offset as usize;
                let call_data = message.input().unwrap();
                if overflow || offset >= call_data.len() {
                    stack.push(u256::ZERO);
                } else {
                    let end = min(call_data.len(), offset + 32);
                    let mut bytes = [0; 32];
                    bytes[..end - offset].copy_from_slice(&call_data[offset..end]);
                    stack.push(bytes.into());
                }
                code_state.next();
            }
            opcode::CALLDATASIZE => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                let call_data = message.input().unwrap();
                stack.push(call_data.len().into());
                code_state.next();
            }
            opcode::PUSH0 => {
                check_min_revision(Revision::EVMC_SHANGHAI, revision)?;
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(u256::ZERO);
                code_state.next();
            }
            opcode::CALLDATACOPY => {
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = pop_from_stack(&mut stack)?;

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
                    let dest = access_memory_slice(&mut memory, dest_offset, len, &mut gas_left)?;
                    copy_slice_padded(src, dest, &mut gas_left)?;
                }

                code_state.next();
            }
            opcode::CODESIZE => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(code_state.code_len().into());
                code_state.next();
            }
            opcode::CODECOPY => {
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = pop_from_stack(&mut stack)?;

                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        return Err((
                            StepStatusCode::EVMC_STEP_FAILED,
                            StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                        ));
                    }

                    let src = get_slice_within_bounds(&code_state, offset, len);
                    let dest = access_memory_slice(&mut memory, dest_offset, len, &mut gas_left)?;
                    copy_slice_padded(src, dest, &mut gas_left)?;
                }

                code_state.next();
            }
            opcode::GASPRICE => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_tx_context().tx_gas_price.into());
                code_state.next();
            }
            opcode::EXTCODESIZE => {
                let [addr] = pop_from_stack(&mut stack)?;
                let addr = addr.into();
                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                stack.push(context.get_code_size(&addr).into());
                code_state.next();
            }
            opcode::EXTCODECOPY => {
                let [addr, dest_offset, offset, len] = pop_from_stack(&mut stack)?;
                let addr = addr.into();

                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        return Err((
                            StepStatusCode::EVMC_STEP_FAILED,
                            StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                        ));
                    }

                    let src_len = context.get_code_size(&addr) as u64;
                    let dest = access_memory_slice(&mut memory, dest_offset, len, &mut gas_left)?;
                    let (offset, offset_overflow) = offset.into_u64_with_overflow();
                    if offset_overflow || offset >= src_len {
                        copy_slice_padded(&[], dest, &mut gas_left)?;
                    } else if offset + len >= src_len {
                        let copy_end = (src_len - offset) as usize;
                        consume_copy_cost(&mut gas_left, len)?;
                        context.copy_code(&addr, offset as usize, &mut dest[..copy_end]);
                        for byte in &mut dest[copy_end..] {
                            *byte = 0;
                        }
                    } else {
                        consume_copy_cost(&mut gas_left, len)?;
                        context.copy_code(&addr, offset as usize, dest);
                    }
                }

                code_state.next();
            }
            opcode::RETURNDATASIZE => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(
                    last_call_return_data
                        .as_ref()
                        .map(|data| data.len())
                        .unwrap_or_default()
                        .into(),
                );
                code_state.next();
            }
            opcode::RETURNDATACOPY => {
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = pop_from_stack(&mut stack)?;

                let src = last_call_return_data.as_deref().unwrap_or(&[]);
                let (offset, offset_overflow) = offset.into_u64_with_overflow();
                let (len, len_overflow) = len.into_u64_with_overflow();
                if offset_overflow || len_overflow || offset + len >= src.len() as u64 {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                    ));
                }

                if len != 0 {
                    let src = get_slice_within_bounds(src, offset.into(), len);
                    let dest = access_memory_slice(&mut memory, dest_offset, len, &mut gas_left)?;
                    copy_slice_padded(src, dest, &mut gas_left)?;
                }

                code_state.next();
            }
            opcode::EXTCODEHASH => {
                let [addr] = pop_from_stack(&mut stack)?;
                let addr = addr.into();
                consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                stack.push(context.get_code_hash(&addr).into());
                code_state.next();
            }
            opcode::BLOCKHASH => {
                consume_gas::<20>(&mut gas_left)?;
                let [block_number] = pop_from_stack(&mut stack)?;
                let current_block_number = context.get_tx_context().block_number;
                let (idx, idx_overflow) = block_number.into_u64_with_overflow();
                if idx_overflow || idx > current_block_number as u64 + 255 {
                    stack.push(u256::ZERO);
                } else {
                    stack.push(context.get_block_hash(idx as i64).into());
                }
                code_state.next();
            }
            opcode::COINBASE => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_tx_context().block_coinbase.into());
                code_state.next();
            }
            opcode::TIMESTAMP => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push((context.get_tx_context().block_timestamp as u64).into());
                code_state.next();
            }
            opcode::NUMBER => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push((context.get_tx_context().block_number as u64).into());
                code_state.next();
            }
            opcode::PREVRANDAO => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_tx_context().block_prev_randao.into());
                code_state.next();
            }
            opcode::GASLIMIT => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push((context.get_tx_context().block_gas_limit as u64).into());
                code_state.next();
            }
            opcode::CHAINID => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_tx_context().chain_id.into());
                code_state.next();
            }
            opcode::SELFBALANCE => {
                consume_gas::<5>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_balance(message.recipient()).into());
                code_state.next();
            }
            opcode::BASEFEE => {
                check_min_revision(Revision::EVMC_LONDON, revision)?;
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_tx_context().block_base_fee.into());
                code_state.next();
            }
            opcode::BLOBHASH => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<3>(&mut gas_left)?;
                let [idx] = pop_from_stack(&mut stack)?;
                let (idx, idx_overflow) = idx.into_u64_with_overflow();
                let idx = idx as usize;
                let count = context.get_tx_context().blob_hashes_count;
                if !idx_overflow && idx < count {
                    // TODO create new ExecutionTxContext type and do this conversion in mod ffi
                    let hashes = context.get_tx_context().blob_hashes;
                    let hashes: &[Uint256] = if hashes.is_null() {
                        assert_eq!(count, 0);
                        &[]
                    } else {
                        // SAFETY:
                        // hashes is not null and count > 0
                        unsafe { slice::from_raw_parts(hashes, count) }
                    };

                    let hash = hashes[idx];

                    stack.push(hash.into());
                } else {
                    stack.push(u256::ZERO);
                }
                code_state.next();
            }
            opcode::BLOBBASEFEE => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(context.get_tx_context().blob_base_fee.into());
                code_state.next();
            }
            opcode::POP => {
                consume_gas::<2>(&mut gas_left)?;
                let [_] = pop_from_stack(&mut stack)?;
                code_state.next();
            }
            opcode::MLOAD => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset] = pop_from_stack(&mut stack)?;

                let memory_access: &[u8] = access_memory_word(&mut memory, offset, &mut gas_left)?;
                stack.push(memory_access.try_into().unwrap());
                code_state.next();
            }
            opcode::MSTORE => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset, value] = pop_from_stack(&mut stack)?;

                let bytes: [u8; 32] = value.into();
                let memory_access = access_memory_word(&mut memory, offset, &mut gas_left)?;
                memory_access.copy_from_slice(&bytes);
                code_state.next();
            }
            opcode::MSTORE8 => {
                consume_gas::<3>(&mut gas_left)?;
                let [offset, value] = pop_from_stack(&mut stack)?;

                let memory_access = access_memory_byte(&mut memory, offset, &mut gas_left)?;
                *memory_access = value[31];
                code_state.next();
            }
            opcode::SLOAD => {
                if revision < Revision::EVMC_BERLIN {
                    consume_gas::<800>(&mut gas_left)?;
                }
                let [key] = pop_from_stack(&mut stack)?;
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
                stack.push(value.into());
                code_state.next();
            }
            opcode::SSTORE => {
                if revision >= Revision::EVMC_BYZANTIUM {
                    check_not_read_only(message)?;
                }
                let [key, value] = pop_from_stack(&mut stack)?;
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
                // TODO ct does not like this check
                //if revision >= Revision::EVMC_ISTANBUL && gas_left <= 2300 {
                //return Err((
                //StepStatusCode::EVMC_STEP_FAILED,
                //StatusCode::EVMC_OUT_OF_GAS,
                //));
                //}
                gas_refund += gas_refund_change;
                code_state.next();
            }
            opcode::JUMP => {
                consume_gas::<8>(&mut gas_left)?;
                let [dest] = pop_from_stack(&mut stack)?;
                code_state.try_jump(dest)?;
            }
            opcode::JUMPI => {
                consume_gas::<10>(&mut gas_left)?;
                let [dest, cond] = pop_from_stack(&mut stack)?;
                if cond == u256::ZERO {
                    code_state.next();
                } else {
                    code_state.try_jump(dest)?;
                }
            }
            opcode::PC => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(code_state.pc().into());
                code_state.next();
            }
            opcode::MSIZE => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(memory.len().into());
                code_state.next();
            }
            opcode::GAS => {
                consume_gas::<2>(&mut gas_left)?;
                check_stack_overflow::<1>(&stack)?;
                stack.push(gas_left.into());
                code_state.next();
            }
            opcode::JUMPDEST => {
                consume_gas::<1>(&mut gas_left)?;
                code_state.next();
            }
            opcode::TLOAD => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<100>(&mut gas_left)?;
                let [key] = pop_from_stack(&mut stack)?;
                let addr = message.recipient();
                let value = context.get_transient_storage(addr, &key.into());
                stack.push(value.into());
                code_state.next();
            }
            opcode::TSTORE => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                check_not_read_only(message)?;
                consume_gas::<100>(&mut gas_left)?;
                let [key, value] = pop_from_stack(&mut stack)?;
                let addr = message.recipient();
                context.set_transient_storage(addr, &key.into(), &value.into());
                code_state.next();
            }
            opcode::MCOPY => {
                check_min_revision(Revision::EVMC_CANCUN, revision)?;
                consume_gas::<3>(&mut gas_left)?;
                let [dest_offset, offset, len] = pop_from_stack(&mut stack)?;

                if len != u256::ZERO {
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    if len_overflow {
                        return Err((
                            StepStatusCode::EVMC_STEP_FAILED,
                            StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                        ));
                    }
                    consume_copy_cost(&mut gas_left, len)?;

                    let src =
                        access_memory_slice(&mut memory, offset, len, &mut gas_left)?.to_owned();
                    let dest = access_memory_slice(&mut memory, dest_offset, len, &mut gas_left)?;
                    dest.copy_from_slice(&src);
                }

                code_state.next();
            }
            opcode::PUSH1 => push::<1>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH2 => push::<2>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH3 => push::<3>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH4 => push::<4>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH5 => push::<5>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH6 => push::<6>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH7 => push::<7>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH8 => push::<8>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH9 => push::<9>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH10 => push::<10>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH11 => push::<11>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH12 => push::<12>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH13 => push::<13>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH14 => push::<14>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH15 => push::<15>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH16 => push::<16>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH17 => push::<17>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH18 => push::<18>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH19 => push::<19>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH20 => push::<20>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH21 => push::<21>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH22 => push::<22>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH23 => push::<23>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH24 => push::<24>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH25 => push::<25>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH26 => push::<26>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH27 => push::<27>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH28 => push::<28>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH29 => push::<29>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH30 => push::<30>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH31 => push::<31>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::PUSH32 => push::<32>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP1 => dup::<1>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP2 => dup::<2>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP3 => dup::<3>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP4 => dup::<4>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP5 => dup::<5>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP6 => dup::<6>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP7 => dup::<7>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP8 => dup::<8>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP9 => dup::<9>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP10 => dup::<10>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP11 => dup::<11>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP12 => dup::<12>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP13 => dup::<13>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP14 => dup::<14>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP15 => dup::<15>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::DUP16 => dup::<16>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP1 => swap::<1>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP2 => swap::<2>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP3 => swap::<3>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP4 => swap::<4>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP5 => swap::<5>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP6 => swap::<6>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP7 => swap::<7>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP8 => swap::<8>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP9 => swap::<9>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP10 => swap::<10>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP11 => swap::<11>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP12 => swap::<12>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP13 => swap::<13>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP14 => swap::<14>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP15 => swap::<15>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::SWAP16 => swap::<16>(&mut code_state, &mut stack, &mut gas_left)?,
            opcode::LOG0 => log::<0>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            opcode::LOG1 => log::<1>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            opcode::LOG2 => log::<2>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            opcode::LOG3 => log::<3>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            opcode::LOG4 => log::<4>(
                &mut code_state,
                &mut stack,
                &mut memory,
                context,
                message,
                revision,
                &mut gas_left,
            )?,
            opcode::CREATE => unimplemented!(),
            opcode::CALL => unimplemented!(),
            opcode::CALLCODE => unimplemented!(),
            opcode::RETURN => {
                let [offset, len] = pop_from_stack(&mut stack)?;
                let (len, len_overflow) = len.into_u64_with_overflow();
                if len_overflow {
                    return Err((
                        StepStatusCode::EVMC_STEP_FAILED,
                        StatusCode::EVMC_OUT_OF_GAS,
                    ));
                }
                let memory_access = access_memory_slice(&mut memory, offset, len, &mut gas_left)?;
                output = Some(memory_access.to_owned());
                step_status_code = StepStatusCode::EVMC_STEP_RETURNED;
                code_state.next();
                break;
            }
            opcode::DELEGATECALL => unimplemented!(),
            opcode::CREATE2 => unimplemented!(),
            opcode::STATICCALL => unimplemented!(),
            opcode::REVERT => unimplemented!(),
            opcode::INVALID => {
                check_min_revision(Revision::EVMC_HOMESTEAD, revision)?;
                return Err((
                    StepStatusCode::EVMC_STEP_FAILED,
                    StatusCode::EVMC_INVALID_INSTRUCTION,
                ));
            }
            opcode::SELFDESTRUCT => {
                if revision >= Revision::EVMC_BYZANTIUM {
                    check_not_read_only(message)?;
                }
                consume_gas::<5000>(&mut gas_left)?;
                let [addr] = pop_from_stack(&mut stack)?;
                let addr = addr.into();

                //consume_address_access_cost(&mut gas_left, &addr, context, revision)?;
                let tx_context = context.get_tx_context();
                if revision >= Revision::EVMC_BERLIN {
                    if addr != tx_context.tx_origin
                        //&& addr != tx_context.tx_to // TODO
                        && !(revision >= Revision::EVMC_SHANGHAI && addr == tx_context.block_coinbase)
                        && context.access_account(&addr) == AccessStatus::EVMC_ACCESS_COLD
                    {
                        consume_gas::<2600>(&mut gas_left)?;
                    }
                }

                if !context.account_exists(&addr) {
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
            op => {
                println!("invalid opcode 0x{op:x?}");
                return Err((
                    StepStatusCode::EVMC_STEP_FAILED,
                    StatusCode::EVMC_INVALID_INSTRUCTION,
                ));
            }
        }
    }

    stack.reverse();
    Ok(StepResult::new(
        step_status_code,
        status_code,
        revision,
        code_state.pc() as u64,
        gas_left as i64,
        gas_refund,
        output,
        // SAFETY
        // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
        // layout.
        unsafe { mem::transmute::<Vec<u256>, Vec<Uint256>>(stack) },
        memory,
        last_call_return_data,
    ))
}

fn push<const N: usize>(
    code_state: &mut CodeState,
    stack: &mut Vec<u256>,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_gas::<3>(gas_left)?;
    check_stack_overflow::<1>(stack)?;

    let data = code_state.try_get_push_data(N)?;
    stack.push(data.try_into().unwrap());

    Ok(())
}

fn dup<const N: usize>(
    code_state: &mut CodeState,
    stack: &mut Vec<u256>,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_gas::<3>(gas_left)?;
    check_stack_overflow::<1>(stack)?;
    let nth = nth_ref_from_stack::<N>(stack)?;
    let nth = *nth;

    stack.push(nth);
    code_state.next();

    Ok(())
}

fn swap<const N: usize>(
    code_state: &mut CodeState,
    stack: &mut [u256],
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_gas::<3>(gas_left)?;
    let len = stack.len();
    if len < N + 1 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_UNDERFLOW,
        ));
    }

    stack.swap(len - 1, len - 1 - N);
    code_state.next();

    Ok(())
}

fn log<const N: usize>(
    code_state: &mut CodeState,
    stack: &mut Vec<u256>,
    memory: &mut Vec<u8>,
    context: &mut ExecutionContext,
    message: &ExecutionMessage,
    revision: Revision,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    if revision >= Revision::EVMC_BYZANTIUM {
        check_not_read_only(message)?;
    }
    consume_gas::<375>(gas_left)?;
    let [offset, len] = pop_from_stack(stack)?;
    let topics: [u256; N] = pop_from_stack(stack)?;
    let (len, len_overflow) = len.into_u64_with_overflow();
    if len_overflow {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    consume_dyn_gas(gas_left, 375 * N as u64 + 8 * len)?;

    let memory_access = access_memory_slice(memory, offset, len, gas_left)?;
    let topics: &[_; N] = unsafe { mem::transmute(&topics) };
    context.emit_log(message.recipient(), memory_access, topics.as_slice());
    code_state.next();
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
            StatusCode::EVMC_UNDEFINED_INSTRUCTION,
        ));
    }
    Ok(())
}

#[inline(always)]
fn consume_gas<const GAS: u64>(gas_left: &mut u64) -> Result<(), (StepStatusCode, StatusCode)> {
    if *gas_left < GAS {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    *gas_left -= GAS;
    Ok(())
}

#[inline(always)]
fn consume_dyn_gas(gas_left: &mut u64, gas: u64) -> Result<(), (StepStatusCode, StatusCode)> {
    if *gas_left < gas {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_OUT_OF_GAS,
        ));
    }
    *gas_left -= gas;
    Ok(())
}

#[inline(always)]
fn consume_address_access_cost(
    gas_left: &mut u64,
    addr: &Address,
    context: &mut ExecutionContext,
    revision: Revision,
) -> Result<(), (StepStatusCode, StatusCode)> {
    let tx_context = context.get_tx_context();
    if revision >= Revision::EVMC_BERLIN {
        if *addr != tx_context.tx_origin
            //&& addr != tx_context.tx_to // TODO
            && !(revision >= Revision::EVMC_SHANGHAI && *addr == tx_context.block_coinbase)
            && context.access_account(addr) == AccessStatus::EVMC_ACCESS_COLD
        {
            consume_gas::<2600>(gas_left)?;
        } else {
            consume_gas::<100>(gas_left)?;
        }
    } else {
        consume_gas::<700>(gas_left)?;
    }
    Ok(())
}

/// consume 3 * minimum_word_size
#[inline(always)]
fn consume_copy_cost(gas_left: &mut u64, len: u64) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_dyn_gas(gas_left, 3 * word_size(len))?;
    Ok(())
}

#[inline(always)]
fn consume_memory_expansion_cost(
    gas_left: &mut u64,
    current_len: u64,
    new_len: u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    fn memory_cost(size: u64) -> Result<u64, (StepStatusCode, StatusCode)> {
        let memory_size_word = word_size(size);
        let Some(pow2) = memory_size_word.checked_pow(2) else {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_OUT_OF_GAS,
            ));
        };
        Ok(pow2 / 512 + (3 * memory_size_word))
    }

    if new_len > current_len {
        let memory_expansion_cost = memory_cost(new_len)? - memory_cost(current_len)?;
        consume_dyn_gas(gas_left, memory_expansion_cost)?;
    }
    Ok(())
}

#[inline(always)]
fn check_stack_overflow<const N: usize>(
    stack: &[u256],
) -> Result<(), (StepStatusCode, StatusCode)> {
    if stack.len() + N > 1024 {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_OVERFLOW,
        ));
    }
    Ok(())
}

#[inline(always)]
fn check_not_read_only(message: &ExecutionMessage) -> Result<(), (StepStatusCode, StatusCode)> {
    let read_only = message.flags() == MessageFlags::EVMC_STATIC as u32;
    if read_only {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STATIC_MODE_VIOLATION,
        ));
    }
    Ok(())
}

#[inline(always)]
fn pop_from_stack<const N: usize>(
    stack: &mut Vec<u256>,
) -> Result<[u256; N], (StepStatusCode, StatusCode)> {
    if stack.len() < N {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_UNDERFLOW,
        ));
    }
    let mut array = [u256::ZERO; N];
    for element in &mut array {
        *element = stack.pop().unwrap();
    }

    Ok(array)
}

#[inline(always)]
fn nth_ref_from_stack<const N: usize>(
    stack: &[u256],
) -> Result<&u256, (StepStatusCode, StatusCode)> {
    if stack.len() < N {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_STACK_UNDERFLOW,
        ));
    }

    Ok(&stack[stack.len() - N])
}

fn expand_memory(
    memory: &mut Vec<u8>,
    new_len: u64,
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    let current_len = memory.len() as u64;
    if new_len > current_len {
        consume_memory_expansion_cost(gas_left, current_len, new_len)?;
        memory.extend(iter::repeat(0).take((new_len - current_len) as usize))
    }
    Ok(())
}

fn access_memory_slice<'m>(
    memory: &'m mut Vec<u8>,
    offset: u256,
    len: u64,
    gas_left: &mut u64,
) -> Result<&'m mut [u8], (StepStatusCode, StatusCode)> {
    if len == 0 {
        return Ok(&mut []);
    }
    let (offset, offset_overflow) = offset.into_u64_with_overflow();
    if offset_overflow {
        return Err((
            StepStatusCode::EVMC_STEP_FAILED,
            StatusCode::EVMC_INVALID_MEMORY_ACCESS,
        ));
    }
    let end = offset + len;
    let new_len = word_size(end) * 32;
    expand_memory(memory, new_len, gas_left)?;

    Ok(&mut memory[offset as usize..end as usize])
}

fn access_memory_word<'m>(
    memory: &'m mut Vec<u8>,
    offset: u256,
    gas_left: &mut u64,
) -> Result<&'m mut [u8], (StepStatusCode, StatusCode)> {
    access_memory_slice(memory, offset, 32u8.into(), gas_left)
}

fn access_memory_byte<'m>(
    memory: &'m mut Vec<u8>,
    offset: u256,
    gas_left: &mut u64,
) -> Result<&'m mut u8, (StepStatusCode, StatusCode)> {
    access_memory_slice(memory, offset, 1u8.into(), gas_left).map(|slice| &mut slice[0])
}

fn word_size(bytes: u64) -> u64 {
    (bytes + 31) / 32
}

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
    if offset + len < data.len() {
        &data[offset..offset + len]
    } else if offset < data.len() {
        &data[offset..]
    } else {
        &[]
    }
}

fn copy_slice_padded(
    src: &[u8],
    dest: &mut [u8],
    gas_left: &mut u64,
) -> Result<(), (StepStatusCode, StatusCode)> {
    consume_copy_cost(gas_left, dest.len() as u64)?;
    dest[..src.len()].copy_from_slice(src);
    for byte in &mut dest[src.len()..] {
        *byte = 0;
    }
    Ok(())
}

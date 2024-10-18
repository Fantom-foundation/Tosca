use std::{cmp::min, mem};

use evmc_vm::{
    AccessStatus, ExecutionMessage, MessageFlags, MessageKind, Revision,
    StatusCode as EvmcStatusCode, StorageStatus, Uint256,
};

use crate::{
    interpreter::{Interpreter, OpResult},
    types::{hash_cache, ExecStatus, ExecutionTxContext, FailStatus},
    u256,
    utils::{check_min_revision, check_not_read_only, word_size, SliceExt},
    ExecutionContextTrait,
};

#[cfg(feature = "jumptable")]
pub fn jumptable_placeholder<E: ExecutionContextTrait>(_i: &mut Interpreter<E>) -> OpResult {
    Err(FailStatus::Failure)
}

pub fn stop<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.exec_status = ExecStatus::Stopped;
    Ok(())
}

pub fn add<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value2, value1] = i.stack.pop()?;
    i.stack.push(value1 + value2)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn mul<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(5)?;
    let [fac2, fac1] = i.stack.pop()?;
    i.stack.push(fac1 * fac2)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn sub<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value2, value1] = i.stack.pop()?;
    i.stack.push(value1 - value2)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn div<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(5)?;
    let [denominator, value] = i.stack.pop()?;
    i.stack.push(value / denominator)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn s_div<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(5)?;
    let [denominator, value] = i.stack.pop()?;
    i.stack.push(value.sdiv(denominator))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn mod_<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(5)?;
    let [denominator, value] = i.stack.pop()?;
    i.stack.push(value % denominator)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn s_mod<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(5)?;
    let [denominator, value] = i.stack.pop()?;
    i.stack.push(value.srem(denominator))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn add_mod<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(8)?;
    let [denominator, value2, value1] = i.stack.pop()?;
    i.stack.push(u256::addmod(value1, value2, denominator))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn mul_mod<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(8)?;
    let [denominator, fac2, fac1] = i.stack.pop()?;
    i.stack.push(u256::mulmod(fac1, fac2, denominator))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn exp<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(10)?;
    let [exp, value] = i.stack.pop()?;
    let byte_size = 32 - exp.into_iter().take_while(|byte| *byte == 0).count() as u64;
    i.gas_left.consume(byte_size * 50)?; // * does not overflow
    i.stack.push(value.pow(exp))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn sign_extend<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(5)?;
    let [value, size] = i.stack.pop()?;
    i.stack.push(u256::signextend(size, value))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn lt<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs < rhs)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn gt<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs > rhs)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn s_lt<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs.slt(&rhs))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn s_gt<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs.sgt(&rhs))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn eq<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs == rhs)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn is_zero<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value] = i.stack.pop()?;
    i.stack.push(value == u256::ZERO)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn and<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs & rhs)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn or<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs | rhs)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn xor<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [rhs, lhs] = i.stack.pop()?;
    i.stack.push(lhs ^ rhs)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn not<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value] = i.stack.pop()?;
    i.stack.push(!value)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn byte<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value, offset] = i.stack.pop()?;
    i.stack.push(value.byte(offset))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn shl<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value, shift] = i.stack.pop()?;
    i.stack.push(value << shift)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn shr<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value, shift] = i.stack.pop()?;
    i.stack.push(value >> shift)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn sar<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value, shift] = i.stack.pop()?;
    i.stack.push(value.sar(shift))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn sha3<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(30)?;
    let [len, offset] = i.stack.pop()?;

    let len = len.try_into().map_err(|_| FailStatus::OutOfGas)?;
    i.gas_left.consume(6 * word_size(len)?)?; // * does not overflow

    let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
    i.stack.push(hash_cache::hash(data))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn address<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.message.recipient())?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn balance<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    if i.revision < Revision::EVMC_BERLIN {
        i.gas_left.consume(700)?;
    }
    let [addr] = i.stack.pop()?;
    let addr = addr.into();
    i.gas_left
        .consume_address_access_cost(&addr, i.revision, i.context)?;
    i.stack.push(i.context.get_balance(&addr))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn origin<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.context.get_tx_context().tx_origin)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn caller<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.message.sender())?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn call_value<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(*i.message.value())?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn call_data_load<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [offset] = i.stack.pop()?;
    let (offset, overflow) = offset.into_u64_with_overflow();
    let offset = offset as usize;
    #[allow(clippy::map_identity)]
    let call_data = i
        .message
        .input()
        .map(
            #[cfg(not(feature = "custom-evmc"))]
            Vec::as_slice,
            #[cfg(feature = "custom-evmc")]
            std::convert::identity,
        )
        .unwrap_or_default();
    if overflow || offset >= call_data.len() {
        i.stack.push(u256::ZERO)?;
    } else {
        let end = min(call_data.len(), offset + 32);
        let mut bytes = u256::ZERO;
        bytes[..end - offset].copy_from_slice(&call_data[offset..end]);
        i.stack.push(bytes)?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn call_data_size<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    let call_data_len = i
        .message
        .input()
        .map(|m| {
            #[allow(clippy::redundant_closure)]
            m.len()
        })
        .unwrap_or_default();
    i.stack.push(call_data_len)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn push0<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_SHANGHAI, i.revision)?;
    i.gas_left.consume(2)?;
    i.stack.push(u256::ZERO)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn call_data_copy<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [len, offset, dest_offset] = i.stack.pop()?;

    if len != u256::ZERO {
        let len = len
            .try_into()
            .map_err(|_| FailStatus::InvalidMemoryAccess)?;

        #[allow(clippy::map_identity)]
        let src = i
            .message
            .input()
            .map(
                #[cfg(not(feature = "custom-evmc"))]
                Vec::as_slice,
                #[cfg(feature = "custom-evmc")]
                std::convert::identity,
            )
            .unwrap_or_default()
            .get_within_bounds(offset, len);
        let dest = i.memory.get_mut_slice(dest_offset, len, &mut i.gas_left)?;
        dest.copy_padded(src, &mut i.gas_left)?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn code_size<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.code_reader.len())?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn code_copy<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [len, offset, dest_offset] = i.stack.pop()?;

    if len != u256::ZERO {
        let len = len.try_into().map_err(|_| FailStatus::OutOfGas)?;

        let src = i.code_reader.get_within_bounds(offset, len);
        let dest = i.memory.get_mut_slice(dest_offset, len, &mut i.gas_left)?;
        dest.copy_padded(src, &mut i.gas_left)?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn gas_price<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.context.get_tx_context().tx_gas_price)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn ext_code_size<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    if i.revision < Revision::EVMC_BERLIN {
        i.gas_left.consume(700)?;
    }
    let [addr] = i.stack.pop()?;
    let addr = addr.into();
    i.gas_left
        .consume_address_access_cost(&addr, i.revision, i.context)?;
    i.stack.push(i.context.get_code_size(&addr))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn ext_code_copy<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    if i.revision < Revision::EVMC_BERLIN {
        i.gas_left.consume(700)?;
    }
    let [len, offset, dest_offset, addr] = i.stack.pop()?;
    let addr = addr.into();

    i.gas_left
        .consume_address_access_cost(&addr, i.revision, i.context)?;
    if len != u256::ZERO {
        let len = len.try_into().map_err(|_| FailStatus::OutOfGas)?;

        let dest = i.memory.get_mut_slice(dest_offset, len, &mut i.gas_left)?;
        let (offset, offset_overflow) = offset.into_u64_with_overflow();
        i.gas_left.consume_copy_cost(len)?;
        let bytes_written = i.context.copy_code(&addr, offset as usize, dest);
        if offset_overflow {
            dest.set_to_zero();
        } else if (bytes_written as u64) < len {
            dest[bytes_written..].set_to_zero();
        }
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn return_data_size<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(
        i.last_call_return_data
            .as_ref()
            .map(Vec::len)
            .unwrap_or_default(),
    )?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn return_data_copy<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [len, offset, dest_offset] = i.stack.pop()?;

    let src = i.last_call_return_data.as_deref().unwrap_or_default();
    let (offset, offset_overflow) = offset.into_u64_with_overflow();
    let (len, len_overflow) = len.into_u64_with_overflow();
    let (end, end_overflow) = offset.overflowing_add(len);
    if offset_overflow || len_overflow || end_overflow || end > src.len() as u64 {
        return Err(FailStatus::InvalidMemoryAccess);
    }

    if len != 0 {
        let src = &src[offset as usize..end as usize];
        let dest = i.memory.get_mut_slice(dest_offset, len, &mut i.gas_left)?;
        dest.copy_padded(src, &mut i.gas_left)?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn ext_code_hash<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    if i.revision < Revision::EVMC_BERLIN {
        i.gas_left.consume(700)?;
    }
    let [addr] = i.stack.pop()?;
    let addr = addr.into();
    i.gas_left
        .consume_address_access_cost(&addr, i.revision, i.context)?;
    i.stack.push(i.context.get_code_hash(&addr))?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn block_hash<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(20)?;
    let [block_number] = i.stack.pop()?;
    i.stack.push(
        block_number
            .try_into()
            .map(|idx: u64| i.context.get_block_hash(idx as i64))
            .unwrap_or(u256::ZERO.into()),
    )?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn coinbase<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.context.get_tx_context().block_coinbase)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn timestamp<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack
        .push(i.context.get_tx_context().block_timestamp as u64)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn number<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack
        .push(i.context.get_tx_context().block_number as u64)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn prev_randao<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.context.get_tx_context().block_prev_randao)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn gas_limit<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack
        .push(i.context.get_tx_context().block_gas_limit as u64)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn chain_id<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.context.get_tx_context().chain_id)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn self_balance<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_ISTANBUL, i.revision)?;
    i.gas_left.consume(5)?;
    let addr = i.message.recipient();
    if u256::from(addr) == u256::ZERO {
        i.stack.push(u256::ZERO)?;
    } else {
        i.stack.push(i.context.get_balance(addr))?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn base_fee<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_LONDON, i.revision)?;
    i.gas_left.consume(2)?;
    i.stack.push(i.context.get_tx_context().block_base_fee)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn blob_hash<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
    i.gas_left.consume(3)?;
    let [idx] = i.stack.pop()?;
    let (idx, idx_overflow) = idx.into_u64_with_overflow();
    let idx = idx as usize;
    let hashes = ExecutionTxContext::from(i.context.get_tx_context()).blob_hashes;
    if !idx_overflow && idx < hashes.len() {
        i.stack.push(hashes[idx])?;
    } else {
        i.stack.push(u256::ZERO)?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn blob_base_fee<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
    i.gas_left.consume(2)?;
    i.stack.push(i.context.get_tx_context().blob_base_fee)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn pop<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    let [_] = i.stack.pop()?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn m_load<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [offset] = i.stack.pop()?;

    i.stack.push(i.memory.get_word(offset, &mut i.gas_left)?)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn m_store<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value, offset] = i.stack.pop()?;

    let dest = i.memory.get_mut_slice(offset, 32, &mut i.gas_left)?;
    dest.copy_from_slice(value.as_slice());
    i.code_reader.next();
    i.return_from_op()
}

pub fn m_store8<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(3)?;
    let [value, offset] = i.stack.pop()?;

    let dest = i.memory.get_mut_byte(offset, &mut i.gas_left)?;
    *dest = value[31];
    i.code_reader.next();
    i.return_from_op()
}

pub fn s_load<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    if i.revision < Revision::EVMC_BERLIN {
        i.gas_left.consume(800)?;
    }
    let [key] = i.stack.pop()?;
    let key = key.into();
    let addr = i.message.recipient();
    if i.revision >= Revision::EVMC_BERLIN {
        if i.context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD {
            i.gas_left.consume(2_100)?;
        } else {
            i.gas_left.consume(100)?;
        }
    }
    let value = i.context.get_storage(addr, &key);
    i.stack.push(value)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn jump<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(8)?;
    let [dest] = i.stack.pop()?;
    i.code_reader.try_jump(dest)?;
    i.return_from_op()
}

pub fn jump_i<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(10)?;
    let [cond, dest] = i.stack.pop()?;
    if cond == u256::ZERO {
        i.code_reader.next();
    } else {
        i.code_reader.try_jump(dest)?;
    }
    i.return_from_op()
}

pub fn pc<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.code_reader.pc())?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn m_size<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.memory.len())?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn gas<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(2)?;
    i.stack.push(i.gas_left.as_u64())?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn jump_dest<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    i.gas_left.consume(1)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn t_load<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
    i.gas_left.consume(100)?;
    let [key] = i.stack.pop()?;
    let addr = i.message.recipient();
    let value = i.context.get_transient_storage(addr, &key.into());
    i.stack.push(value)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn t_store<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
    check_not_read_only(i)?;
    i.gas_left.consume(100)?;
    let [value, key] = i.stack.pop()?;
    let addr = i.message.recipient();
    i.context
        .set_transient_storage(addr, &key.into(), &value.into());
    i.code_reader.next();
    i.return_from_op()
}

pub fn m_copy<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
    i.gas_left.consume(3)?;
    let [len, offset, dest_offset] = i.stack.pop()?;
    if len != u256::ZERO {
        i.memory
            .copy_within(offset, dest_offset, len, &mut i.gas_left)?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn return_<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    let [len, offset] = i.stack.pop()?;
    let len = len.try_into().map_err(|_| FailStatus::OutOfGas)?;
    let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
    #[cfg(not(feature = "custom-evmc"))]
    {
        i.output = Some((data).to_owned());
    }
    #[cfg(feature = "custom-evmc")]
    {
        i.output = Some(Box::from(&*data));
    }
    i.exec_status = ExecStatus::Returned;
    Ok(())
}

pub fn revert<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    let [len, offset] = i.stack.pop()?;
    let len = len.try_into().map_err(|_| FailStatus::OutOfGas)?;
    let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
    // TODO revert i changes
    // gas_refund = original_gas_refund;
    #[cfg(not(feature = "custom-evmc"))]
    {
        i.output = Some((data).to_owned());
    }
    #[cfg(feature = "custom-evmc")]
    {
        i.output = Some(Box::from(&*data));
    }
    i.exec_status = ExecStatus::Revert;
    Ok(())
}

pub fn invalid<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_min_revision(Revision::EVMC_HOMESTEAD, i.revision)?;
    Err(FailStatus::InvalidInstruction)
}

pub fn self_destruct<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_not_read_only(i)?;
    i.gas_left.consume(5_000)?;
    let [addr] = i.stack.pop()?;
    let addr = addr.into();

    if i.revision >= Revision::EVMC_BERLIN
        && i.context.access_account(&addr) == AccessStatus::EVMC_ACCESS_COLD
    {
        i.gas_left.consume(2_600)?;
    }

    if u256::from(i.context.get_balance(i.message.recipient())) > u256::ZERO
        && !i.context.account_exists(&addr)
    {
        i.gas_left.consume(25_000)?;
    }

    let destructed = i.context.selfdestruct(i.message.recipient(), &addr);
    if i.revision <= Revision::EVMC_BERLIN && destructed {
        i.gas_refund += 24_000;
    }

    i.exec_status = ExecStatus::Stopped;
    Ok(())
}

pub fn sstore<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_not_read_only(i)?;

    if i.revision >= Revision::EVMC_ISTANBUL && i.gas_left <= 2_300 {
        return Err(FailStatus::OutOfGas);
    }
    let [value, key] = i.stack.pop()?;
    let key = key.into();
    let addr = i.message.recipient();

    let (dyn_gas_1, dyn_gas_2, dyn_gas_3, refund_1, refund_2, refund_3) =
        if i.revision >= Revision::EVMC_LONDON {
            (100, 2_900, 20_000, 5_000 - 2_100 - 100, 4_800, 20_000 - 100)
        } else if i.revision >= Revision::EVMC_BERLIN {
            (
                100,
                2_900,
                20_000,
                5_000 - 2_100 - 100,
                15_000,
                20_000 - 100,
            )
        } else if i.revision >= Revision::EVMC_ISTANBUL {
            (800, 5_000, 20_000, 4_200, 15_000, 19_200)
        } else {
            (5_000, 5_000, 20_000, 0, 0, 0)
        };

    let status = i.context.set_storage(addr, &key, &value.into());
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
    if i.revision >= Revision::EVMC_BERLIN
        && i.context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD
    {
        dyn_gas += 2_100;
    }
    i.gas_left.consume(dyn_gas)?;
    i.gas_refund += gas_refund_change;
    i.code_reader.next();
    i.return_from_op()
}

pub fn push<E: ExecutionContextTrait>(i: &mut Interpreter<E>, len: usize) -> OpResult {
    i.gas_left.consume(3)?;
    i.code_reader.next();
    i.stack.push(i.code_reader.get_push_data(len))?;
    i.return_from_op()
}

pub fn dup<E: ExecutionContextTrait>(i: &mut Interpreter<E>, nth: usize) -> OpResult {
    i.gas_left.consume(3)?;
    i.stack.push(i.stack.nth(nth - 1)?)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn swap<E: ExecutionContextTrait>(i: &mut Interpreter<E>, nth: usize) -> OpResult {
    i.gas_left.consume(3)?;
    i.stack.swap_with_top(nth)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn log<const N: usize, E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    check_not_read_only(i)?;
    i.gas_left.consume(375)?;
    let [len, offset] = i.stack.pop()?;
    let mut topics: [u256; N] = i.stack.pop()?;
    let (len, len_overflow) = len.into_u64_with_overflow();
    let (len8, len8_overflow) = len.overflowing_mul(8);
    let (cost, cost_overflow) = (375 * N as u64).overflowing_add(len8);
    if len_overflow || len8_overflow || cost_overflow {
        return Err(FailStatus::OutOfGas);
    }
    i.gas_left.consume(cost)?;

    let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
    topics.reverse();
    // SAFETY:
    // [u256] is a newtype of [Uint256] with repr(transparent) which guarantees the same memory
    // layout.
    let topics = unsafe { mem::transmute::<&[u256], &[Uint256]>(topics.as_slice()) };
    i.context.emit_log(i.message.recipient(), data, topics);
    i.code_reader.next();
    i.return_from_op()
}

pub fn create<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    create_or_create2::<false, _>(i)
}

pub fn create2<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    create_or_create2::<true, _>(i)
}

pub fn create_or_create2<const CREATE2: bool, E: ExecutionContextTrait>(
    i: &mut Interpreter<E>,
) -> OpResult {
    i.gas_left.consume(32_000)?;
    check_not_read_only(i)?;
    let [len, offset, value] = i.stack.pop()?;
    let salt = if CREATE2 {
        let [salt] = i.stack.pop()?;
        salt
    } else {
        u256::ZERO // ignored
    };
    let len = len.try_into().map_err(|_| FailStatus::OutOfGas)?;

    let init_code_word_size = word_size(len)?;
    if i.revision >= Revision::EVMC_SHANGHAI {
        const MAX_INIT_CODE_LEN: u64 = 2 * 24576;
        if len > MAX_INIT_CODE_LEN {
            return Err(FailStatus::OutOfGas);
        }
        let init_code_cost = 2 * init_code_word_size; // does not overflow
        i.gas_left.consume(init_code_cost)?;
    }
    if CREATE2 {
        let hash_cost = 6 * init_code_word_size; // does not overflow
        i.gas_left.consume(hash_cost)?;
    }

    let init_code = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;

    if value > i.context.get_balance(i.message.recipient()).into() {
        i.last_call_return_data = None;
        i.stack.push(u256::ZERO)?;
        i.code_reader.next();
        return i.return_from_op();
    }

    let gas_left = i.gas_left.as_u64();
    let gas_limit = gas_left - gas_left / 64;
    i.gas_left.consume(gas_limit)?;

    let message = ExecutionMessage::new(
        if CREATE2 {
            MessageKind::EVMC_CREATE2
        } else {
            MessageKind::EVMC_CREATE
        },
        i.message.flags(),
        i.message.depth() + 1,
        gas_limit as i64,
        u256::ZERO.into(), // ignored
        *i.message.recipient(),
        Some(init_code),
        value.into(),
        salt.into(),
        u256::ZERO.into(), // ignored
        None,
        None,
    );
    let result = i.context.call(&message);

    i.gas_left.add(result.gas_left() as u64);
    i.gas_refund += result.gas_refund();

    if result.status_code() == EvmcStatusCode::EVMC_SUCCESS {
        let Some(addr) = result.create_address() else {
            return Err(FailStatus::InternalError);
        };

        i.last_call_return_data = None;
        i.stack.push(addr)?;
    } else {
        i.last_call_return_data = result.output().map(ToOwned::to_owned);
        i.stack.push(u256::ZERO)?;
    }
    i.code_reader.next();
    i.return_from_op()
}

pub fn call<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    call_or_call_code::<false, _>(i)
}

pub fn call_code<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    call_or_call_code::<true, _>(i)
}

pub fn call_or_call_code<const CODE: bool, E: ExecutionContextTrait>(
    i: &mut Interpreter<E>,
) -> OpResult {
    if i.revision < Revision::EVMC_BERLIN {
        i.gas_left.consume(700)?;
    }
    let [ret_len, ret_offset, args_len, args_offset, value, addr, gas] = i.stack.pop()?;

    if !CODE && value != u256::ZERO {
        check_not_read_only(i)?;
    }

    let addr = addr.into();
    let args_len = args_len.try_into().map_err(|_| FailStatus::OutOfGas)?;
    let ret_len = ret_len.try_into().map_err(|_| FailStatus::OutOfGas)?;

    i.gas_left
        .consume_address_access_cost(&addr, i.revision, i.context)?;
    i.gas_left.consume_positive_value_cost(&value)?;
    if !CODE {
        i.gas_left
            .consume_value_to_empty_account_cost(&value, &addr, i.context)?;
    }
    // access slice to consume potential memory expansion cost but drop it so that we can get
    // another mutable reference into memory for input
    let _dest = i
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut i.gas_left)?;
    let input = i
        .memory
        .get_mut_slice(args_offset, args_len, &mut i.gas_left)?;

    let gas_left = i.gas_left.as_u64();
    let limit = gas_left - gas_left / 64;
    let mut endowment = gas.into_u64_saturating();
    if i.revision >= Revision::EVMC_TANGERINE_WHISTLE {
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
    }
    let stipend = if value == u256::ZERO { 0 } else { 2_300 };
    i.gas_left.add(stipend);

    if value > u256::from(i.context.get_balance(i.message.recipient())) {
        i.last_call_return_data = None;
        i.stack.push(u256::ZERO)?;
        i.code_reader.next();
        return i.return_from_op();
    }

    let call_message = if CODE {
        ExecutionMessage::new(
            MessageKind::EVMC_CALLCODE,
            i.message.flags(),
            i.message.depth() + 1,
            (endowment + stipend) as i64,
            *i.message.recipient(),
            *i.message.recipient(),
            Some(input),
            value.into(),
            u256::ZERO.into(), // ignored
            addr,
            None,
            None,
        )
    } else {
        ExecutionMessage::new(
            MessageKind::EVMC_CALL,
            i.message.flags(),
            i.message.depth() + 1,
            (endowment + stipend) as i64,
            addr,
            *i.message.recipient(),
            Some(input),
            value.into(),
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            None,
            None,
        )
    };

    let result = i.context.call(&call_message);
    i.last_call_return_data = result.output().map(ToOwned::to_owned);
    let dest = i
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut i.gas_left)?;
    if let Some(output) = &i.last_call_return_data {
        let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
        dest[..min_len].copy_from_slice(&output[..min_len]);
    }

    i.gas_left.add(result.gas_left() as u64);
    i.gas_left.consume(endowment)?;
    i.gas_left.consume(stipend)?;
    i.gas_refund += result.gas_refund();

    i.stack
        .push(result.status_code() == EvmcStatusCode::EVMC_SUCCESS)?;
    i.code_reader.next();
    i.return_from_op()
}

pub fn static_call<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    static_or_delegate_call::<false, _>(i)
}

pub fn delegate_call<E: ExecutionContextTrait>(i: &mut Interpreter<E>) -> OpResult {
    static_or_delegate_call::<true, _>(i)
}

pub fn static_or_delegate_call<const DELEGATE: bool, E: ExecutionContextTrait>(
    i: &mut Interpreter<E>,
) -> OpResult {
    if i.revision < Revision::EVMC_BERLIN {
        i.gas_left.consume(700)?;
    }
    let [ret_len, ret_offset, args_len, args_offset, addr, gas] = i.stack.pop()?;

    let addr = addr.into();
    let args_len = args_len.try_into().map_err(|_| FailStatus::OutOfGas)?;
    let ret_len = ret_len.try_into().map_err(|_| FailStatus::OutOfGas)?;

    i.gas_left
        .consume_address_access_cost(&addr, i.revision, i.context)?;
    // access slice to consume potential memory expansion cost but drop it so that we can get
    // another mutable reference into memory for input
    let _dest = i
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut i.gas_left)?;
    let input = i
        .memory
        .get_mut_slice(args_offset, args_len, &mut i.gas_left)?;

    let gas_left = i.gas_left.as_u64();
    let limit = gas_left - gas_left / 64;
    let mut endowment = gas.into_u64_saturating();
    if i.revision >= Revision::EVMC_TANGERINE_WHISTLE {
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
    }

    let call_message = if DELEGATE {
        ExecutionMessage::new(
            MessageKind::EVMC_DELEGATECALL,
            i.message.flags(),
            i.message.depth() + 1,
            endowment as i64,
            *i.message.recipient(),
            *i.message.sender(),
            Some(input),
            *i.message.value(),
            u256::ZERO.into(), // ignored
            addr,
            None,
            None,
        )
    } else {
        ExecutionMessage::new(
            MessageKind::EVMC_CALL,
            MessageFlags::EVMC_STATIC as u32,
            i.message.depth() + 1,
            endowment as i64,
            addr,
            *i.message.recipient(),
            Some(input),
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            u256::ZERO.into(), // ignored
            None,
            None,
        )
    };

    let result = i.context.call(&call_message);
    i.last_call_return_data = result.output().map(ToOwned::to_owned);
    let dest = i
        .memory
        .get_mut_slice(ret_offset, ret_len, &mut i.gas_left)?;
    if let Some(output) = &i.last_call_return_data {
        let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
        dest[..min_len].copy_from_slice(&output[..min_len]);
    }

    i.gas_left.add(result.gas_left() as u64);
    i.gas_left.consume(endowment)?;
    i.gas_refund += result.gas_refund();

    i.stack
        .push(result.status_code() == EvmcStatusCode::EVMC_SUCCESS)?;
    i.code_reader.next();
    i.return_from_op()
}

use std::{cmp::min, mem};

use evmc_vm::{
    AccessStatus, ExecutionContext, ExecutionMessage, ExecutionResult, MessageFlags, MessageKind,
    Revision, StatusCode, StepResult, StepStatusCode, StorageStatus, Uint256,
};
use sha3::{Digest, Keccak256};

use crate::{
    types::{u256, CodeReader, ExecutionTxContext, GetOpcodeError, Memory, Opcode, Stack},
    utils::{
        check_min_revision, check_not_read_only, consume_address_access_cost, consume_copy_cost,
        consume_gas, consume_positive_value_cost, consume_value_to_empty_account_cost, word_size,
        SliceExt,
    },
};

pub struct Interpreter<'a> {
    pub step_status_code: StepStatusCode,
    pub status_code: StatusCode,
    pub message: &'a ExecutionMessage,
    pub context: &'a mut ExecutionContext<'a>,
    pub revision: Revision,
    pub code_reader: CodeReader<'a>,
    pub gas_left: u64,
    pub gas_refund: i64,
    pub output: Option<Vec<u8>>,
    pub stack: Stack,
    pub memory: Memory,
    pub last_call_return_data: Option<Vec<u8>>,
    pub steps: Option<i32>,
}

impl<'a> Interpreter<'a> {
    pub fn new(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut ExecutionContext<'a>,
        code: &'a [u8],
    ) -> Self {
        Self {
            step_status_code: StepStatusCode::EVMC_STEP_RUNNING,
            status_code: StatusCode::EVMC_SUCCESS,
            message,
            context,
            revision,
            code_reader: CodeReader::new(code, 0),
            gas_left: message.gas() as u64,
            gas_refund: 0,
            output: None,
            stack: Stack::new(Vec::new()),
            memory: Memory::new(Vec::new()),
            last_call_return_data: None,
            steps: None,
        }
    }

    #[allow(clippy::too_many_arguments)]
    pub fn new_steppable(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut ExecutionContext<'a>,
        step_status_code: StepStatusCode,
        code: &'a [u8],
        pc: usize,
        gas_refund: i64,
        stack: Stack,
        memory: Memory,
        last_call_return_data: Option<Vec<u8>>,
        steps: Option<i32>,
    ) -> Self {
        Self {
            step_status_code,
            status_code: StatusCode::EVMC_SUCCESS,
            message,
            context,
            revision,
            code_reader: CodeReader::new(code, pc),
            gas_left: message.gas() as u64,
            gas_refund,
            output: None,
            stack,
            memory,
            last_call_return_data,
            steps,
        }
    }

    pub fn run(mut self) -> Result<Self, StatusCode> {
        loop {
            match &mut self.steps {
                None => (),
                Some(0) => break,
                Some(steps) => *steps -= 1,
            }
            let op = match self.code_reader.get() {
                Ok(op) => op,
                Err(GetOpcodeError::OutOfRange) => {
                    self.step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                    break;
                }
                Err(GetOpcodeError::Invalid) => {
                    return Err(StatusCode::EVMC_INVALID_INSTRUCTION);
                }
            };
            match op {
                Opcode::Stop => {
                    self.step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                    self.status_code = StatusCode::EVMC_SUCCESS;
                    break;
                }
                Opcode::Add => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [value1, value2] = self.stack.pop()?;
                    self.stack.push(value1 + value2)?;
                }
                Opcode::Mul => {
                    consume_gas(&mut self.gas_left, 5)?;
                    let [fac1, fac2] = self.stack.pop()?;
                    self.stack.push(fac1 * fac2)?;
                }
                Opcode::Sub => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [value1, value2] = self.stack.pop()?;
                    self.stack.push(value1 - value2)?;
                }
                Opcode::Div => {
                    consume_gas(&mut self.gas_left, 5)?;
                    let [value, denominator] = self.stack.pop()?;
                    self.stack.push(value / denominator)?;
                }
                Opcode::SDiv => {
                    consume_gas(&mut self.gas_left, 5)?;
                    let [value, denominator] = self.stack.pop()?;
                    self.stack.push(value.sdiv(denominator))?;
                }
                Opcode::Mod => {
                    consume_gas(&mut self.gas_left, 5)?;
                    let [value, denominator] = self.stack.pop()?;
                    self.stack.push(value % denominator)?;
                }
                Opcode::SMod => {
                    consume_gas(&mut self.gas_left, 5)?;
                    let [value, denominator] = self.stack.pop()?;
                    self.stack.push(value.srem(denominator))?;
                }
                Opcode::AddMod => {
                    consume_gas(&mut self.gas_left, 8)?;
                    let [value1, value2, denominator] = self.stack.pop()?;
                    self.stack.push(u256::addmod(value1, value2, denominator))?;
                }
                Opcode::MulMod => {
                    consume_gas(&mut self.gas_left, 8)?;
                    let [fac1, fac2, denominator] = self.stack.pop()?;
                    self.stack.push(u256::mulmod(fac1, fac2, denominator))?;
                }
                Opcode::Exp => {
                    consume_gas(&mut self.gas_left, 10)?;
                    let [value, exp] = self.stack.pop()?;
                    let byte_size =
                        32 - exp.into_iter().take_while(|byte| *byte == 0).count() as u64;
                    consume_gas(&mut self.gas_left, byte_size * 50)?; // * does not overflow
                    self.stack.push(value.pow(exp))?;
                }
                Opcode::SignExtend => {
                    consume_gas(&mut self.gas_left, 5)?;
                    let [size, value] = self.stack.pop()?;
                    self.stack.push(u256::signextend(size, value))?;
                }
                Opcode::Lt => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs < rhs)?;
                }
                Opcode::Gt => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs > rhs)?;
                }
                Opcode::SLt => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs.slt(&rhs))?;
                }
                Opcode::SGt => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs.sgt(&rhs))?;
                }
                Opcode::Eq => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs == rhs)?;
                }
                Opcode::IsZero => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [value] = self.stack.pop()?;
                    self.stack.push(value == u256::ZERO)?;
                }
                Opcode::And => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs & rhs)?;
                }
                Opcode::Or => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs | rhs)?;
                }
                Opcode::Xor => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [lhs, rhs] = self.stack.pop()?;
                    self.stack.push(lhs ^ rhs)?;
                }
                Opcode::Not => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [value] = self.stack.pop()?;
                    self.stack.push(!value)?;
                }
                Opcode::Byte => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [offset, value] = self.stack.pop()?;
                    self.stack.push(value.byte(offset))?;
                }
                Opcode::Shl => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [shift, value] = self.stack.pop()?;
                    self.stack.push(value << shift)?;
                }
                Opcode::Shr => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [shift, value] = self.stack.pop()?;
                    self.stack.push(value >> shift)?;
                }
                Opcode::Sar => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [shift, value] = self.stack.pop()?;
                    self.stack.push(value.sar(shift))?;
                }
                Opcode::Sha3 => {
                    consume_gas(&mut self.gas_left, 30)?;
                    let [offset, len] = self.stack.pop()?;

                    let len = len.try_into().map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;
                    consume_gas(&mut self.gas_left, 6 * word_size(len)?)?; // * does not overflow

                    let data = self.memory.get_mut_slice(offset, len, &mut self.gas_left)?;
                    let mut hasher = Keccak256::new();
                    hasher.update(data);
                    let mut bytes = [0; 32];
                    hasher.finalize_into((&mut bytes).into());
                    self.stack.push(bytes)?;
                }
                Opcode::Address => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.message.recipient())?;
                }
                Opcode::Balance => {
                    if self.revision < Revision::EVMC_BERLIN {
                        consume_gas(&mut self.gas_left, 700)?;
                    }
                    let [addr] = self.stack.pop()?;
                    let addr = addr.into();
                    consume_address_access_cost(&addr, &mut self)?;
                    self.stack.push(self.context.get_balance(&addr))?;
                }
                Opcode::Origin => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.context.get_tx_context().tx_origin)?;
                }
                Opcode::Caller => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.message.sender())?;
                }
                Opcode::CallValue => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(*self.message.value())?;
                }
                Opcode::CallDataLoad => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [offset] = self.stack.pop()?;
                    let (offset, overflow) = offset.into_u64_with_overflow();
                    let offset = offset as usize;
                    let call_data = self.message.input().map(Vec::as_slice).unwrap_or_default();
                    if overflow || offset >= call_data.len() {
                        self.stack.push(u256::ZERO)?;
                    } else {
                        let end = min(call_data.len(), offset + 32);
                        let mut bytes = [0; 32];
                        bytes[..end - offset].copy_from_slice(&call_data[offset..end]);
                        self.stack.push(bytes)?;
                    }
                }
                Opcode::CallDataSize => {
                    consume_gas(&mut self.gas_left, 2)?;
                    let call_data_len = self.message.input().map(Vec::len).unwrap_or_default();
                    self.stack.push(call_data_len)?;
                }
                Opcode::Push0 => {
                    check_min_revision(Revision::EVMC_SHANGHAI, self.revision)?;
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(u256::ZERO)?;
                }
                Opcode::CallDataCopy => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [dest_offset, offset, len] = self.stack.pop()?;

                    if len != u256::ZERO {
                        let len = len
                            .try_into()
                            .map_err(|_| StatusCode::EVMC_INVALID_MEMORY_ACCESS)?;

                        let src = self
                            .message
                            .input()
                            .map(Vec::as_slice)
                            .unwrap_or_default()
                            .get_within_bounds(offset, len);
                        let dest =
                            self.memory
                                .get_mut_slice(dest_offset, len, &mut self.gas_left)?;
                        dest.copy_padded(src, &mut self.gas_left)?;
                    }
                }
                Opcode::CodeSize => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.code_reader.len())?;
                }
                Opcode::CodeCopy => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [dest_offset, offset, len] = self.stack.pop()?;

                    if len != u256::ZERO {
                        let len = len.try_into().map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;

                        let src = self.code_reader.get_within_bounds(offset, len);
                        let dest =
                            self.memory
                                .get_mut_slice(dest_offset, len, &mut self.gas_left)?;
                        dest.copy_padded(src, &mut self.gas_left)?;
                    }
                }
                Opcode::GasPrice => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().tx_gas_price)?;
                }
                Opcode::ExtCodeSize => {
                    if self.revision < Revision::EVMC_BERLIN {
                        consume_gas(&mut self.gas_left, 700)?;
                    }
                    let [addr] = self.stack.pop()?;
                    let addr = addr.into();
                    consume_address_access_cost(&addr, &mut self)?;
                    self.stack.push(self.context.get_code_size(&addr))?;
                }
                Opcode::ExtCodeCopy => {
                    if self.revision < Revision::EVMC_BERLIN {
                        consume_gas(&mut self.gas_left, 700)?;
                    }
                    let [addr, dest_offset, offset, len] = self.stack.pop()?;
                    let addr = addr.into();

                    consume_address_access_cost(&addr, &mut self)?;
                    if len != u256::ZERO {
                        let len = len.try_into().map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;

                        let dest =
                            self.memory
                                .get_mut_slice(dest_offset, len, &mut self.gas_left)?;
                        let (offset, offset_overflow) = offset.into_u64_with_overflow();
                        consume_copy_cost(&mut self.gas_left, len)?;
                        let bytes_written = self.context.copy_code(&addr, offset as usize, dest);
                        if offset_overflow {
                            dest.set_to_zero();
                        } else if (bytes_written as u64) < len {
                            dest[bytes_written..].set_to_zero();
                        }
                    }
                }
                Opcode::ReturnDataSize => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(
                        self.last_call_return_data
                            .as_ref()
                            .map(Vec::len)
                            .unwrap_or_default(),
                    )?;
                }
                Opcode::ReturnDataCopy => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [dest_offset, offset, len] = self.stack.pop()?;

                    let src = self.last_call_return_data.as_deref().unwrap_or_default();
                    let (offset, offset_overflow) = offset.into_u64_with_overflow();
                    let (len, len_overflow) = len.into_u64_with_overflow();
                    let (end, end_overflow) = offset.overflowing_add(len);
                    if offset_overflow || len_overflow || end_overflow || end > src.len() as u64 {
                        return Err(StatusCode::EVMC_INVALID_MEMORY_ACCESS);
                    }

                    if len != 0 {
                        let src = &src[offset as usize..end as usize];
                        let dest =
                            self.memory
                                .get_mut_slice(dest_offset, len, &mut self.gas_left)?;
                        dest.copy_padded(src, &mut self.gas_left)?;
                    }
                }
                Opcode::ExtCodeHash => {
                    if self.revision < Revision::EVMC_BERLIN {
                        consume_gas(&mut self.gas_left, 700)?;
                    }
                    let [addr] = self.stack.pop()?;
                    let addr = addr.into();
                    consume_address_access_cost(&addr, &mut self)?;
                    self.stack.push(self.context.get_code_hash(&addr))?;
                }
                Opcode::BlockHash => {
                    consume_gas(&mut self.gas_left, 20)?;
                    let [block_number] = self.stack.pop()?;
                    self.stack.push(
                        block_number
                            .try_into()
                            .map(|idx: u64| self.context.get_block_hash(idx as i64))
                            .unwrap_or(u256::ZERO.into()),
                    )?;
                }
                Opcode::Coinbase => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().block_coinbase)?;
                }
                Opcode::Timestamp => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().block_timestamp as u64)?;
                }
                Opcode::Number => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().block_number as u64)?;
                }
                Opcode::PrevRandao => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().block_prev_randao)?;
                }
                Opcode::GasLimit => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().block_gas_limit as u64)?;
                }
                Opcode::ChainId => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.context.get_tx_context().chain_id)?;
                }
                Opcode::SelfBalance => {
                    check_min_revision(Revision::EVMC_ISTANBUL, self.revision)?;
                    consume_gas(&mut self.gas_left, 5)?;
                    let addr = self.message.recipient();
                    if u256::from(addr) == u256::ZERO {
                        self.stack.push(u256::ZERO)?;
                    } else {
                        self.stack.push(self.context.get_balance(addr))?;
                    }
                }
                Opcode::BaseFee => {
                    check_min_revision(Revision::EVMC_LONDON, self.revision)?;
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().block_base_fee)?;
                }
                Opcode::BlobHash => {
                    check_min_revision(Revision::EVMC_CANCUN, self.revision)?;
                    consume_gas(&mut self.gas_left, 3)?;
                    let [idx] = self.stack.pop()?;
                    let (idx, idx_overflow) = idx.into_u64_with_overflow();
                    let idx = idx as usize;
                    let hashes =
                        ExecutionTxContext::from(self.context.get_tx_context()).blob_hashes;
                    if !idx_overflow && idx < hashes.len() {
                        self.stack.push(hashes[idx])?;
                    } else {
                        self.stack.push(u256::ZERO)?;
                    }
                }
                Opcode::BlobBaseFee => {
                    check_min_revision(Revision::EVMC_CANCUN, self.revision)?;
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack
                        .push(self.context.get_tx_context().blob_base_fee)?;
                }
                Opcode::Pop => {
                    consume_gas(&mut self.gas_left, 2)?;
                    let [_] = self.stack.pop()?;
                }
                Opcode::MLoad => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [offset] = self.stack.pop()?;

                    self.stack
                        .push(self.memory.get_word(offset, &mut self.gas_left)?)?;
                }
                Opcode::MStore => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [offset, value] = self.stack.pop()?;

                    let dest = self.memory.get_mut_slice(offset, 32, &mut self.gas_left)?;
                    dest.copy_from_slice(value.as_slice());
                }
                Opcode::MStore8 => {
                    consume_gas(&mut self.gas_left, 3)?;
                    let [offset, value] = self.stack.pop()?;

                    let dest = self.memory.get_mut_byte(offset, &mut self.gas_left)?;
                    *dest = value[31];
                }
                Opcode::SLoad => {
                    if self.revision < Revision::EVMC_BERLIN {
                        consume_gas(&mut self.gas_left, 800)?;
                    }
                    let [key] = self.stack.pop()?;
                    let key = key.into();
                    let addr = self.message.recipient();
                    if self.revision >= Revision::EVMC_BERLIN {
                        if self.context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD
                        {
                            consume_gas(&mut self.gas_left, 2100)?;
                        } else {
                            consume_gas(&mut self.gas_left, 100)?;
                        }
                    }
                    let value = self.context.get_storage(addr, &key);
                    self.stack.push(value)?;
                }
                Opcode::SStore => self.sstore()?,
                Opcode::Jump => {
                    consume_gas(&mut self.gas_left, 8)?;
                    let [dest] = self.stack.pop()?;
                    self.code_reader.try_jump(dest)?;
                }
                Opcode::JumpI => {
                    consume_gas(&mut self.gas_left, 10)?;
                    let [dest, cond] = self.stack.pop()?;
                    if cond == u256::ZERO {
                        self.code_reader.next();
                    } else {
                        self.code_reader.try_jump(dest)?;
                    }
                }
                Opcode::Pc => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.code_reader.pc())?;
                }
                Opcode::MSize => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.memory.len())?;
                }
                Opcode::Gas => {
                    consume_gas(&mut self.gas_left, 2)?;
                    self.stack.push(self.gas_left)?;
                }
                Opcode::JumpDest => {
                    consume_gas(&mut self.gas_left, 1)?;
                }
                Opcode::TLoad => {
                    check_min_revision(Revision::EVMC_CANCUN, self.revision)?;
                    consume_gas(&mut self.gas_left, 100)?;
                    let [key] = self.stack.pop()?;
                    let addr = self.message.recipient();
                    let value = self.context.get_transient_storage(addr, &key.into());
                    self.stack.push(value)?;
                }
                Opcode::TStore => {
                    check_min_revision(Revision::EVMC_CANCUN, self.revision)?;
                    check_not_read_only(&self)?;
                    consume_gas(&mut self.gas_left, 100)?;
                    let [key, value] = self.stack.pop()?;
                    let addr = self.message.recipient();
                    self.context
                        .set_transient_storage(addr, &key.into(), &value.into());
                }
                Opcode::MCopy => {
                    check_min_revision(Revision::EVMC_CANCUN, self.revision)?;
                    consume_gas(&mut self.gas_left, 3)?;
                    let [dest_offset, offset, len] = self.stack.pop()?;
                    if len != u256::ZERO {
                        self.memory
                            .copy_within(offset, dest_offset, len, &mut self.gas_left)?;
                    }
                }
                Opcode::Push1 => self.push(1)?,
                Opcode::Push2 => self.push(2)?,
                Opcode::Push3 => self.push(3)?,
                Opcode::Push4 => self.push(4)?,
                Opcode::Push5 => self.push(5)?,
                Opcode::Push6 => self.push(6)?,
                Opcode::Push7 => self.push(7)?,
                Opcode::Push8 => self.push(8)?,
                Opcode::Push9 => self.push(9)?,
                Opcode::Push10 => self.push(10)?,
                Opcode::Push11 => self.push(11)?,
                Opcode::Push12 => self.push(12)?,
                Opcode::Push13 => self.push(13)?,
                Opcode::Push14 => self.push(14)?,
                Opcode::Push15 => self.push(15)?,
                Opcode::Push16 => self.push(16)?,
                Opcode::Push17 => self.push(17)?,
                Opcode::Push18 => self.push(18)?,
                Opcode::Push19 => self.push(19)?,
                Opcode::Push20 => self.push(20)?,
                Opcode::Push21 => self.push(21)?,
                Opcode::Push22 => self.push(22)?,
                Opcode::Push23 => self.push(23)?,
                Opcode::Push24 => self.push(24)?,
                Opcode::Push25 => self.push(25)?,
                Opcode::Push26 => self.push(26)?,
                Opcode::Push27 => self.push(27)?,
                Opcode::Push28 => self.push(28)?,
                Opcode::Push29 => self.push(29)?,
                Opcode::Push30 => self.push(30)?,
                Opcode::Push31 => self.push(31)?,
                Opcode::Push32 => self.push(32)?,
                Opcode::Dup1 => self.dup(1)?,
                Opcode::Dup2 => self.dup(2)?,
                Opcode::Dup3 => self.dup(3)?,
                Opcode::Dup4 => self.dup(4)?,
                Opcode::Dup5 => self.dup(5)?,
                Opcode::Dup6 => self.dup(6)?,
                Opcode::Dup7 => self.dup(7)?,
                Opcode::Dup8 => self.dup(8)?,
                Opcode::Dup9 => self.dup(9)?,
                Opcode::Dup10 => self.dup(10)?,
                Opcode::Dup11 => self.dup(11)?,
                Opcode::Dup12 => self.dup(12)?,
                Opcode::Dup13 => self.dup(13)?,
                Opcode::Dup14 => self.dup(14)?,
                Opcode::Dup15 => self.dup(15)?,
                Opcode::Dup16 => self.dup(16)?,
                Opcode::Swap1 => self.swap(1)?,
                Opcode::Swap2 => self.swap(2)?,
                Opcode::Swap3 => self.swap(3)?,
                Opcode::Swap4 => self.swap(4)?,
                Opcode::Swap5 => self.swap(5)?,
                Opcode::Swap6 => self.swap(6)?,
                Opcode::Swap7 => self.swap(7)?,
                Opcode::Swap8 => self.swap(8)?,
                Opcode::Swap9 => self.swap(9)?,
                Opcode::Swap10 => self.swap(10)?,
                Opcode::Swap11 => self.swap(11)?,
                Opcode::Swap12 => self.swap(12)?,
                Opcode::Swap13 => self.swap(13)?,
                Opcode::Swap14 => self.swap(14)?,
                Opcode::Swap15 => self.swap(15)?,
                Opcode::Swap16 => self.swap(16)?,
                Opcode::Log0 => self.log::<0>()?,
                Opcode::Log1 => self.log::<1>()?,
                Opcode::Log2 => self.log::<2>()?,
                Opcode::Log3 => self.log::<3>()?,
                Opcode::Log4 => self.log::<4>()?,
                Opcode::Create => self.create()?,
                Opcode::Call => self.call()?,
                Opcode::CallCode => self.call_code()?,
                Opcode::Return => {
                    let [offset, len] = self.stack.pop()?;
                    let len = len.try_into().map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;
                    let data = self.memory.get_mut_slice(offset, len, &mut self.gas_left)?;
                    self.output = Some(data.to_owned());
                    self.step_status_code = StepStatusCode::EVMC_STEP_RETURNED;
                    break;
                }
                Opcode::DelegateCall => self.delegate_call()?,
                Opcode::Create2 => self.create2()?,
                Opcode::StaticCall => self.static_call()?,
                Opcode::Revert => {
                    let [offset, len] = self.stack.pop()?;
                    let len = len.try_into().map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;
                    let data = self.memory.get_mut_slice(offset, len, &mut self.gas_left)?;
                    // TODO revert self changes
                    // gas_refund = original_gas_refund;
                    self.output = Some(data.to_owned());
                    self.step_status_code = StepStatusCode::EVMC_STEP_REVERTED;
                    self.status_code = StatusCode::EVMC_REVERT;
                    break;
                }
                Opcode::Invalid => {
                    check_min_revision(Revision::EVMC_HOMESTEAD, self.revision)?;
                    return Err(StatusCode::EVMC_INVALID_INSTRUCTION);
                }
                Opcode::SelfDestruct => {
                    check_not_read_only(&self)?;
                    consume_gas(&mut self.gas_left, 5000)?;
                    let [addr] = self.stack.pop()?;
                    let addr = addr.into();

                    if self.revision >= Revision::EVMC_BERLIN
                        && self.context.access_account(&addr) == AccessStatus::EVMC_ACCESS_COLD
                    {
                        consume_gas(&mut self.gas_left, 2600)?;
                    }

                    if u256::from(self.context.get_balance(self.message.recipient())) > u256::ZERO
                        && !self.context.account_exists(&addr)
                    {
                        consume_gas(&mut self.gas_left, 25000)?;
                    }

                    let destructed = self.context.selfdestruct(self.message.recipient(), &addr);
                    if self.revision <= Revision::EVMC_BERLIN && destructed {
                        self.gas_refund += 24000;
                    }

                    self.step_status_code = StepStatusCode::EVMC_STEP_STOPPED;
                    break;
                }
            }

            if !(Opcode::Push1 as u8..=Opcode::Push32 as u8).contains(&(op as u8))
                && op != Opcode::Jump
                && op != Opcode::JumpI
            {
                self.code_reader.next();
            }
        }

        Ok(self)
    }

    fn sstore(&mut self) -> Result<(), StatusCode> {
        check_not_read_only(self)?;

        if self.revision >= Revision::EVMC_ISTANBUL && self.gas_left <= 2300 {
            return Err(StatusCode::EVMC_OUT_OF_GAS);
        }
        let [key, value] = self.stack.pop()?;
        let key = key.into();
        let addr = self.message.recipient();

        let (dyn_gas_1, dyn_gas_2, dyn_gas_3, refund_1, refund_2, refund_3) =
            if self.revision >= Revision::EVMC_LONDON {
                (100, 2900, 20000, 5000 - 2100 - 100, 4800, 20000 - 100)
            } else if self.revision >= Revision::EVMC_BERLIN {
                (100, 2900, 20000, 5000 - 2100 - 100, 15000, 20000 - 100)
            } else if self.revision >= Revision::EVMC_ISTANBUL {
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

        let status = self.context.set_storage(addr, &key, &value.into());
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
        if self.revision >= Revision::EVMC_BERLIN
            && self.context.access_storage(addr, &key) == AccessStatus::EVMC_ACCESS_COLD
        {
            dyn_gas += 2100;
        }
        consume_gas(&mut self.gas_left, dyn_gas)?;
        self.gas_refund += gas_refund_change;
        Ok(())
    }

    fn push(&mut self, len: usize) -> Result<(), StatusCode> {
        consume_gas(&mut self.gas_left, 3)?;
        self.code_reader.next();
        self.stack.push(self.code_reader.get_push_data(len))?;
        Ok(())
    }

    fn dup(&mut self, nth: usize) -> Result<(), StatusCode> {
        consume_gas(&mut self.gas_left, 3)?;
        self.stack.push(self.stack.nth(nth - 1)?)?;
        Ok(())
    }

    fn swap(&mut self, nth: usize) -> Result<(), StatusCode> {
        consume_gas(&mut self.gas_left, 3)?;
        self.stack.swap_with_top(nth)?;
        Ok(())
    }

    fn log<const N: usize>(&mut self) -> Result<(), StatusCode> {
        check_not_read_only(self)?;
        consume_gas(&mut self.gas_left, 375)?;
        let [offset, len] = self.stack.pop()?;
        let topics: [u256; N] = self.stack.pop()?;
        let (len, len_overflow) = len.into_u64_with_overflow();
        let (len8, len8_overflow) = len.overflowing_mul(8);
        let (cost, cost_overflow) = (375 * N as u64).overflowing_add(len8);
        if len_overflow || len8_overflow || cost_overflow {
            return Err(StatusCode::EVMC_OUT_OF_GAS);
        }
        consume_gas(&mut self.gas_left, cost)?;

        let data = self.memory.get_mut_slice(offset, len, &mut self.gas_left)?;
        // SAFETY:
        // [u256] is a newtype of [Uint256] with repr(transparent) which guarantees the same memory
        // layout.
        let topics = unsafe { mem::transmute::<&[u256], &[Uint256]>(topics.as_slice()) };
        self.context
            .emit_log(self.message.recipient(), data, topics);
        Ok(())
    }

    fn create(&mut self) -> Result<(), StatusCode> {
        self.create_or_create2::<false>()
    }

    fn create2(&mut self) -> Result<(), StatusCode> {
        self.create_or_create2::<true>()
    }

    fn create_or_create2<const CREATE2: bool>(&mut self) -> Result<(), StatusCode> {
        consume_gas(&mut self.gas_left, 32000)?;
        check_not_read_only(self)?;
        let [value, offset, len] = self.stack.pop()?;
        let salt = if CREATE2 {
            self.stack.pop::<1>()?[0]
        } else {
            u256::ZERO // ignored
        };
        let len = len.try_into().map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;

        let init_code_word_size = word_size(len)?;
        if self.revision >= Revision::EVMC_SHANGHAI {
            const MAX_INIT_CODE_LEN: u64 = 2 * 24576;
            if len > MAX_INIT_CODE_LEN {
                return Err(StatusCode::EVMC_OUT_OF_GAS);
            }
            let init_code_cost = 2 * init_code_word_size; // does not overflow
            consume_gas(&mut self.gas_left, init_code_cost)?;
        }
        if CREATE2 {
            let hash_cost = 6 * init_code_word_size; // does not overflow
            consume_gas(&mut self.gas_left, hash_cost)?;
        }

        let init_code = self.memory.get_mut_slice(offset, len, &mut self.gas_left)?;

        if value > self.context.get_balance(self.message.recipient()).into() {
            self.last_call_return_data = None;
            self.stack.push(u256::ZERO)?;
            return Ok(());
        }

        let gas_limit = self.gas_left - self.gas_left / 64;
        consume_gas(&mut self.gas_left, gas_limit)?;

        let message = ExecutionMessage::new(
            if CREATE2 {
                MessageKind::EVMC_CREATE2
            } else {
                MessageKind::EVMC_CREATE
            },
            self.message.flags(),
            self.message.depth() + 1,
            gas_limit as i64,
            u256::ZERO.into(), // ignored
            *self.message.recipient(),
            Some(init_code),
            value.into(),
            salt.into(),
            u256::ZERO.into(), // ignored
            None,
        );
        let result = self.context.call(&message);

        self.gas_left += result.gas_left() as u64;
        self.gas_refund += result.gas_refund();

        if result.status_code() == StatusCode::EVMC_SUCCESS {
            let Some(addr) = result.create_address() else {
                return Err(StatusCode::EVMC_INTERNAL_ERROR);
            };

            self.last_call_return_data = None;
            self.stack.push(addr)?;
        } else {
            self.last_call_return_data = result.output().map(ToOwned::to_owned);
            self.stack.push(u256::ZERO)?;
        }
        Ok(())
    }

    fn call(&mut self) -> Result<(), StatusCode> {
        self.call_or_call_code::<false>()
    }

    fn call_code(&mut self) -> Result<(), StatusCode> {
        self.call_or_call_code::<true>()
    }

    fn call_or_call_code<const CODE: bool>(&mut self) -> Result<(), StatusCode> {
        if self.revision < Revision::EVMC_BERLIN {
            consume_gas(&mut self.gas_left, 700)?;
        }
        let [gas, addr, value, args_offset, args_len, ret_offset, ret_len] = self.stack.pop()?;

        if !CODE && value != u256::ZERO {
            check_not_read_only(self)?;
        }

        let addr = addr.into();
        let args_len = args_len
            .try_into()
            .map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;
        let ret_len = ret_len
            .try_into()
            .map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;

        consume_address_access_cost(&addr, self)?;
        consume_positive_value_cost(&value, &mut self.gas_left)?;
        if !CODE {
            consume_value_to_empty_account_cost(&value, &addr, self)?;
        }
        // access slice to consume potential memory expansion cost but drop it so that we can get
        // another mutable reference into memory for input
        let _dest = self
            .memory
            .get_mut_slice(ret_offset, ret_len, &mut self.gas_left)?;
        let input = self
            .memory
            .get_mut_slice(args_offset, args_len, &mut self.gas_left)?;

        let limit = self.gas_left - self.gas_left / 64;
        let mut endowment = gas.into_u64_saturating();
        if self.revision >= Revision::EVMC_TANGERINE_WHISTLE {
            endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
        }
        let stipend = if value == u256::ZERO { 0 } else { 2300 };
        self.gas_left += stipend;

        if value > u256::from(self.context.get_balance(self.message.recipient())) {
            self.last_call_return_data = None;
            self.stack.push(u256::ZERO)?;
            return Ok(());
        }

        let call_message = if CODE {
            ExecutionMessage::new(
                MessageKind::EVMC_CALLCODE,
                self.message.flags(),
                self.message.depth() + 1,
                (endowment + stipend) as i64,
                *self.message.recipient(),
                *self.message.recipient(),
                Some(input),
                value.into(),
                u256::ZERO.into(), // ignored
                addr,
                None,
            )
        } else {
            ExecutionMessage::new(
                MessageKind::EVMC_CALL,
                self.message.flags(),
                self.message.depth() + 1,
                (endowment + stipend) as i64,
                addr,
                *self.message.recipient(),
                Some(input),
                value.into(),
                u256::ZERO.into(), // ignored
                u256::ZERO.into(), // ignored
                None,
            )
        };

        let result = self.context.call(&call_message);
        self.last_call_return_data = result.output().map(ToOwned::to_owned);
        let dest = self
            .memory
            .get_mut_slice(ret_offset, ret_len, &mut self.gas_left)?;
        if let Some(output) = &self.last_call_return_data {
            let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
            dest[..min_len].copy_from_slice(&output[..min_len]);
        }

        self.gas_left += result.gas_left() as u64;
        consume_gas(&mut self.gas_left, endowment)?;
        consume_gas(&mut self.gas_left, stipend)?;
        self.gas_refund += result.gas_refund();

        self.stack
            .push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
        Ok(())
    }

    fn static_call(&mut self) -> Result<(), StatusCode> {
        self.static_or_delegate_call::<false>()
    }

    fn delegate_call(&mut self) -> Result<(), StatusCode> {
        self.static_or_delegate_call::<true>()
    }

    fn static_or_delegate_call<const DELEGATE: bool>(&mut self) -> Result<(), StatusCode> {
        if self.revision < Revision::EVMC_BERLIN {
            consume_gas(&mut self.gas_left, 700)?;
        }
        let [gas, addr, args_offset, args_len, ret_offset, ret_len] = self.stack.pop()?;

        let addr = addr.into();
        let args_len = args_len
            .try_into()
            .map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;
        let ret_len = ret_len
            .try_into()
            .map_err(|_| StatusCode::EVMC_OUT_OF_GAS)?;

        consume_address_access_cost(&addr, self)?;
        // access slice to consume potential memory expansion cost but drop it so that we can get
        // another mutable reference into memory for input
        let _dest = self
            .memory
            .get_mut_slice(ret_offset, ret_len, &mut self.gas_left)?;
        let input = self
            .memory
            .get_mut_slice(args_offset, args_len, &mut self.gas_left)?;

        let limit = self.gas_left - self.gas_left / 64;
        let mut endowment = gas.into_u64_saturating();
        if self.revision >= Revision::EVMC_TANGERINE_WHISTLE {
            endowment = min(endowment, limit); // cap gas at all but one 64th of gas left
        }

        let call_message = if DELEGATE {
            ExecutionMessage::new(
                MessageKind::EVMC_DELEGATECALL,
                self.message.flags(),
                self.message.depth() + 1,
                endowment as i64,
                *self.message.recipient(),
                *self.message.sender(),
                Some(input),
                *self.message.value(),
                u256::ZERO.into(), // ignored
                addr,
                None,
            )
        } else {
            ExecutionMessage::new(
                MessageKind::EVMC_CALL,
                MessageFlags::EVMC_STATIC as u32,
                self.message.depth() + 1,
                endowment as i64,
                addr,
                *self.message.recipient(),
                Some(input),
                u256::ZERO.into(), // ignored
                u256::ZERO.into(), // ignored
                u256::ZERO.into(), // ignored
                None,
            )
        };

        let result = self.context.call(&call_message);
        self.last_call_return_data = result.output().map(ToOwned::to_owned);
        let dest = self
            .memory
            .get_mut_slice(ret_offset, ret_len, &mut self.gas_left)?;
        if let Some(output) = &self.last_call_return_data {
            let min_len = min(output.len(), ret_len as usize); // ret_len == dest.len()
            dest[..min_len].copy_from_slice(&output[..min_len]);
        }

        self.gas_left += result.gas_left() as u64;
        consume_gas(&mut self.gas_left, endowment)?;
        self.gas_refund += result.gas_refund();

        self.stack
            .push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
        Ok(())
    }
}

impl<'a> From<Interpreter<'a>> for StepResult {
    fn from(value: Interpreter) -> Self {
        let stack = value.stack.into_inner();
        // SAFETY:
        // u256 is a newtype of Uint256 with repr(transparent) which guarantees the same memory
        // layout.
        let stack = unsafe { mem::transmute::<Vec<u256>, Vec<Uint256>>(stack) };
        Self::new(
            value.step_status_code,
            value.status_code,
            value.revision,
            value.code_reader.pc() as u64,
            value.gas_left as i64,
            value.gas_refund,
            value.output,
            stack,
            value.memory.into_inner(),
            value.last_call_return_data,
        )
    }
}

impl<'a> From<Interpreter<'a>> for ExecutionResult {
    fn from(value: Interpreter) -> Self {
        Self::new(
            value.status_code,
            value.gas_left as i64,
            value.gas_refund,
            value.output.as_deref(),
        )
    }
}

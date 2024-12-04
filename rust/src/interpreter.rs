use std::cmp::min;

use evmc_vm::{
    AccessStatus, ExecutionMessage, ExecutionResult, MessageFlags, MessageKind, Revision,
    StatusCode, StepResult, StorageStatus, Uint256,
};

#[cfg(not(feature = "needs-fn-ptr-conversion"))]
use crate::types::Opcode;
#[cfg(feature = "needs-jumptable")]
use crate::utils::GetGenericStatic;
use crate::{
    types::{
        hash_cache, u256, CodeReader, ExecStatus, ExecutionContextTrait, ExecutionTxContext,
        FailStatus, GetOpcodeError, Memory, Observer, Stack,
    },
    utils::{check_min_revision, check_not_read_only, word_size, Gas, GasRefund, SliceExt},
};

type OpResult = Result<(), FailStatus>;

#[cfg(feature = "needs-jumptable")]
pub type OpFn<const STEPPABLE: bool> = fn(&mut Interpreter<STEPPABLE>) -> OpResult;

// The closures here are necessary because methods capture the lifetime of the type which we
// want to avoid.
#[cfg(feature = "needs-jumptable")]
const fn gen_jumptable<const STEPPABLE: bool>() -> [OpFn<STEPPABLE>; 256] {
    [
        Interpreter::stop,
        Interpreter::add,
        Interpreter::mul,
        Interpreter::sub,
        Interpreter::div,
        Interpreter::s_div,
        Interpreter::mod_,
        Interpreter::s_mod,
        Interpreter::add_mod,
        Interpreter::mul_mod,
        Interpreter::exp,
        Interpreter::sign_extend,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::lt,
        Interpreter::gt,
        Interpreter::s_lt,
        Interpreter::s_gt,
        Interpreter::eq,
        Interpreter::is_zero,
        Interpreter::and,
        Interpreter::or,
        Interpreter::xor,
        Interpreter::not,
        Interpreter::byte,
        Interpreter::shl,
        Interpreter::shr,
        Interpreter::sar,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::sha3,
        #[cfg(feature = "needs-fn-ptr-conversion")]
        Interpreter::no_op,
        #[cfg(feature = "needs-fn-ptr-conversion")]
        Interpreter::skip_no_ops,
        #[cfg(not(feature = "needs-fn-ptr-conversion"))]
        Interpreter::jumptable_placeholder,
        #[cfg(not(feature = "needs-fn-ptr-conversion"))]
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::address,
        Interpreter::balance,
        Interpreter::origin,
        Interpreter::caller,
        Interpreter::call_value,
        Interpreter::call_data_load,
        Interpreter::call_data_size,
        Interpreter::call_data_copy,
        Interpreter::code_size,
        Interpreter::code_copy,
        Interpreter::gas_price,
        Interpreter::ext_code_size,
        Interpreter::ext_code_copy,
        Interpreter::return_data_size,
        Interpreter::return_data_copy,
        Interpreter::ext_code_hash,
        Interpreter::block_hash,
        Interpreter::coinbase,
        Interpreter::timestamp,
        Interpreter::number,
        Interpreter::prev_randao,
        Interpreter::gas_limit,
        Interpreter::chain_id,
        Interpreter::self_balance,
        Interpreter::base_fee,
        Interpreter::blob_hash,
        Interpreter::blob_base_fee,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::pop,
        Interpreter::m_load,
        Interpreter::m_store,
        Interpreter::m_store8,
        Interpreter::s_load,
        Interpreter::sstore,
        Interpreter::jump,
        Interpreter::jump_i,
        Interpreter::pc,
        Interpreter::m_size,
        Interpreter::gas,
        Interpreter::jump_dest,
        Interpreter::t_load,
        Interpreter::t_store,
        Interpreter::m_copy,
        Interpreter::push0,
        Interpreter::push::<1>,
        Interpreter::push::<2>,
        Interpreter::push::<3>,
        Interpreter::push::<4>,
        Interpreter::push::<5>,
        Interpreter::push::<6>,
        Interpreter::push::<7>,
        Interpreter::push::<8>,
        Interpreter::push::<9>,
        Interpreter::push::<10>,
        Interpreter::push::<11>,
        Interpreter::push::<12>,
        Interpreter::push::<13>,
        Interpreter::push::<14>,
        Interpreter::push::<15>,
        Interpreter::push::<16>,
        Interpreter::push::<17>,
        Interpreter::push::<18>,
        Interpreter::push::<19>,
        Interpreter::push::<20>,
        Interpreter::push::<21>,
        Interpreter::push::<22>,
        Interpreter::push::<23>,
        Interpreter::push::<24>,
        Interpreter::push::<25>,
        Interpreter::push::<26>,
        Interpreter::push::<27>,
        Interpreter::push::<28>,
        Interpreter::push::<29>,
        Interpreter::push::<30>,
        Interpreter::push::<31>,
        Interpreter::push::<32>,
        Interpreter::dup::<1>,
        Interpreter::dup::<2>,
        Interpreter::dup::<3>,
        Interpreter::dup::<4>,
        Interpreter::dup::<5>,
        Interpreter::dup::<6>,
        Interpreter::dup::<7>,
        Interpreter::dup::<8>,
        Interpreter::dup::<9>,
        Interpreter::dup::<10>,
        Interpreter::dup::<11>,
        Interpreter::dup::<12>,
        Interpreter::dup::<13>,
        Interpreter::dup::<14>,
        Interpreter::dup::<15>,
        Interpreter::dup::<16>,
        Interpreter::swap::<1>,
        Interpreter::swap::<2>,
        Interpreter::swap::<3>,
        Interpreter::swap::<4>,
        Interpreter::swap::<5>,
        Interpreter::swap::<6>,
        Interpreter::swap::<7>,
        Interpreter::swap::<8>,
        Interpreter::swap::<9>,
        Interpreter::swap::<10>,
        Interpreter::swap::<11>,
        Interpreter::swap::<12>,
        Interpreter::swap::<13>,
        Interpreter::swap::<14>,
        Interpreter::swap::<15>,
        Interpreter::swap::<16>,
        Interpreter::log::<0>,
        Interpreter::log::<1>,
        Interpreter::log::<2>,
        Interpreter::log::<3>,
        Interpreter::log::<4>,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::create,
        Interpreter::call,
        Interpreter::call_code,
        Interpreter::return_,
        Interpreter::delegate_call,
        Interpreter::create2,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::static_call,
        Interpreter::jumptable_placeholder,
        Interpreter::jumptable_placeholder,
        Interpreter::revert,
        Interpreter::invalid,
        Interpreter::self_destruct,
    ]
}

#[cfg(feature = "needs-jumptable")]
pub struct GenericJumptable;

#[cfg(feature = "needs-jumptable")]
impl GetGenericStatic for GenericJumptable {
    type I<const STEPPABLE: bool> = [OpFn<STEPPABLE>; 256];

    fn get<const STEPPABLE: bool>() -> &'static Self::I<STEPPABLE> {
        static JUMPTABLE_STEPPABLE: [OpFn<true>; 256] = gen_jumptable();
        static JUMPTABLE_NON_STEPPABLE: [OpFn<false>; 256] = gen_jumptable();
        Self::get_with_args(&JUMPTABLE_STEPPABLE, &JUMPTABLE_NON_STEPPABLE)
    }
}

pub struct Interpreter<'a, const STEPPABLE: bool> {
    pub exec_status: ExecStatus,
    #[cfg(not(feature = "custom-evmc"))]
    pub message: &'a ExecutionMessage,
    #[cfg(feature = "custom-evmc")]
    pub message: &'a ExecutionMessage<'a>,
    pub context: &'a mut dyn ExecutionContextTrait,
    pub revision: Revision,
    pub code_reader: CodeReader<'a, STEPPABLE>,
    pub gas_left: Gas,
    pub gas_refund: GasRefund,
    #[cfg(not(feature = "custom-evmc"))]
    pub output: Option<Vec<u8>>,
    #[cfg(feature = "custom-evmc")]
    pub output: Option<Box<[u8]>>,
    pub stack: Stack,
    pub memory: Memory,
    pub last_call_return_data: Option<Vec<u8>>,
    pub steps: Option<i32>,
}

impl<'a> Interpreter<'a, false> {
    pub fn new(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut dyn ExecutionContextTrait,
        code: &'a [u8],
    ) -> Self {
        Self {
            exec_status: ExecStatus::Running,
            message,
            context,
            revision,
            code_reader: CodeReader::new(code, message.code_hash().map(|h| u256::from(*h)), 0),
            gas_left: Gas::new(message.gas()),
            gas_refund: GasRefund::new(0),
            output: None,
            stack: Stack::new(&[]),
            memory: Memory::new(&[]),
            last_call_return_data: None,
            steps: None,
        }
    }
}

impl<'a> Interpreter<'a, true> {
    #[allow(clippy::too_many_arguments)]
    pub fn new_steppable(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut dyn ExecutionContextTrait,
        code: &'a [u8],
        pc: usize,
        gas_refund: i64,
        stack: Stack,
        memory: Memory,
        last_call_return_data: Option<Vec<u8>>,
        steps: Option<i32>,
    ) -> Self {
        Self {
            exec_status: ExecStatus::Running,
            message,
            context,
            revision,
            code_reader: CodeReader::new(code, message.code_hash().map(|h| u256::from(*h)), pc),
            gas_left: Gas::new(message.gas()),
            gas_refund: GasRefund::new(gas_refund),
            output: None,
            stack,
            memory,
            last_call_return_data,
            steps,
        }
    }
}

#[allow(clippy::needless_lifetimes)]
impl<'i, const STEPPABLE: bool> Interpreter<'i, STEPPABLE> {
    /// R is expected to be [ExecutionResult] or [StepResult].
    #[cfg(not(feature = "tail-call"))]
    pub fn run<O, R>(mut self, observer: &mut O) -> R
    where
        O: Observer<STEPPABLE>,
        R: From<Self> + From<FailStatus>,
    {
        loop {
            if self.exec_status != ExecStatus::Running {
                break;
            }

            if STEPPABLE {
                match &mut self.steps {
                    None => (),
                    Some(0) => break,
                    Some(steps) => *steps -= 1,
                }
            }
            let op = match self.code_reader.get() {
                Ok(op) => op,
                Err(GetOpcodeError::OutOfRange) => {
                    self.exec_status = ExecStatus::Stopped;
                    break;
                }
                Err(GetOpcodeError::Invalid) => {
                    return FailStatus::InvalidInstruction.into();
                }
            };
            observer.pre_op(&self);
            if let Err(err) = self.run_op(op) {
                return err.into();
            }
            observer.post_op(&self);
        }

        self.into()
    }
    /// R is expected to be [ExecutionResult] or [StepResult].
    #[cfg(feature = "tail-call")]
    #[inline(always)]
    pub fn run<O, R>(mut self, observer: &mut O) -> R
    where
        O: Observer<STEPPABLE>,
        R: From<Interpreter<'i, STEPPABLE>> + From<FailStatus>,
    {
        observer.log("feature \"tail-call\" does not support logging".into());
        if let Err(err) = self.next() {
            return err.into();
        }
        self.into()
    }
    #[cfg(feature = "tail-call")]
    #[inline(always)]
    pub fn next(&mut self) -> OpResult {
        if STEPPABLE {
            match &mut self.steps {
                None => (),
                Some(0) => return Ok(()),
                Some(steps) => *steps -= 1,
            }
        }
        let op = match self.code_reader.get() {
            Ok(op) => op,
            Err(GetOpcodeError::OutOfRange) => {
                self.exec_status = ExecStatus::Stopped;
                return Ok(());
            }
            Err(GetOpcodeError::Invalid) => {
                return Err(FailStatus::InvalidInstruction);
            }
        };
        self.run_op(op)
    }

    #[cfg(feature = "needs-fn-ptr-conversion")]
    fn run_op(&mut self, op: OpFn<STEPPABLE>) -> OpResult {
        op(self)
    }
    #[cfg(all(
        feature = "jumptable-dispatch",
        not(feature = "needs-fn-ptr-conversion")
    ))]
    fn run_op(&mut self, op: Opcode) -> OpResult {
        GenericJumptable::get()[op as u8 as usize](self)
    }
    #[cfg(not(feature = "needs-jumptable"))]
    fn run_op(&mut self, op: Opcode) -> OpResult {
        match op {
            Opcode::Stop => Self::stop(self),
            Opcode::Add => Self::add(self),
            Opcode::Mul => Self::mul(self),
            Opcode::Sub => Self::sub(self),
            Opcode::Div => Self::div(self),
            Opcode::SDiv => Self::s_div(self),
            Opcode::Mod => Self::mod_(self),
            Opcode::SMod => Self::s_mod(self),
            Opcode::AddMod => Self::add_mod(self),
            Opcode::MulMod => Self::mul_mod(self),
            Opcode::Exp => Self::exp(self),
            Opcode::SignExtend => Self::sign_extend(self),
            Opcode::Lt => Self::lt(self),
            Opcode::Gt => Self::gt(self),
            Opcode::SLt => Self::s_lt(self),
            Opcode::SGt => Self::s_gt(self),
            Opcode::Eq => Self::eq(self),
            Opcode::IsZero => Self::is_zero(self),
            Opcode::And => Self::and(self),
            Opcode::Or => Self::or(self),
            Opcode::Xor => Self::xor(self),
            Opcode::Not => Self::not(self),
            Opcode::Byte => Self::byte(self),
            Opcode::Shl => Self::shl(self),
            Opcode::Shr => Self::shr(self),
            Opcode::Sar => Self::sar(self),
            Opcode::Sha3 => Self::sha3(self),
            Opcode::Address => Self::address(self),
            Opcode::Balance => Self::balance(self),
            Opcode::Origin => Self::origin(self),
            Opcode::Caller => Self::caller(self),
            Opcode::CallValue => Self::call_value(self),
            Opcode::CallDataLoad => Self::call_data_load(self),
            Opcode::CallDataSize => Self::call_data_size(self),
            Opcode::CallDataCopy => Self::call_data_copy(self),
            Opcode::CodeSize => Self::code_size(self),
            Opcode::CodeCopy => Self::code_copy(self),
            Opcode::GasPrice => Self::gas_price(self),
            Opcode::ExtCodeSize => Self::ext_code_size(self),
            Opcode::ExtCodeCopy => Self::ext_code_copy(self),
            Opcode::ReturnDataSize => Self::return_data_size(self),
            Opcode::ReturnDataCopy => Self::return_data_copy(self),
            Opcode::ExtCodeHash => Self::ext_code_hash(self),
            Opcode::BlockHash => Self::block_hash(self),
            Opcode::Coinbase => Self::coinbase(self),
            Opcode::Timestamp => Self::timestamp(self),
            Opcode::Number => Self::number(self),
            Opcode::PrevRandao => Self::prev_randao(self),
            Opcode::GasLimit => Self::gas_limit(self),
            Opcode::ChainId => Self::chain_id(self),
            Opcode::SelfBalance => Self::self_balance(self),
            Opcode::BaseFee => Self::base_fee(self),
            Opcode::BlobHash => Self::blob_hash(self),
            Opcode::BlobBaseFee => Self::blob_base_fee(self),
            Opcode::Pop => Self::pop(self),
            Opcode::MLoad => Self::m_load(self),
            Opcode::MStore => Self::m_store(self),
            Opcode::MStore8 => Self::m_store8(self),
            Opcode::SLoad => Self::s_load(self),
            Opcode::SStore => Self::sstore(self),
            Opcode::Jump => Self::jump(self),
            Opcode::JumpI => Self::jump_i(self),
            Opcode::Pc => Self::pc(self),
            Opcode::MSize => Self::m_size(self),
            Opcode::Gas => Self::gas(self),
            Opcode::JumpDest => Self::jump_dest(self),
            Opcode::TLoad => Self::t_load(self),
            Opcode::TStore => Self::t_store(self),
            Opcode::MCopy => Self::m_copy(self),
            Opcode::Push0 => Self::push0(self),
            Opcode::Push1 => Self::push::<1>(self),
            Opcode::Push2 => Self::push::<2>(self),
            Opcode::Push3 => Self::push::<3>(self),
            Opcode::Push4 => Self::push::<4>(self),
            Opcode::Push5 => Self::push::<5>(self),
            Opcode::Push6 => Self::push::<6>(self),
            Opcode::Push7 => Self::push::<7>(self),
            Opcode::Push8 => Self::push::<8>(self),
            Opcode::Push9 => Self::push::<9>(self),
            Opcode::Push10 => Self::push::<10>(self),
            Opcode::Push11 => Self::push::<11>(self),
            Opcode::Push12 => Self::push::<12>(self),
            Opcode::Push13 => Self::push::<13>(self),
            Opcode::Push14 => Self::push::<14>(self),
            Opcode::Push15 => Self::push::<15>(self),
            Opcode::Push16 => Self::push::<16>(self),
            Opcode::Push17 => Self::push::<17>(self),
            Opcode::Push18 => Self::push::<18>(self),
            Opcode::Push19 => Self::push::<19>(self),
            Opcode::Push20 => Self::push::<20>(self),
            Opcode::Push21 => Self::push::<21>(self),
            Opcode::Push22 => Self::push::<22>(self),
            Opcode::Push23 => Self::push::<23>(self),
            Opcode::Push24 => Self::push::<24>(self),
            Opcode::Push25 => Self::push::<25>(self),
            Opcode::Push26 => Self::push::<26>(self),
            Opcode::Push27 => Self::push::<27>(self),
            Opcode::Push28 => Self::push::<28>(self),
            Opcode::Push29 => Self::push::<29>(self),
            Opcode::Push30 => Self::push::<30>(self),
            Opcode::Push31 => Self::push::<31>(self),
            Opcode::Push32 => Self::push::<32>(self),
            Opcode::Dup1 => Self::dup::<1>(self),
            Opcode::Dup2 => Self::dup::<2>(self),
            Opcode::Dup3 => Self::dup::<3>(self),
            Opcode::Dup4 => Self::dup::<4>(self),
            Opcode::Dup5 => Self::dup::<5>(self),
            Opcode::Dup6 => Self::dup::<6>(self),
            Opcode::Dup7 => Self::dup::<7>(self),
            Opcode::Dup8 => Self::dup::<8>(self),
            Opcode::Dup9 => Self::dup::<9>(self),
            Opcode::Dup10 => Self::dup::<10>(self),
            Opcode::Dup11 => Self::dup::<11>(self),
            Opcode::Dup12 => Self::dup::<12>(self),
            Opcode::Dup13 => Self::dup::<13>(self),
            Opcode::Dup14 => Self::dup::<14>(self),
            Opcode::Dup15 => Self::dup::<15>(self),
            Opcode::Dup16 => Self::dup::<16>(self),
            Opcode::Swap1 => Self::swap::<1>(self),
            Opcode::Swap2 => Self::swap::<2>(self),
            Opcode::Swap3 => Self::swap::<3>(self),
            Opcode::Swap4 => Self::swap::<4>(self),
            Opcode::Swap5 => Self::swap::<5>(self),
            Opcode::Swap6 => Self::swap::<6>(self),
            Opcode::Swap7 => Self::swap::<7>(self),
            Opcode::Swap8 => Self::swap::<8>(self),
            Opcode::Swap9 => Self::swap::<9>(self),
            Opcode::Swap10 => Self::swap::<10>(self),
            Opcode::Swap11 => Self::swap::<11>(self),
            Opcode::Swap12 => Self::swap::<12>(self),
            Opcode::Swap13 => Self::swap::<13>(self),
            Opcode::Swap14 => Self::swap::<14>(self),
            Opcode::Swap15 => Self::swap::<15>(self),
            Opcode::Swap16 => Self::swap::<16>(self),
            Opcode::Log0 => Self::log::<0>(self),
            Opcode::Log1 => Self::log::<1>(self),
            Opcode::Log2 => Self::log::<2>(self),
            Opcode::Log3 => Self::log::<3>(self),
            Opcode::Log4 => Self::log::<4>(self),
            Opcode::Create => Self::create(self),
            Opcode::Call => Self::call(self),
            Opcode::CallCode => Self::call_code(self),
            Opcode::Return => Self::return_(self),
            Opcode::DelegateCall => Self::delegate_call(self),
            Opcode::Create2 => Self::create2(self),
            Opcode::StaticCall => Self::static_call(self),
            Opcode::Revert => Self::revert(self),
            Opcode::Invalid => Self::invalid(self),
            Opcode::SelfDestruct => Self::self_destruct(self),
        }
    }

    #[allow(clippy::unused_self)]
    #[inline(always)]
    fn return_from_op(&mut self) -> OpResult {
        #[cfg(not(feature = "tail-call"))]
        return Ok(());
        #[cfg(feature = "tail-call")]
        return self.next();
    }

    #[cfg(feature = "needs-jumptable")]
    pub fn jumptable_placeholder(_i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Err(FailStatus::Failure)
    }

    #[cfg(feature = "needs-fn-ptr-conversion")]
    pub fn no_op(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.code_reader.next();
        i.return_from_op()
    }

    #[cfg(feature = "needs-fn-ptr-conversion")]
    pub fn skip_no_ops(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.code_reader.jump_to();
        i.return_from_op()
    }

    fn stop(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.exec_status = ExecStatus::Stopped;
        Ok(())
    }

    fn add(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value2, value1] = i.stack.pop()?;
        i.stack.push(value1 + value2)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn mul(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(5)?;
        let [fac2, fac1] = i.stack.pop()?;
        i.stack.push(fac1 * fac2)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn sub(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value2, value1] = i.stack.pop()?;
        i.stack.push(value1 - value2)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn div(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(5)?;
        let [denominator, value] = i.stack.pop()?;
        i.stack.push(value / denominator)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn s_div(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(5)?;
        let [denominator, value] = i.stack.pop()?;
        i.stack.push(value.sdiv(denominator))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn mod_(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(5)?;
        let [denominator, value] = i.stack.pop()?;
        i.stack.push(value % denominator)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn s_mod(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(5)?;
        let [denominator, value] = i.stack.pop()?;
        i.stack.push(value.srem(denominator))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn add_mod(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(8)?;
        let [denominator, value2, value1] = i.stack.pop()?;
        i.stack.push(u256::addmod(value1, value2, denominator))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn mul_mod(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(8)?;
        let [denominator, fac2, fac1] = i.stack.pop()?;
        i.stack.push(u256::mulmod(fac1, fac2, denominator))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn exp(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(10)?;
        let [exp, value] = i.stack.pop()?;
        i.gas_left.consume(exp.bits().div_ceil(8) as u64 * 50)?; // * does not overflow
        i.stack.push(value.pow(exp))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn sign_extend(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(5)?;
        let [value, size] = i.stack.pop()?;
        i.stack.push(u256::signextend(size, value))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn lt(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs < rhs)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn gt(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs > rhs)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn s_lt(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs.slt(&rhs))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn s_gt(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs.sgt(&rhs))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn eq(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs == rhs)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn is_zero(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value] = i.stack.pop()?;
        i.stack.push(value == u256::ZERO)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn and(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs & rhs)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn or(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs | rhs)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn xor(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [rhs, lhs] = i.stack.pop()?;
        i.stack.push(lhs ^ rhs)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn not(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value] = i.stack.pop()?;
        i.stack.push(!value)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn byte(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value, offset] = i.stack.pop()?;
        i.stack.push(value.byte(offset))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn shl(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value, shift] = i.stack.pop()?;
        i.stack.push(value << shift)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn shr(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value, shift] = i.stack.pop()?;
        i.stack.push(value >> shift)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn sar(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value, shift] = i.stack.pop()?;
        i.stack.push(value.sar(shift))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn sha3(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(30)?;
        let [len, offset] = i.stack.pop()?;

        let len = u64::try_from(len).map_err(|_| FailStatus::OutOfGas)?;
        i.gas_left.consume(6 * word_size(len)?)?; // * does not overflow

        let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
        i.stack.push(hash_cache::hash(data))?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn address(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.message.recipient())?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn balance(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn origin(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.context.get_tx_context().tx_origin)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn caller(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.message.sender())?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn call_value(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(*i.message.value())?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn call_data_load(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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
            let mut bytes = [0; 32];
            bytes[..end - offset].copy_from_slice(&call_data[offset..end]);
            i.stack.push(u256::from_be_bytes(bytes))?;
        }
        i.code_reader.next();
        i.return_from_op()
    }

    fn call_data_size(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn push0(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_min_revision(Revision::EVMC_SHANGHAI, i.revision)?;
        i.gas_left.consume(2)?;
        i.stack.push(u256::ZERO)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn call_data_copy(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [len, offset, dest_offset] = i.stack.pop()?;

        if len != u256::ZERO {
            let len = u64::try_from(len).map_err(|_| FailStatus::InvalidMemoryAccess)?;

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

    fn code_size(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.code_reader.len())?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn code_copy(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [len, offset, dest_offset] = i.stack.pop()?;

        if len != u256::ZERO {
            let len = u64::try_from(len).map_err(|_| FailStatus::OutOfGas)?;

            let src = i.code_reader.get_within_bounds(offset, len);
            let dest = i.memory.get_mut_slice(dest_offset, len, &mut i.gas_left)?;
            dest.copy_padded(src, &mut i.gas_left)?;
        }
        i.code_reader.next();
        i.return_from_op()
    }

    fn gas_price(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.context.get_tx_context().tx_gas_price)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn ext_code_size(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn ext_code_copy(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        if i.revision < Revision::EVMC_BERLIN {
            i.gas_left.consume(700)?;
        }
        let [len, offset, dest_offset, addr] = i.stack.pop()?;
        let addr = addr.into();

        i.gas_left
            .consume_address_access_cost(&addr, i.revision, i.context)?;
        if len != u256::ZERO {
            let len = u64::try_from(len).map_err(|_| FailStatus::OutOfGas)?;

            let dest = i.memory.get_mut_slice(dest_offset, len, &mut i.gas_left)?;
            let (offset, offset_overflow) = offset.into_u64_with_overflow();
            i.gas_left.consume_copy_cost(len)?;
            let bytes_written = i.context.copy_code(&addr, offset as usize, dest);
            if offset_overflow {
                dest.fill(0);
            } else if (bytes_written as u64) < len {
                dest[bytes_written..].fill(0);
            }
        }
        i.code_reader.next();
        i.return_from_op()
    }

    fn return_data_size(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn return_data_copy(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn ext_code_hash(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn block_hash(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(20)?;
        let [block_number] = i.stack.pop()?;
        i.stack.push(
            u64::try_from(block_number)
                .map(|idx| i.context.get_block_hash(idx as i64).into())
                .unwrap_or(u256::ZERO),
        )?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn coinbase(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.context.get_tx_context().block_coinbase)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn timestamp(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack
            .push(i.context.get_tx_context().block_timestamp as u64)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn number(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack
            .push(i.context.get_tx_context().block_number as u64)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn prev_randao(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.context.get_tx_context().block_prev_randao)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn gas_limit(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack
            .push(i.context.get_tx_context().block_gas_limit as u64)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn chain_id(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.context.get_tx_context().chain_id)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn self_balance(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn base_fee(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_min_revision(Revision::EVMC_LONDON, i.revision)?;
        i.gas_left.consume(2)?;
        i.stack.push(i.context.get_tx_context().block_base_fee)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn blob_hash(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn blob_base_fee(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
        i.gas_left.consume(2)?;
        i.stack.push(i.context.get_tx_context().blob_base_fee)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn pop(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        let [_] = i.stack.pop()?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn m_load(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [offset] = i.stack.pop()?;

        i.stack.push(i.memory.get_word(offset, &mut i.gas_left)?)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn m_store(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value, offset] = i.stack.pop()?;

        let dest = i.memory.get_mut_slice(offset, 32, &mut i.gas_left)?;
        dest.copy_from_slice(&value.to_le_bytes());
        dest.reverse();
        i.code_reader.next();
        i.return_from_op()
    }

    fn m_store8(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        let [value, offset] = i.stack.pop()?;

        let dest = i.memory.get_mut_byte(offset, &mut i.gas_left)?;
        *dest = value.least_significant_byte();
        i.code_reader.next();
        i.return_from_op()
    }

    fn s_load(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn jump(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(if STEPPABLE { 8 } else { 8 + 1 })?;
        let [dest] = i.stack.pop()?;
        i.code_reader.try_jump(dest)?;
        if !STEPPABLE {
            i.code_reader.next();
        }
        i.return_from_op()
    }

    fn jump_i(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(10)?;
        let [cond, dest] = i.stack.pop()?;
        if cond == u256::ZERO {
            i.code_reader.next();
        } else {
            i.code_reader.try_jump(dest)?;
            if !STEPPABLE {
                i.gas_left.consume(1)?;
                i.code_reader.next();
            }
        }
        i.return_from_op()
    }

    fn pc(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.code_reader.pc())?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn m_size(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.memory.len())?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn gas(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(2)?;
        i.stack.push(i.gas_left.as_u64())?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn jump_dest(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(1)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn t_load(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
        i.gas_left.consume(100)?;
        let [key] = i.stack.pop()?;
        let addr = i.message.recipient();
        let value = i.context.get_transient_storage(addr, &key.into());
        i.stack.push(value)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn t_store(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_min_revision(Revision::EVMC_CANCUN, i.revision)?;
        check_not_read_only(i.message)?;
        i.gas_left.consume(100)?;
        let [value, key] = i.stack.pop()?;
        let addr = i.message.recipient();
        i.context
            .set_transient_storage(addr, &key.into(), &value.into());
        i.code_reader.next();
        i.return_from_op()
    }

    fn m_copy(i: &mut Interpreter<STEPPABLE>) -> OpResult {
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

    fn return_(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        let [len, offset] = i.stack.pop()?;
        let len = u64::try_from(len).map_err(|_| FailStatus::OutOfGas)?;
        let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
        #[cfg(not(feature = "custom-evmc"))]
        {
            i.output = Some(data.to_owned());
        }
        #[cfg(feature = "custom-evmc")]
        {
            i.output = Some(Box::from(&*data));
        }
        i.exec_status = ExecStatus::Returned;
        Ok(())
    }

    fn revert(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        let [len, offset] = i.stack.pop()?;
        let len = u64::try_from(len).map_err(|_| FailStatus::OutOfGas)?;
        let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
        #[cfg(not(feature = "custom-evmc"))]
        {
            i.output = Some(data.to_owned());
        }
        #[cfg(feature = "custom-evmc")]
        {
            i.output = Some(Box::from(&*data));
        }
        i.exec_status = ExecStatus::Revert;
        Ok(())
    }

    #[allow(clippy::unused_self)]
    fn invalid(_i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Err(FailStatus::InvalidInstruction)
    }

    fn self_destruct(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_not_read_only(i.message)?;
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
            i.gas_refund.add(24_000)?;
        }

        i.exec_status = ExecStatus::Stopped;
        Ok(())
    }

    fn sstore(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_not_read_only(i.message)?;

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
        i.gas_refund.add(gas_refund_change)?;
        i.code_reader.next();
        i.return_from_op()
    }

    #[allow(unused_variables)]
    fn push<const N: usize>(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        #[cfg(not(feature = "fn-ptr-conversion-expanded-dispatch"))]
        i.code_reader.next();
        #[cfg(not(feature = "fn-ptr-conversion-expanded-dispatch"))]
        i.stack.push(i.code_reader.get_push_data::<N>())?;
        #[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
        i.stack.push(i.code_reader.get_push_data())?;
        i.return_from_op()
    }

    fn dup<const N: usize>(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        i.stack.dup::<N>()?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn swap<const N: usize>(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(3)?;
        i.stack.swap_with_top::<N>()?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn log<const N: usize>(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        check_not_read_only(i.message)?;
        i.gas_left.consume(375)?;
        let [len, offset] = i.stack.pop()?;
        let topics: [u256; N] = i.stack.pop()?;
        let (len, len_overflow) = len.into_u64_with_overflow();
        let (len8, len8_overflow) = len.overflowing_mul(8);
        let (cost, cost_overflow) = (375 * N as u64).overflowing_add(len8);
        if len_overflow || len8_overflow || cost_overflow {
            return Err(FailStatus::OutOfGas);
        }
        i.gas_left.consume(cost)?;

        let data = i.memory.get_mut_slice(offset, len, &mut i.gas_left)?;
        let mut topics_uint256 = [Uint256 { bytes: [0; 32] }; N];
        for i in 0..N {
            topics_uint256[i] = Uint256::from(topics[N - 1 - i]);
        }
        i.context
            .emit_log(i.message.recipient(), data, &topics_uint256);
        i.code_reader.next();
        i.return_from_op()
    }

    fn create(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Interpreter::create_or_create2::<false>(i)
    }

    fn create2(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Interpreter::create_or_create2::<true>(i)
    }

    fn create_or_create2<const CREATE2: bool>(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        i.gas_left.consume(32_000)?;
        check_not_read_only(i.message)?;
        let [len, offset, value] = i.stack.pop()?;
        let salt = if CREATE2 {
            let [salt] = i.stack.pop()?;
            salt
        } else {
            u256::ZERO // ignored
        };
        let len = u64::try_from(len).map_err(|_| FailStatus::OutOfGas)?;

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

        i.gas_left.add(result.gas_left())?;
        i.gas_refund.add(result.gas_refund())?;

        if result.status_code() == StatusCode::EVMC_SUCCESS {
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

    fn call(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Interpreter::call_or_call_code::<false>(i)
    }

    fn call_code(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Interpreter::call_or_call_code::<true>(i)
    }

    fn call_or_call_code<const CODE: bool>(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        if i.revision < Revision::EVMC_BERLIN {
            i.gas_left.consume(700)?;
        }
        let [ret_len, ret_offset, args_len, args_offset, value, addr, gas] = i.stack.pop()?;

        if !CODE && value != u256::ZERO {
            check_not_read_only(i.message)?;
        }

        let addr = addr.into();
        let args_len = u64::try_from(args_len).map_err(|_| FailStatus::OutOfGas)?;
        let ret_len = u64::try_from(ret_len).map_err(|_| FailStatus::OutOfGas)?;

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
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left

        let stipend: u64 = if value == u256::ZERO { 0 } else { 2_300 };
        i.gas_left.add(stipend as i64)?;

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
                addr,
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

        i.gas_left.add(result.gas_left())?;
        i.gas_left.consume(endowment)?;
        i.gas_left.consume(stipend)?;
        i.gas_refund.add(result.gas_refund())?;

        i.stack
            .push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
        i.code_reader.next();
        i.return_from_op()
    }

    fn static_call(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Interpreter::static_or_delegate_call::<false>(i)
    }

    fn delegate_call(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        Interpreter::static_or_delegate_call::<true>(i)
    }

    fn static_or_delegate_call<const DELEGATE: bool>(i: &mut Interpreter<STEPPABLE>) -> OpResult {
        if i.revision < Revision::EVMC_BERLIN {
            i.gas_left.consume(700)?;
        }
        let [ret_len, ret_offset, args_len, args_offset, addr, gas] = i.stack.pop()?;

        let addr = addr.into();
        let args_len = u64::try_from(args_len).map_err(|_| FailStatus::OutOfGas)?;
        let ret_len = u64::try_from(ret_len).map_err(|_| FailStatus::OutOfGas)?;

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
        endowment = min(endowment, limit); // cap gas at all but one 64th of gas left

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
                addr,
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

        i.gas_left.add(result.gas_left())?;
        i.gas_left.consume(endowment)?;
        i.gas_refund.add(result.gas_refund())?;

        i.stack
            .push(result.status_code() == StatusCode::EVMC_SUCCESS)?;
        i.code_reader.next();
        i.return_from_op()
    }
}

impl<const STEPPABLE: bool> From<Interpreter<'_, STEPPABLE>> for StepResult {
    fn from(value: Interpreter<STEPPABLE>) -> Self {
        let stack = value
            .stack
            .as_slice()
            .iter()
            .copied()
            .map(Into::into)
            .collect();
        Self::new(
            value.exec_status.into(),
            StatusCode::EVMC_SUCCESS,
            value.revision,
            value.code_reader.pc() as u64,
            value.gas_left.as_u64() as i64,
            value.gas_refund.as_i64(),
            value.output,
            stack,
            value.memory.as_slice().to_vec(),
            value.last_call_return_data,
        )
    }
}

impl<const STEPPABLE: bool> From<Interpreter<'_, STEPPABLE>> for ExecutionResult {
    fn from(value: Interpreter<STEPPABLE>) -> Self {
        Self::new(
            value.exec_status.into(),
            value.gas_left.as_u64() as i64,
            value.gas_refund.as_i64(),
            #[cfg(not(feature = "custom-evmc"))]
            value.output.as_deref(),
            #[cfg(feature = "custom-evmc")]
            value.output,
        )
    }
}

#[cfg(test)]
mod tests {
    use evmc_vm::{
        Address, ExecutionResult, MessageKind, Revision, StatusCode, StepResult, StepStatusCode,
        Uint256,
    };
    use mockall::predicate;

    use crate::{
        interpreter::Interpreter,
        types::{
            u256, Memory, MockExecutionContextTrait, MockExecutionMessage, NoOpObserver, Opcode,
            Stack,
        },
    };

    #[test]
    fn empty_code() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let interpreter = Interpreter::new(Revision::EVMC_ISTANBUL, &message, &mut context, &[]);
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_STOPPED);
        assert_eq!(result.pc, 0);
        assert_eq!(
            result.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS as i64
        );
    }

    #[test]
    fn pc_after_end() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let interpreter = Interpreter::new_steppable(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Add as u8],
            1,
            0,
            Stack::new(&[]),
            Memory::new(&[]),
            None,
            None,
        );
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_STOPPED);
        assert_eq!(result.pc, 1);
        assert_eq!(
            result.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS as i64
        );
    }

    // when features "fn-ptr-conversion-expanded-dispatch"  or
    // "fn-ptr-conversion-inline-dispatch" are enabled this in undefined behavior
    #[cfg(not(feature = "needs-fn-ptr-conversion"))]
    #[test]
    fn pc_on_data() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let result: ExecutionResult = Interpreter::new_steppable(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Push1 as u8, 0x00],
            1,
            0,
            Stack::new(&[]),
            Memory::new(&[]),
            None,
            None,
        )
        .run(&mut NoOpObserver());
        assert_eq!(result.status_code, StatusCode::EVMC_INVALID_INSTRUCTION);
    }

    #[test]
    fn zero_steps() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let interpreter = Interpreter::new_steppable(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Add as u8],
            0,
            0,
            Stack::new(&[]),
            Memory::new(&[]),
            None,
            Some(0),
        );
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_RUNNING);
        assert_eq!(result.pc, 0);
        assert_eq!(
            result.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS as i64
        );
    }

    #[test]
    fn add_one_step() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let interpreter = Interpreter::new_steppable(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Add as u8, Opcode::Add as u8],
            0,
            0,
            Stack::new(&[1u8.into(), 2u8.into()]),
            Memory::new(&[]),
            None,
            Some(1),
        );
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_RUNNING);
        assert_eq!(result.stack.as_slice(), [u256::from(3u8).into()]);
        assert_eq!(
            result.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS as i64 - 3
        );
    }

    #[test]
    fn add_single_op() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter = Interpreter::new(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Add as u8],
        );
        interpreter.stack = Stack::new(&[1u8.into(), 2u8.into()]);
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_STOPPED);
        assert_eq!(result.stack.as_slice(), [u256::from(3u8).into()]);
        assert_eq!(
            result.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS as i64 - 3
        );
    }

    #[test]
    fn add_twice() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter = Interpreter::new(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Add as u8, Opcode::Add as u8],
        );
        interpreter.stack = Stack::new(&[1u8.into(), 2u8.into(), 3u8.into()]);
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_STOPPED);
        assert_eq!(result.stack.as_slice(), [u256::from(6u8).into()]);
        assert_eq!(
            result.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS as i64 - 2 * 3
        );
    }

    #[cfg(not(debug_assertions))]
    #[test]
    // When feature tail-call is enabled, but the tail calls are not eliminated the stack will
    // overflow if enough operations are executed. This test makes sure that does not happen.
    // Because it will fail when compiled without optimizations, it is only enabled when
    // debug_assertions are not enabled (the default in release mode).
    fn tail_call_elimination() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let interpreter = Interpreter::new(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::JumpDest as u8; 10_000_000],
        );
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_STOPPED);
    }

    #[test]
    fn add_not_enough_gas() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage {
            gas: 2,
            ..Default::default()
        };
        let message = message.into();
        let mut interpreter = Interpreter::new(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Add as u8],
        );
        interpreter.stack = Stack::new(&[1u8.into(), 2u8.into()]);
        let result: ExecutionResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.status_code, StatusCode::EVMC_OUT_OF_GAS);
    }

    #[test]
    fn call() {
        // helpers to generate unique values; random values are not needed
        let mut unique_values = 1u8..;
        let mut next_value = || unique_values.next().unwrap();

        let memory = [next_value(), next_value(), next_value(), next_value()];
        let ret_data = [next_value(), next_value()];

        let gas = next_value() as u64;
        let addr = next_value().into();
        let value = u256::ZERO;
        let args_offset = 1usize;
        let args_len = memory.len() - args_offset - 1;
        let ret_offset = 1usize;
        let ret_len = ret_data.len();

        let input = memory[args_offset..args_offset + args_len].to_vec();

        let message = MockExecutionMessage {
            recipient: u256::from(next_value()).into(),
            ..Default::default()
        };

        let mut context = MockExecutionContextTrait::new();
        context
            .expect_get_balance()
            .times(1)
            .with(predicate::eq(Address::from(message.recipient)))
            .return_const(Uint256::from(u256::ZERO));
        context
            .expect_call()
            .times(1)
            .withf(move |call_message| {
                call_message.kind() == MessageKind::EVMC_CALL
                    && call_message.flags() == 0
                    && call_message.depth() == message.depth + 1
                    && call_message.gas() == gas as i64
                    && call_message.sender() == &message.recipient
                    && call_message.recipient() == &Address::from(addr)
                    && call_message.input() == Some(&input)
                    && call_message.value() == &Uint256::from(value)
                    && call_message.create2_salt() == &Uint256::from(u256::ZERO)
                    && call_message.code_address() == &Address::from(addr)
                    && call_message.code().is_none()
            })
            .returning(move |_| {
                #[cfg(not(feature = "custom-evmc"))]
                return ExecutionResult::new(StatusCode::EVMC_SUCCESS, 0, 0, Some(&ret_data));
                #[cfg(feature = "custom-evmc")]
                return ExecutionResult::new(
                    StatusCode::EVMC_SUCCESS,
                    0,
                    0,
                    Some(Box::from(ret_data.as_slice())),
                );
            });

        let message = message.into();

        let stack = [
            ret_len.into(),
            ret_offset.into(),
            args_len.into(),
            args_offset.into(),
            value,
            addr,
            gas.into(),
        ];

        let interpreter = Interpreter::new_steppable(
            Revision::EVMC_ISTANBUL,
            &message,
            &mut context,
            &[Opcode::Call as u8],
            0,
            0,
            Stack::new(&stack),
            Memory::new(&memory),
            None,
            None,
        );
        let result: StepResult = interpreter.run(&mut NoOpObserver());
        assert_eq!(result.step_status_code, StepStatusCode::EVMC_STEP_STOPPED);
        assert_eq!(result.pc, 1);
        assert_eq!(
            result.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS as i64 - 700 - gas as i64
        );
        assert_eq!(
            result.last_call_return_data.as_deref(),
            Some(ret_data.as_slice())
        );
        assert_eq!(
            &result.memory[ret_offset..ret_offset + ret_len],
            ret_data.as_slice()
        );
    }
}

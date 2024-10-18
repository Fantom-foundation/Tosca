use evmc_vm::{
    ExecutionMessage, ExecutionResult, Revision, StatusCode as EvmcStatusCode, StepResult,
};

use crate::{
    types::{
        u256, CodeReader, ExecStatus, ExecutionContextTrait, FailStatus, GetOpcodeError, Memory,
        Opcode, Stack,
    },
    utils::Gas,
};

mod opcode_impls;
use opcode_impls::*;

type OpResult = Result<(), FailStatus>;

#[cfg(feature = "jumptable")]
type OpFn<E> = fn(&mut Interpreter<E>) -> OpResult;

#[derive(Debug)]
pub struct Interpreter<'a, E>
where
    E: ExecutionContextTrait,
{
    pub exec_status: ExecStatus,
    #[cfg(not(feature = "custom-evmc"))]
    pub message: &'a ExecutionMessage,
    #[cfg(feature = "custom-evmc")]
    pub message: &'a ExecutionMessage<'a>,
    pub context: &'a mut E,
    pub revision: Revision,
    pub code_reader: CodeReader<'a>,
    pub gas_left: Gas,
    pub gas_refund: i64,
    #[cfg(not(feature = "custom-evmc"))]
    pub output: Option<Vec<u8>>,
    #[cfg(feature = "custom-evmc")]
    pub output: Option<Box<[u8]>>,
    pub stack: Stack,
    pub memory: Memory,
    pub last_call_return_data: Option<Vec<u8>>,
    pub steps: Option<i32>,
}

impl<'a, E> Interpreter<'a, E>
where
    E: ExecutionContextTrait,
{
    pub fn new(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut E,
        code: &'a [u8],
    ) -> Self {
        Self {
            exec_status: ExecStatus::Running,
            message,
            context,
            revision,
            code_reader: CodeReader::new(code, message.code_hash().map(|h| u256::from(*h)), 0),
            gas_left: Gas::new(message.gas() as u64),
            gas_refund: 0,
            output: None,
            stack: Stack::new(&[]),
            memory: Memory::new(Vec::new()),
            last_call_return_data: None,
            steps: None,
        }
    }

    #[allow(clippy::too_many_arguments)]
    pub fn new_steppable(
        revision: Revision,
        message: &'a ExecutionMessage,
        context: &'a mut E,
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
            gas_left: Gas::new(message.gas() as u64),
            gas_refund,
            output: None,
            stack,
            memory,
            last_call_return_data,
            steps,
        }
    }

    #[cfg(not(feature = "jumptable-tail-call"))]
    pub fn run(&mut self) -> Result<(), FailStatus> {
        loop {
            if self.exec_status != ExecStatus::Running {
                break;
            }

            match &mut self.steps {
                None => (),
                Some(0) => break,
                Some(steps) => *steps -= 1,
            }
            let op = match self.code_reader.get() {
                Ok(op) => op,
                Err(GetOpcodeError::OutOfRange) => {
                    self.exec_status = ExecStatus::Stopped;
                    break;
                }
                Err(GetOpcodeError::Invalid) => {
                    return Err(FailStatus::InvalidInstruction);
                }
            };
            self.run_op(op)?;
        }

        Ok(())
    }
    #[cfg(feature = "jumptable-tail-call")]
    #[inline(always)]
    pub fn run(&mut self) -> Result<(), FailStatus> {
        match &mut self.steps {
            None => (),
            Some(0) => return Ok(()),
            Some(steps) => *steps -= 1,
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

    #[cfg(feature = "jumptable")]
    const JUMPTABLE: [OpFn<E>; 256] = [
        stop,
        add,
        mul,
        sub,
        div,
        s_div,
        mod_,
        s_mod,
        add_mod,
        mul_mod,
        exp,
        sign_extend,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        lt,
        gt,
        s_lt,
        s_gt,
        eq,
        is_zero,
        and,
        or,
        xor,
        not,
        byte,
        shl,
        shr,
        sar,
        jumptable_placeholder,
        jumptable_placeholder,
        sha3,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        address,
        balance,
        origin,
        caller,
        call_value,
        call_data_load,
        call_data_size,
        call_data_copy,
        code_size,
        code_copy,
        gas_price,
        ext_code_size,
        ext_code_copy,
        return_data_size,
        return_data_copy,
        ext_code_hash,
        block_hash,
        coinbase,
        timestamp,
        number,
        prev_randao,
        gas_limit,
        chain_id,
        self_balance,
        base_fee,
        blob_hash,
        blob_base_fee,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        pop,
        m_load,
        m_store,
        m_store8,
        s_load,
        sstore,
        jump,
        jump_i,
        pc,
        m_size,
        gas,
        jump_dest,
        t_load,
        t_store,
        m_copy,
        push0,
        (|s| push(s, 1)) as OpFn<_>,
        (|s| push(s, 2)) as OpFn<_>,
        (|s| push(s, 3)) as OpFn<_>,
        (|s| push(s, 4)) as OpFn<_>,
        (|s| push(s, 5)) as OpFn<_>,
        (|s| push(s, 6)) as OpFn<_>,
        (|s| push(s, 7)) as OpFn<_>,
        (|s| push(s, 8)) as OpFn<_>,
        (|s| push(s, 9)) as OpFn<_>,
        (|s| push(s, 10)) as OpFn<_>,
        (|s| push(s, 11)) as OpFn<_>,
        (|s| push(s, 12)) as OpFn<_>,
        (|s| push(s, 13)) as OpFn<_>,
        (|s| push(s, 14)) as OpFn<_>,
        (|s| push(s, 15)) as OpFn<_>,
        (|s| push(s, 16)) as OpFn<_>,
        (|s| push(s, 17)) as OpFn<_>,
        (|s| push(s, 18)) as OpFn<_>,
        (|s| push(s, 19)) as OpFn<_>,
        (|s| push(s, 20)) as OpFn<_>,
        (|s| push(s, 21)) as OpFn<_>,
        (|s| push(s, 22)) as OpFn<_>,
        (|s| push(s, 23)) as OpFn<_>,
        (|s| push(s, 24)) as OpFn<_>,
        (|s| push(s, 25)) as OpFn<_>,
        (|s| push(s, 26)) as OpFn<_>,
        (|s| push(s, 27)) as OpFn<_>,
        (|s| push(s, 28)) as OpFn<_>,
        (|s| push(s, 29)) as OpFn<_>,
        (|s| push(s, 30)) as OpFn<_>,
        (|s| push(s, 31)) as OpFn<_>,
        (|s| push(s, 32)) as OpFn<_>,
        (|s| dup(s, 1)) as OpFn<_>,
        (|s| dup(s, 2)) as OpFn<_>,
        (|s| dup(s, 3)) as OpFn<_>,
        (|s| dup(s, 4)) as OpFn<_>,
        (|s| dup(s, 5)) as OpFn<_>,
        (|s| dup(s, 6)) as OpFn<_>,
        (|s| dup(s, 7)) as OpFn<_>,
        (|s| dup(s, 8)) as OpFn<_>,
        (|s| dup(s, 9)) as OpFn<_>,
        (|s| dup(s, 10)) as OpFn<_>,
        (|s| dup(s, 11)) as OpFn<_>,
        (|s| dup(s, 12)) as OpFn<_>,
        (|s| dup(s, 13)) as OpFn<_>,
        (|s| dup(s, 14)) as OpFn<_>,
        (|s| dup(s, 15)) as OpFn<_>,
        (|s| dup(s, 16)) as OpFn<_>,
        (|s| swap(s, 1)) as OpFn<_>,
        (|s| swap(s, 2)) as OpFn<_>,
        (|s| swap(s, 3)) as OpFn<_>,
        (|s| swap(s, 4)) as OpFn<_>,
        (|s| swap(s, 5)) as OpFn<_>,
        (|s| swap(s, 6)) as OpFn<_>,
        (|s| swap(s, 7)) as OpFn<_>,
        (|s| swap(s, 8)) as OpFn<_>,
        (|s| swap(s, 9)) as OpFn<_>,
        (|s| swap(s, 10)) as OpFn<_>,
        (|s| swap(s, 11)) as OpFn<_>,
        (|s| swap(s, 12)) as OpFn<_>,
        (|s| swap(s, 13)) as OpFn<_>,
        (|s| swap(s, 14)) as OpFn<_>,
        (|s| swap(s, 15)) as OpFn<_>,
        (|s| swap(s, 16)) as OpFn<_>,
        log::<0, _>,
        log::<1, _>,
        log::<2, _>,
        log::<3, _>,
        log::<4, _>,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        create,
        call,
        call_code,
        return_,
        delegate_call,
        create2,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        jumptable_placeholder,
        static_call,
        jumptable_placeholder,
        jumptable_placeholder,
        revert,
        invalid,
        self_destruct,
    ];

    #[cfg(feature = "jumptable")]
    fn run_op(&mut self, op: Opcode) -> OpResult {
        Self::JUMPTABLE[op as u8 as usize](self)
    }

    #[cfg(not(feature = "jumptable"))]
    fn run_op(&mut self, op: Opcode) -> OpResult {
        match op {
            Opcode::Stop => stop(self),
            Opcode::Add => add(self),
            Opcode::Mul => mul(self),
            Opcode::Sub => sub(self),
            Opcode::Div => div(self),
            Opcode::SDiv => s_div(self),
            Opcode::Mod => mod_(self),
            Opcode::SMod => s_mod(self),
            Opcode::AddMod => add_mod(self),
            Opcode::MulMod => mul_mod(self),
            Opcode::Exp => exp(self),
            Opcode::SignExtend => sign_extend(self),
            Opcode::Lt => lt(self),
            Opcode::Gt => gt(self),
            Opcode::SLt => s_lt(self),
            Opcode::SGt => s_gt(self),
            Opcode::Eq => eq(self),
            Opcode::IsZero => is_zero(self),
            Opcode::And => and(self),
            Opcode::Or => or(self),
            Opcode::Xor => xor(self),
            Opcode::Not => not(self),
            Opcode::Byte => byte(self),
            Opcode::Shl => shl(self),
            Opcode::Shr => shr(self),
            Opcode::Sar => sar(self),
            Opcode::Sha3 => sha3(self),
            Opcode::Address => address(self),
            Opcode::Balance => balance(self),
            Opcode::Origin => origin(self),
            Opcode::Caller => caller(self),
            Opcode::CallValue => call_value(self),
            Opcode::CallDataLoad => call_data_load(self),
            Opcode::CallDataSize => call_data_size(self),
            Opcode::CallDataCopy => call_data_copy(self),
            Opcode::CodeSize => code_size(self),
            Opcode::CodeCopy => code_copy(self),
            Opcode::GasPrice => gas_price(self),
            Opcode::ExtCodeSize => ext_code_size(self),
            Opcode::ExtCodeCopy => ext_code_copy(self),
            Opcode::ReturnDataSize => return_data_size(self),
            Opcode::ReturnDataCopy => return_data_copy(self),
            Opcode::ExtCodeHash => ext_code_hash(self),
            Opcode::BlockHash => block_hash(self),
            Opcode::Coinbase => coinbase(self),
            Opcode::Timestamp => timestamp(self),
            Opcode::Number => number(self),
            Opcode::PrevRandao => prev_randao(self),
            Opcode::GasLimit => gas_limit(self),
            Opcode::ChainId => chain_id(self),
            Opcode::SelfBalance => self_balance(self),
            Opcode::BaseFee => base_fee(self),
            Opcode::BlobHash => blob_hash(self),
            Opcode::BlobBaseFee => blob_base_fee(self),
            Opcode::Pop => pop(self),
            Opcode::MLoad => m_load(self),
            Opcode::MStore => m_store(self),
            Opcode::MStore8 => m_store8(self),
            Opcode::SLoad => s_load(self),
            Opcode::SStore => sstore(self),
            Opcode::Jump => jump(self),
            Opcode::JumpI => jump_i(self),
            Opcode::Pc => pc(self),
            Opcode::MSize => m_size(self),
            Opcode::Gas => gas(self),
            Opcode::JumpDest => jump_dest(self),
            Opcode::TLoad => t_load(self),
            Opcode::TStore => t_store(self),
            Opcode::MCopy => m_copy(self),
            Opcode::Push0 => push0(self),
            Opcode::Push1 => push(self, 1),
            Opcode::Push2 => push(self, 2),
            Opcode::Push3 => push(self, 3),
            Opcode::Push4 => push(self, 4),
            Opcode::Push5 => push(self, 5),
            Opcode::Push6 => push(self, 6),
            Opcode::Push7 => push(self, 7),
            Opcode::Push8 => push(self, 8),
            Opcode::Push9 => push(self, 9),
            Opcode::Push10 => push(self, 10),
            Opcode::Push11 => push(self, 11),
            Opcode::Push12 => push(self, 12),
            Opcode::Push13 => push(self, 13),
            Opcode::Push14 => push(self, 14),
            Opcode::Push15 => push(self, 15),
            Opcode::Push16 => push(self, 16),
            Opcode::Push17 => push(self, 17),
            Opcode::Push18 => push(self, 18),
            Opcode::Push19 => push(self, 19),
            Opcode::Push20 => push(self, 20),
            Opcode::Push21 => push(self, 21),
            Opcode::Push22 => push(self, 22),
            Opcode::Push23 => push(self, 23),
            Opcode::Push24 => push(self, 24),
            Opcode::Push25 => push(self, 25),
            Opcode::Push26 => push(self, 26),
            Opcode::Push27 => push(self, 27),
            Opcode::Push28 => push(self, 28),
            Opcode::Push29 => push(self, 29),
            Opcode::Push30 => push(self, 30),
            Opcode::Push31 => push(self, 31),
            Opcode::Push32 => push(self, 32),
            Opcode::Dup1 => dup(self, 1),
            Opcode::Dup2 => dup(self, 2),
            Opcode::Dup3 => dup(self, 3),
            Opcode::Dup4 => dup(self, 4),
            Opcode::Dup5 => dup(self, 5),
            Opcode::Dup6 => dup(self, 6),
            Opcode::Dup7 => dup(self, 7),
            Opcode::Dup8 => dup(self, 8),
            Opcode::Dup9 => dup(self, 9),
            Opcode::Dup10 => dup(self, 10),
            Opcode::Dup11 => dup(self, 11),
            Opcode::Dup12 => dup(self, 12),
            Opcode::Dup13 => dup(self, 13),
            Opcode::Dup14 => dup(self, 14),
            Opcode::Dup15 => dup(self, 15),
            Opcode::Dup16 => dup(self, 16),
            Opcode::Swap1 => swap(self, 1),
            Opcode::Swap2 => swap(self, 2),
            Opcode::Swap3 => swap(self, 3),
            Opcode::Swap4 => swap(self, 4),
            Opcode::Swap5 => swap(self, 5),
            Opcode::Swap6 => swap(self, 6),
            Opcode::Swap7 => swap(self, 7),
            Opcode::Swap8 => swap(self, 8),
            Opcode::Swap9 => swap(self, 9),
            Opcode::Swap10 => swap(self, 10),
            Opcode::Swap11 => swap(self, 11),
            Opcode::Swap12 => swap(self, 12),
            Opcode::Swap13 => swap(self, 13),
            Opcode::Swap14 => swap(self, 14),
            Opcode::Swap15 => swap(self, 15),
            Opcode::Swap16 => swap(self, 16),
            Opcode::Log0 => log::<0, _>(self),
            Opcode::Log1 => log::<1, _>(self),
            Opcode::Log2 => log::<2, _>(self),
            Opcode::Log3 => log::<3, _>(self),
            Opcode::Log4 => log::<4, _>(self),
            Opcode::Create => create(self),
            Opcode::Call => call(self),
            Opcode::CallCode => call_code(self),
            Opcode::Return => return_(self),
            Opcode::DelegateCall => delegate_call(self),
            Opcode::Create2 => create2(self),
            Opcode::StaticCall => static_call(self),
            Opcode::Revert => revert(self),
            Opcode::Invalid => invalid(self),
            Opcode::SelfDestruct => self_destruct(self),
        }
    }

    #[allow(clippy::unused_self)]
    fn return_from_op(&mut self) -> OpResult {
        #[cfg(not(feature = "jumptable-tail-call"))]
        return Ok(());
        #[cfg(feature = "jumptable-tail-call")]
        return self.run();
    }
}

impl<'a, E> From<Interpreter<'a, E>> for StepResult
where
    E: ExecutionContextTrait,
{
    fn from(value: Interpreter<E>) -> Self {
        let stack = value
            .stack
            .as_slice()
            .iter()
            .copied()
            .map(Into::into)
            .collect();
        Self::new(
            value.exec_status.into(),
            EvmcStatusCode::EVMC_SUCCESS,
            value.revision,
            value.code_reader.pc() as u64,
            value.gas_left.as_u64() as i64,
            value.gas_refund,
            value.output,
            stack,
            value.memory.into_inner(),
            value.last_call_return_data,
        )
    }
}

impl<'a, E> From<&mut Interpreter<'a, E>> for ExecutionResult
where
    E: ExecutionContextTrait,
{
    fn from(value: &mut Interpreter<E>) -> Self {
        Self::new(
            value.exec_status.into(),
            value.gas_left.as_u64() as i64,
            value.gas_refund,
            #[cfg(not(feature = "custom-evmc"))]
            value.output.as_deref(),
            #[cfg(feature = "custom-evmc")]
            value.output.take(),
        )
    }
}

#[cfg(test)]
mod tests {
    use evmc_vm::{
        Address, ExecutionResult, MessageKind, Revision, StatusCode as EvmcStatusCode, Uint256,
    };
    use mockall::predicate;

    use crate::{
        interpreter::Interpreter,
        types::{
            u256, ExecStatus, FailStatus, Memory, MockExecutionContextTrait, MockExecutionMessage,
            Opcode, Stack,
        },
    };

    #[test]
    fn empty_code() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter =
            Interpreter::new(Revision::EVMC_FRONTIER, &message, &mut context, &[]);
        let result = interpreter.run();
        assert!(result.is_ok());
        assert_eq!(interpreter.exec_status, ExecStatus::Stopped);
        assert_eq!(interpreter.code_reader.pc(), 0);
        assert_eq!(interpreter.gas_left, MockExecutionMessage::DEFAULT_INIT_GAS);
    }

    #[test]
    fn pc_after_end() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter = Interpreter::new_steppable(
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Add as u8],
            1,
            0,
            Stack::new(&[]),
            Memory::new(Vec::new()),
            None,
            None,
        );
        let result = interpreter.run();
        assert!(result.is_ok());
        assert_eq!(interpreter.exec_status, ExecStatus::Stopped);
        assert_eq!(interpreter.code_reader.pc(), 1);
        assert_eq!(interpreter.gas_left, MockExecutionMessage::DEFAULT_INIT_GAS);
    }

    #[test]
    fn pc_on_data() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let result = Interpreter::new_steppable(
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Push1 as u8, 0x00],
            1,
            0,
            Stack::new(&[]),
            Memory::new(Vec::new()),
            None,
            None,
        )
        .run();
        assert!(result.is_err());
        let status = result.map(|_| ()).unwrap_err();
        assert_eq!(status, FailStatus::InvalidInstruction);
    }

    #[test]
    fn zero_steps() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter = Interpreter::new(
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Add as u8],
        );
        interpreter.steps = Some(0);
        let result = interpreter.run();
        assert!(result.is_ok());
        assert_eq!(interpreter.exec_status, ExecStatus::Running);
        assert_eq!(interpreter.code_reader.pc(), 0);
        assert_eq!(interpreter.gas_left, MockExecutionMessage::DEFAULT_INIT_GAS);
    }

    #[test]
    fn add_one_step() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter = Interpreter::new(
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Add as u8, Opcode::Add as u8],
        );
        interpreter.steps = Some(1);
        interpreter.stack = Stack::new(&[1u8.into(), 2u8.into()]);
        let result = interpreter.run();
        assert!(result.is_ok());
        assert_eq!(interpreter.exec_status, ExecStatus::Running);
        assert_eq!(interpreter.stack.as_slice(), [3u8.into()]);
        assert_eq!(
            interpreter.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS - 3
        );
    }

    #[test]
    fn add_single_op() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter = Interpreter::new(
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Add as u8],
        );
        interpreter.stack = Stack::new(&[1u8.into(), 2u8.into()]);
        let result = interpreter.run();
        assert!(result.is_ok());
        assert_eq!(interpreter.exec_status, ExecStatus::Stopped);
        assert_eq!(interpreter.stack.as_slice(), [3u8.into()]);
        assert_eq!(
            interpreter.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS - 3
        );
    }

    #[test]
    fn add_twice() {
        let mut context = MockExecutionContextTrait::new();
        let message = MockExecutionMessage::default().into();
        let mut interpreter = Interpreter::new(
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Add as u8, Opcode::Add as u8],
        );
        interpreter.stack = Stack::new(&[1u8.into(), 2u8.into(), 3u8.into()]);
        let result = interpreter.run();
        assert!(result.is_ok());
        assert_eq!(interpreter.exec_status, ExecStatus::Stopped);
        assert_eq!(interpreter.stack.as_slice(), [6u8.into()]);
        assert_eq!(
            interpreter.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS - 2 * 3
        );
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
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Add as u8],
        );
        interpreter.stack = Stack::new(&[1u8.into(), 2u8.into()]);
        let result = interpreter.run();
        assert!(result.is_err());
        let status = result.map(|_| ()).unwrap_err();
        assert_eq!(status, FailStatus::OutOfGas);
    }

    #[test]
    fn call() {
        // helpers to generate unique values; random values are not needed
        let mut unique_values = 1u8..;
        let mut next_value = || unique_values.next().unwrap();

        let memory = vec![next_value(), next_value(), next_value(), next_value()];
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
                    && call_message.code_address() == &Address::from(u256::ZERO)
                    && call_message.code().is_none()
            })
            .returning(move |_| {
                #[cfg(not(feature = "custom-evmc"))]
                return ExecutionResult::new(EvmcStatusCode::EVMC_SUCCESS, 0, 0, Some(&ret_data));
                #[cfg(feature = "custom-evmc")]
                return ExecutionResult::new(
                    EvmcStatusCode::EVMC_SUCCESS,
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

        let mut interpreter = Interpreter::new_steppable(
            Revision::EVMC_FRONTIER,
            &message,
            &mut context,
            &[Opcode::Call as u8],
            0,
            0,
            Stack::new(&stack),
            Memory::new(memory),
            None,
            None,
        );
        let result = interpreter.run();
        assert!(result.is_ok());
        assert_eq!(interpreter.exec_status, ExecStatus::Stopped);
        assert_eq!(interpreter.code_reader.pc(), 1);
        assert_eq!(
            interpreter.gas_left,
            MockExecutionMessage::DEFAULT_INIT_GAS - 700 - gas
        );
        assert_eq!(
            interpreter.last_call_return_data.as_deref(),
            Some(ret_data.as_slice())
        );
        assert_eq!(
            &interpreter.memory.into_inner()[ret_offset..ret_offset + ret_len],
            ret_data.as_slice()
        );
    }
}

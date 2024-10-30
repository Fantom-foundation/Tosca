use std::fmt::Debug;

use crate::{
    interpreter::{Interpreter, OpFn},
    types::CodeByteType,
    u256,
};

#[derive(Clone, PartialEq, Eq)]
pub struct OpFnData {
    func: Option<OpFn>,
    data: u256,
}

impl OpFnData {
    pub fn data(data: u256) -> Self {
        OpFnData { func: None, data }
    }

    pub fn skip_no_ops_iter(count: usize) -> impl Iterator<Item = Self> {
        std::iter::once(OpFnData {
            func: Some(Interpreter::SKIP_NO_OPS_FN),
            data: (count as u64).into(),
        })
        .chain(
            std::iter::repeat_with(move || OpFnData {
                func: Some(Interpreter::NO_OP_FN),
                data: u256::ZERO,
            })
            .take(count - 1),
        )
    }

    pub fn func(op: u8, data: u256) -> Self {
        OpFnData {
            func: Some(Interpreter::JUMPTABLE[op as usize]),
            data,
        }
    }

    pub fn jump_dest() -> Self {
        OpFnData {
            func: Some(Interpreter::JUMP_DEST_FN),
            data: u256::ZERO,
        }
    }

    pub fn code_byte_type(&self) -> CodeByteType {
        match self.func {
            None => CodeByteType::DataOrInvalid,
            Some(func) if func == Interpreter::JUMP_DEST_FN => CodeByteType::JumpDest,
            Some(_) => CodeByteType::Opcode,
        }
    }

    pub fn get_func(&self) -> Option<OpFn> {
        self.func
    }

    pub fn get_data(&self) -> u256 {
        self.data
    }
}

impl Debug for OpFnData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpFnData")
            .field("func", &self.func.map(|f| f as *const u8))
            .field("data", &self.data)
            .finish()
    }
}

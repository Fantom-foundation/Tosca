use std::fmt::Debug;

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
use crate::u256;
use crate::{
    interpreter::{
        self, OpFn, JUMPTABLE_NO_STEPCHECK_JUMPDEST, JUMPTABLE_NO_STEPCHECK_NO_JUMPDEST,
        JUMPTABLE_STEPCHECK_JUMPDEST, JUMPTABLE_STEPCHECK_NO_JUMPDEST,
    },
    types::CodeByteType,
    Opcode,
};

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
pub const OP_FN_DATA_SIZE: usize = 4;

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
#[derive(Debug, PartialEq, Eq)]
#[repr(u8)]
enum OpFnDataType {
    Opcode = 0,
    JumpDest = 1,
    DataOrInvalid = 2,
}

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
#[derive(Clone, PartialEq, Eq)]
#[repr(align(8))]
pub struct OpFnData {
    raw: *const (),
}

// SAFETY:
// OpFnData only stores function pointers or [u8; 4] so it is safe to share across threads;
unsafe impl<const STEP_CHECK: bool, const JUMPDEST: bool> Send for OpFnData<STEP_CHECK, JUMPDEST> {}

// SAFETY:
// OpFnData only stores function pointers or [u8; 4] so it is safe to share across threads;
unsafe impl<const STEP_CHECK: bool, const JUMPDEST: bool> Sync for OpFnData<STEP_CHECK, JUMPDEST> {}

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
impl OpFnData {
    pub fn data(data: [u8; OP_FN_DATA_SIZE]) -> Self {
        // assumes native endian = little endian
        let mut raw = [0; 8];
        raw[..OP_FN_DATA_SIZE].copy_from_slice(&data);
        raw[7] = OpFnDataType::DataOrInvalid as u8;

        OpFnData {
            raw: usize::from_ne_bytes(raw) as *const (),
        }
    }

    pub fn skip_no_ops_iter(count: usize) -> impl Iterator<Item = Self> {
        std::iter::once(OpFnData {
            raw: interpreter::SKIP_NO_OPS_FN as *const (),
        })
        .chain(Some(OpFnData::data((count as u32).to_ne_bytes())))
        .chain(
            std::iter::repeat_with(move || OpFnData {
                raw: interpreter::NO_OP_FN as usize as *const (),
            })
            .take(count - 2),
        )
    }

    pub fn func<const JUMPDEST: bool>(op: u8) -> Self {
        let ptr_value = if JUMPDEST {
            interpreter::JUMPTABLE[op as usize]
        } else {
            interpreter::JUMPTABLE_SKIP_JUMPDEST[op as usize]
        };
        OpFnData {
            raw: ptr_value as *const (),
        }
    }

    pub fn jump_dest() -> Self {
        let mut ptr_value = interpreter::JUMP_DEST_FN as usize;
        ptr_value |= 0x0100000000000000; // OpFnDataType::JumpDest
        OpFnData {
            raw: ptr_value as *const (),
        }
    }

    pub fn code_byte_type(&self) -> CodeByteType {
        match (self.raw as usize).to_ne_bytes()[7] {
            t if t == OpFnDataType::Opcode as u8 => CodeByteType::Opcode,
            t if t == OpFnDataType::JumpDest as u8 => CodeByteType::JumpDest,
            _ => CodeByteType::DataOrInvalid,
        }
    }

    pub fn get_func(&self) -> Option<OpFn> {
        if (self.raw as usize).to_ne_bytes()[7] == OpFnDataType::DataOrInvalid as u8 {
            None
        } else {
            let mut ptr_value = self.raw as usize;
            ptr_value &= 0x0000ffffffffffff;
            let ptr = ptr_value as *const ();
            // SAFETY:
            // During code analysis self.raw was created from a function pointer. The highest bit
            // was used for marking it as such, but was masked out. As long as only the lower 6
            // bytes are used for pointers, the value is the same as before the conversion.
            Some(unsafe { std::mem::transmute::<*const (), OpFn>(ptr) })
        }
    }

    pub fn get_data(&self) -> [u8; OP_FN_DATA_SIZE] {
        // SAFETY:
        // A pointer to an 8 byte array can be safely cast to a pointer to an 4 byte and then read
        // as such, because the alignment is the same and all reads are in bounds.
        unsafe { std::ptr::read(&self.raw as *const _ as *const [u8; OP_FN_DATA_SIZE]) }
    }
}

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
impl Debug for OpFnData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpFnData").field("raw", &self.raw).finish()
    }
}

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
#[derive(Clone, PartialEq, Eq)]
pub struct OpFnData<const STEP_CHECK: bool, const JUMPDEST: bool> {
    func: Option<OpFn<STEP_CHECK, JUMPDEST>>,
    data: u256,
}

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
impl<const STEP_CHECK: bool, const JUMPDEST: bool> OpFnData<STEP_CHECK, JUMPDEST> {
    pub fn data(data: u256) -> Self {
        OpFnData { func: None, data }
    }

    pub fn skip_no_ops_iter(count: usize) -> impl Iterator<Item = Self> {
        let skip_no_ops = Self::func(Opcode::SkipNoOps as u8, (count as u64).into());
        let gen_no_ops = move || Self::func(Opcode::NoOp as u8, u256::ZERO);
        std::iter::once(skip_no_ops).chain(std::iter::repeat_with(gen_no_ops).take(count - 1))
    }

    fn jumptable_lookup(op: u8) -> OpFn<STEP_CHECK, JUMPDEST> {
        unsafe {
            match (STEP_CHECK, JUMPDEST) {
                (true, true) => std::mem::transmute(JUMPTABLE_STEPCHECK_JUMPDEST[op as usize]),
                (true, false) => std::mem::transmute(JUMPTABLE_STEPCHECK_NO_JUMPDEST[op as usize]),
                (false, true) => std::mem::transmute(JUMPTABLE_NO_STEPCHECK_JUMPDEST[op as usize]),
                (false, false) => {
                    std::mem::transmute(JUMPTABLE_NO_STEPCHECK_NO_JUMPDEST[op as usize])
                }
            }
        }
    }

    pub fn func(op: u8, data: u256) -> Self {
        OpFnData {
            func: Some(Self::jumptable_lookup(op)),
            data,
        }
    }

    pub fn jump_dest() -> Self {
        Self::func(Opcode::JumpDest as u8, u256::ZERO)
    }

    pub fn code_byte_type(&self) -> CodeByteType {
        match self.func {
            None => CodeByteType::DataOrInvalid,
            Some(func) if func == Self::jumptable_lookup(Opcode::JumpDest as u8) => {
                CodeByteType::JumpDest
            }
            Some(_) => CodeByteType::Opcode,
        }
    }

    pub fn get_func(&self) -> Option<OpFn<STEP_CHECK, JUMPDEST>> {
        self.func
    }

    pub fn get_data(&self) -> u256 {
        self.data
    }
}

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
impl<const STEP_CHECK: bool, const JUMPDEST: bool> Debug for OpFnData<STEP_CHECK, JUMPDEST> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpFnData")
            .field("func", &self.func.map(|f| f as *const u8))
            .field("data", &self.data)
            .finish()
    }
}

use std::fmt::Debug;

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
use crate::types::u256;
use crate::{
    interpreter::{GenericJumptable, OpFn},
    types::{CodeByteType, Opcode},
    utils::GetGenericStatic,
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
pub struct OpFnData<const STEPPABLE: bool> {
    raw: *const (),
}

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
// SAFETY:
// OpFnData only stores function pointers or [u8; 4] so it is safe to share across threads;
unsafe impl<const STEPPABLE: bool> Send for OpFnData<STEPPABLE> {}

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
// SAFETY:
// OpFnData only stores function pointers or [u8; 4] so it is safe to share across threads;
unsafe impl<const STEPPABLE: bool> Sync for OpFnData<STEPPABLE> {}

#[cfg(all(
    not(feature = "fn-ptr-conversion-expanded-dispatch"),
    feature = "fn-ptr-conversion-inline-dispatch"
))]
impl<const STEPPABLE: bool> OpFnData<STEPPABLE> {
    pub fn data(data: [u8; OP_FN_DATA_SIZE]) -> Self {
        // assumes native endian = little endian
        let mut raw = [0; 8];
        raw[..OP_FN_DATA_SIZE].copy_from_slice(&data);
        raw[7] = OpFnDataType::DataOrInvalid as u8;

        Self {
            raw: usize::from_ne_bytes(raw) as *const (),
        }
    }

    pub fn skip_no_ops_iter(count: usize) -> impl Iterator<Item = Self> {
        let skip_no_ops = Self::func(Opcode::SkipNoOps as u8);
        let count_data = Self::data((count as u32).to_ne_bytes());
        let gen_no_ops = move || Self::func(Opcode::NoOp as u8);
        std::iter::once(skip_no_ops)
            .chain(std::iter::once(count_data))
            .chain(std::iter::repeat_with(gen_no_ops).take(count - 2))
    }

    pub fn func(op: u8) -> Self {
        OpFnData {
            raw: GenericJumptable::get::<STEPPABLE>()[op as usize] as *const (),
        }
    }

    pub fn jump_dest() -> Self {
        let mut ptr_value =
            GenericJumptable::get::<STEPPABLE>()[Opcode::JumpDest as u8 as usize] as usize;
        ptr_value |= 0x0100000000000000; // OpFnDataType::JumpDest
        Self {
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

    pub fn get_func(&self) -> Option<OpFn<STEPPABLE>> {
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
            Some(unsafe { std::mem::transmute::<*const (), OpFn<STEPPABLE>>(ptr) })
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
impl<const STEPPABLE: bool> Debug for OpFnData<STEPPABLE> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpFnData").field("raw", &self.raw).finish()
    }
}

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
#[derive(Clone, PartialEq, Eq)]
pub struct OpFnData<const STEPPABLE: bool> {
    func: Option<OpFn<STEPPABLE>>,
    data: u256,
}

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
impl<const STEPPABLE: bool> OpFnData<STEPPABLE> {
    pub fn data(data: u256) -> Self {
        Self { func: None, data }
    }

    pub fn skip_no_ops_iter(count: usize) -> impl Iterator<Item = Self> {
        let skip_no_ops = Self::func(Opcode::SkipNoOps as u8, (count as u64).into());
        let gen_no_ops = move || Self::func(Opcode::NoOp as u8, u256::ZERO);
        std::iter::once(skip_no_ops).chain(std::iter::repeat_with(gen_no_ops).take(count - 1))
    }

    pub fn func(op: u8, data: u256) -> Self {
        Self {
            func: Some(GenericJumptable::get()[op as usize]),
            data,
        }
    }

    pub fn jump_dest() -> Self {
        Self::func(Opcode::JumpDest as u8, u256::ZERO)
    }

    pub fn code_byte_type(&self) -> CodeByteType {
        match self.func {
            None => CodeByteType::DataOrInvalid,
            Some(func) if func == GenericJumptable::get()[Opcode::JumpDest as u8 as usize] => {
                CodeByteType::JumpDest
            }
            Some(_) => CodeByteType::Opcode,
        }
    }

    pub fn get_func(&self) -> Option<OpFn<STEPPABLE>> {
        self.func
    }

    pub fn get_data(&self) -> u256 {
        self.data
    }
}

#[cfg(feature = "fn-ptr-conversion-expanded-dispatch")]
impl<const STEPPABLE: bool> Debug for OpFnData<STEPPABLE> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpFnData")
            .field("func", &self.func.map(|f| f as *const u8))
            .field("data", &self.data)
            .finish()
    }
}

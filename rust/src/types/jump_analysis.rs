#[cfg(feature = "opcode-fn-ptr-conversion")]
use std::cmp::min;
#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
use std::rc::Rc;
#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache")))]
use std::sync::Arc;
use std::{fmt::Debug, ops::Deref};

#[cfg(feature = "jump-cache")]
use nohash_hasher::BuildNoHashHasher;

#[cfg(feature = "opcode-fn-ptr-conversion")]
use crate::interpreter::Interpreter;
#[cfg(feature = "jump-cache")]
use crate::types::Cache;
#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
use crate::types::LocalKeyExt;
use crate::{
    interpreter::OpFn,
    types::{code_byte_type, u256, CodeByteType},
};

/// This type represents a hash value in form of a u256.
/// Because it is already a hash value there is no need to hash it again when implementing Hash.
#[cfg(feature = "jump-cache")]
#[allow(non_camel_case_types)]
#[derive(Debug, PartialEq, Eq)]
struct u256Hash(u256);

#[cfg(feature = "jump-cache")]
impl std::hash::Hash for u256Hash {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        state.write_u64(self.0.into_u64_with_overflow().0);
    }
}

#[cfg(not(feature = "jump-cache"))]
type AnalysisContainer<T> = Box<T>;
#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache")))]
type AnalysisContainer<T> = Arc<T>;
#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
type AnalysisContainer<T> = Rc<T>;

#[cfg(not(feature = "opcode-fn-ptr-conversion"))]
pub type AnalysisItem = CodeByteType;
#[cfg(feature = "opcode-fn-ptr-conversion")]
pub type AnalysisItem = OpFnData;

#[cfg(feature = "opcode-fn-ptr-conversion")]
#[derive(Clone, PartialEq, Eq)]
pub struct OpFnData {
    func: Option<OpFn>,
    data: [u8; 8],
}

#[cfg(feature = "opcode-fn-ptr-conversion")]
impl OpFnData {
    fn data(data: [u8; 8]) -> Self {
        OpFnData { func: None, data }
    }

    fn skip_no_ops_iter(count: usize) -> impl Iterator<Item = Self> {
        std::iter::once(OpFnData {
            func: Some(Interpreter::SKIP_NO_OPS_FN),
            data: ((count - 1) as u64).to_ne_bytes(),
        })
        .chain(std::iter::repeat_with(move || OpFnData {
            func: Some(Interpreter::NO_OP_FN),
            data: [0; 8],
        }))
        .take(count - 1)
    }

    fn func(func: OpFn, data: [u8; 8]) -> Self {
        OpFnData {
            func: Some(func),
            data,
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

    pub fn get_data(&self) -> [u8; 8] {
        self.data
    }
}

#[cfg(feature = "opcode-fn-ptr-conversion")]
impl Debug for OpFnData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpFnData")
            .field("func", &self.func.map(|f| f as *const u8))
            .field("data", &self.data)
            .finish()
    }
}

#[derive(Debug, Clone)]
pub struct JumpAnalysis(AnalysisContainer<[AnalysisItem]>);

impl Deref for JumpAnalysis {
    type Target = [AnalysisItem];

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl JumpAnalysis {
    #[allow(unused_variables)]
    pub fn new(code: &[u8], code_hash: Option<u256>) -> Self {
        #[cfg(feature = "jump-cache")]
        match code_hash {
            Some(code_hash) if code_hash != u256::ZERO => {
                JUMP_CACHE.get_or_insert(u256Hash(code_hash), || {
                    JumpAnalysis(AnalysisContainer::from(
                        compute_code_byte_types(code).as_slice(),
                    ))
                })
            }
            _ => JumpAnalysis(AnalysisContainer::from(
                compute_code_byte_types(code).as_slice(),
            )),
        }
        #[cfg(not(feature = "jump-cache"))]
        JumpAnalysis(compute_code_byte_types(code).into_boxed_slice())
    }
}

#[cfg(feature = "jump-cache")]
const CACHE_SIZE: usize = 1 << 16; // value taken from evmzero

#[cfg(feature = "jump-cache")]
type JumpCache = Cache<CACHE_SIZE, u256Hash, JumpAnalysis, BuildNoHashHasher<u64>>;

#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache")))]
static JUMP_CACHE: JumpCache = JumpCache::new();

#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
thread_local! {
    static JUMP_CACHE: JumpCache = JumpCache::new();
}

#[cfg(not(feature = "opcode-fn-ptr-conversion"))]
fn compute_code_byte_types(code: &[u8]) -> Vec<AnalysisItem> {
    let mut code_byte_types = vec![CodeByteType::DataOrInvalid; code.len()];

    let mut pc = 0;
    while let Some(op) = code.get(pc).copied() {
        let (code_byte_type, inc) = code_byte_type(op);
        code_byte_types[pc] = code_byte_type;
        pc += inc;
    }

    code_byte_types
}
#[cfg(feature = "opcode-fn-ptr-conversion")]
fn compute_code_byte_types(code: &[u8]) -> Vec<AnalysisItem> {
    let mut code_byte_types = Vec::with_capacity(code.len());

    let mut pc = 0;
    let mut no_ops = 0;
    while let Some(op) = code.get(pc).copied() {
        let (code_byte_type, mut inc) = code_byte_type(op);
        pc += 1;
        match code_byte_type {
            CodeByteType::JumpDest => {
                code_byte_types.extend(OpFnData::skip_no_ops_iter(no_ops));
                no_ops = 0;
                code_byte_types.push(OpFnData::func(Interpreter::JUMPTABLE[op as usize], [0; 8]));
            }
            CodeByteType::Opcode => {
                inc -= 1;

                let capped_inc = min(inc, 8);
                let mut data = [0; 8];
                data[..capped_inc].copy_from_slice(&code[pc..pc + capped_inc]);
                code_byte_types.push(OpFnData::func(Interpreter::JUMPTABLE[op as usize], data));
                inc -= capped_inc;
                pc += capped_inc;

                while inc > 0 {
                    let capped_inc = min(inc, 8);
                    let mut data = [0; 8];
                    data[..capped_inc].copy_from_slice(&code[pc..pc + capped_inc]);
                    code_byte_types.push(OpFnData::data(data));
                    no_ops += 1;
                    inc -= capped_inc;
                    pc += capped_inc;
                }
            }
            CodeByteType::DataOrInvalid => {
                // This should only be the case if an invalid opcode was not preceded by a push.
                // In this case we don't care what the data contains.
                code_byte_types.push(OpFnData::data([0; 8]));
            }
        };
    }

    code_byte_types
}

#[cfg(feature = "xx")]
#[cfg(test)]
mod tests {
    use crate::types::{jump_analysis::compute_code_byte_types, CodeByteType, Opcode};

    #[test]
    fn compute_code_byte_types_single_byte() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Add as u8]),
            [CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push2 as u8]),
            [CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8]),
            [CodeByteType::JumpDest]
        );
        assert_eq!(
            *compute_code_byte_types(&[0xc0]),
            [CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn compute_byte_types_jumpdest() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8, Opcode::Add as u8]),
            [CodeByteType::JumpDest, CodeByteType::Opcode]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::JumpDest as u8, 0xc0]),
            [CodeByteType::JumpDest, CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn compute_code_byte_types_push_with_data() {
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push1 as u8, Opcode::Add as u8, Opcode::Add as u8]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[Opcode::Push1 as u8, Opcode::Add as u8, 0xc0]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                0xc0,
                Opcode::Add as u8
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            *compute_code_byte_types(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                0xc0
            ]),
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
    }
}

use std::ops::Deref;
#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
use std::rc::Rc;
#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache")))]
use std::sync::Arc;

#[cfg(feature = "jump-cache")]
use nohash_hasher::BuildNoHashHasher;

#[cfg(feature = "jump-cache")]
use crate::types::Cache;
#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
use crate::types::LocalKeyExt;
use crate::types::{code_byte_type, u256, CodeByteType};

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

#[derive(Debug, Clone)]
pub struct JumpAnalysis(AnalysisContainer<[CodeByteType]>);

impl Deref for JumpAnalysis {
    type Target = [CodeByteType];

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

fn compute_code_byte_types(code: &[u8]) -> Vec<CodeByteType> {
    let mut code_byte_types = vec![CodeByteType::DataOrInvalid; code.len()];

    let mut pc = 0;
    while pc < code.len() {
        let (code_byte_type, inc) = code_byte_type(code[pc]);
        code_byte_types[pc] = code_byte_type;
        pc += inc;
    }

    code_byte_types
}

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

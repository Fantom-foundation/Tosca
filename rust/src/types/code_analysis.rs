#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
use std::rc::Rc;
#[cfg(all(feature = "code-analysis-cache", not(feature = "thread-local-cache")))]
use std::sync::Arc;

#[cfg(feature = "code-analysis-cache")]
use nohash_hasher::BuildNoHashHasher;

#[cfg(feature = "code-analysis-cache")]
use crate::types::Cache;
#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
use crate::types::LocalKeyExt;
use crate::types::{code_byte_type, u256, CodeByteType};

/// This type represents a hash value in form of a u256.
/// Because it is already a hash value there is no need to hash it again when implementing Hash.
#[cfg(feature = "code-analysis-cache")]
#[allow(non_camel_case_types)]
#[derive(Debug, PartialEq, Eq)]
struct u256Hash(u256);

#[cfg(feature = "code-analysis-cache")]
impl std::hash::Hash for u256Hash {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        state.write_u64(self.0.into_u64_with_overflow().0);
    }
}

#[cfg(not(feature = "code-analysis-cache"))]
pub type AnalysisContainer<T> = T;
#[cfg(all(feature = "code-analysis-cache", not(feature = "thread-local-cache")))]
pub type AnalysisContainer<T> = Arc<T>;
#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
pub type AnalysisContainer<T> = Rc<T>;

pub type AnalysisItem = CodeByteType;

#[cfg(feature = "code-analysis-cache")]
const CACHE_SIZE: usize = 1 << 16; // value taken from evmzero

#[cfg(feature = "code-analysis-cache")]
type CodeAnalysisCache =
    Cache<CACHE_SIZE, u256Hash, AnalysisContainer<CodeAnalysis>, BuildNoHashHasher<u64>>;

#[cfg(all(feature = "code-analysis-cache", not(feature = "thread-local-cache")))]
static CODE_ANALYSIS_CACHE: CodeAnalysisCache = CodeAnalysisCache::new();

#[cfg(all(feature = "code-analysis-cache", feature = "thread-local-cache"))]
thread_local! {
    static CODE_ANALYSIS_CACHE: CodeAnalysisCache = CodeAnalysisCache::new();
}

#[derive(Debug)]
pub struct CodeAnalysis {
    pub analysis: Vec<AnalysisItem>,
}

impl CodeAnalysis {
    #[allow(unused_variables)]
    pub fn new(code: &[u8], code_hash: Option<u256>) -> AnalysisContainer<Self> {
        #[cfg(feature = "code-analysis-cache")]
        match code_hash {
            Some(code_hash) if code_hash != u256::ZERO => CODE_ANALYSIS_CACHE
                .get_or_insert(u256Hash(code_hash), || {
                    AnalysisContainer::new(Self::analyze_code(code))
                }),
            _ => AnalysisContainer::new(Self::analyze_code(code)),
        }
        #[cfg(not(feature = "code-analysis-cache"))]
        Self::analyze_code(code)
    }

    fn analyze_code(code: &[u8]) -> Self {
        let mut code_byte_types = vec![CodeByteType::DataOrInvalid; code.len()];

        let mut pc = 0;
        while let Some(op) = code.get(pc).copied() {
            let (code_byte_type, data) = code_byte_type(op);
            code_byte_types[pc] = code_byte_type;
            pc += 1 + data;
        }

        CodeAnalysis {
            analysis: code_byte_types,
        }
    }
}

#[cfg(test)]
mod tests {
    use crate::types::{CodeAnalysis, CodeByteType, Opcode};

    #[test]
    fn analyze_code_single_byte() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Add as u8]).analysis,
            [CodeByteType::Opcode]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push2 as u8]).analysis,
            [CodeByteType::Opcode]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8]).analysis,
            [CodeByteType::JumpDest]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[0xc0]).analysis,
            [CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn analyze_code_jumpdest() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, Opcode::Add as u8]).analysis,
            [CodeByteType::JumpDest, CodeByteType::Opcode]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::JumpDest as u8, 0xc0]).analysis,
            [CodeByteType::JumpDest, CodeByteType::DataOrInvalid]
        );
    }

    #[test]
    fn analyze_code_push_with_data() {
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[Opcode::Push1 as u8, Opcode::Add as u8, 0xc0]).analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push1 as u8,
                Opcode::Add as u8,
                0xc0,
                Opcode::Add as u8
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::Opcode,
            ]
        );
        assert_eq!(
            CodeAnalysis::analyze_code(&[
                Opcode::Push2 as u8,
                Opcode::Add as u8,
                Opcode::Add as u8,
                0xc0
            ])
            .analysis,
            [
                CodeByteType::Opcode,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
                CodeByteType::DataOrInvalid,
            ]
        );
    }
}

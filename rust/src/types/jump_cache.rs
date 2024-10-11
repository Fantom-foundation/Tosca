use std::ops::Deref;
#[cfg(feature = "jump-cache")]
use std::sync::Arc;

#[cfg(feature = "jump-cache")]
use nohash_hasher::BuildNoHashHasher;

#[cfg(feature = "jump-cache")]
use crate::types::Cache;
use crate::types::{u256, CodeByteType};

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

#[cfg(feature = "jump-cache")]
const CACHE_SIZE: usize = 1 << 16; // value taken from evmzero

#[cfg(not(feature = "jump-cache"))]
type AnalysisContainer<T> = Box<T>;
#[cfg(feature = "jump-cache")]
type AnalysisContainer<T> = Arc<T>;

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
    pub fn new<G>(code_hash: Option<u256>, gen: G) -> Self
    where
        G: FnOnce() -> Vec<CodeByteType>,
    {
        #[cfg(feature = "jump-cache")]
        match code_hash {
            Some(code_hash) if code_hash != u256::ZERO => JUMP_CACHE
                .get_or_insert(u256Hash(code_hash), || {
                    JumpAnalysis(AnalysisContainer::from(gen().as_slice()))
                }),
            _ => JumpAnalysis(AnalysisContainer::from(gen().as_slice())),
        }
        #[cfg(not(feature = "jump-cache"))]
        JumpAnalysis(gen().into_boxed_slice())
    }
}

#[cfg(feature = "jump-cache")]
type JumpCache = Cache<CACHE_SIZE, u256Hash, JumpAnalysis, BuildNoHashHasher<u64>>;

#[cfg(feature = "jump-cache")]
static JUMP_CACHE: JumpCache = JumpCache::new();

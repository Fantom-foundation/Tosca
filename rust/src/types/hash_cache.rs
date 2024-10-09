use sha3::{Digest, Keccak256};

#[cfg(feature = "hash-cache")]
use crate::types::Cache;
#[cfg(all(feature = "hash-cache", feature = "thread-local-cache"))]
use crate::types::LocalKeyExt;
use crate::u256;

#[cfg(feature = "hash-cache")]
const CACHE_SIZE: usize = 1024; // value taken from evmzero

#[cfg(feature = "hash-cache")]
pub type HashCache32 = Cache<CACHE_SIZE, [u8; 32], u256>;
#[cfg(feature = "hash-cache")]
pub type HashCache64 = Cache<CACHE_SIZE, [u8; 64], u256>;

#[cfg(all(feature = "hash-cache", not(feature = "thread-local-cache")))]
static HASH_CACHE_32: HashCache32 = HashCache32::new();
#[cfg(all(feature = "hash-cache", not(feature = "thread-local-cache")))]
static HASH_CACHE_64: HashCache64 = HashCache64::new();

#[cfg(all(feature = "hash-cache", feature = "thread-local-cache"))]
thread_local! {
    static HASH_CACHE_32: HashCache32 = HashCache32::new();
    static HASH_CACHE_64: HashCache64 = HashCache64::new();
}

fn sha3(data: &[u8]) -> u256 {
    let mut hasher = Keccak256::new();
    hasher.update(data);
    let mut bytes = [0; 32];
    hasher.finalize_into((&mut bytes).into());

    bytes.into()
}

pub fn hash(data: &[u8]) -> u256 {
    #[cfg(feature = "hash-cache")]
    if data.len() == 32 {
        // SAFETY:
        // data has length 32 so it is safe to cast it to &[u8; 32].
        let data = unsafe { &*(data.as_ptr() as *const [u8; 32]) };
        HASH_CACHE_32.get_or_insert_ref(data, || sha3(data))
    } else if data.len() == 64 {
        // SAFETY:
        // data has length 64 so it is safe to cast it to &[u8; 64].
        let data = unsafe { &*(data.as_ptr() as *const [u8; 64]) };
        HASH_CACHE_64.get_or_insert_ref(data, || sha3(data))
    } else {
        sha3(data)
    }
    #[cfg(not(feature = "hash-cache"))]
    sha3(data)
}

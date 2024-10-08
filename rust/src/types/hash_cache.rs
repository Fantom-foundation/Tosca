#[cfg(feature = "hash-cache")]
use std::{
    num::NonZeroUsize,
    sync::{LazyLock, Mutex},
};

#[cfg(feature = "hash-cache")]
use lru::LruCache;
use sha3::{Digest, Keccak256};

use crate::u256;

#[cfg(feature = "hash-cache")]
const CACHE_SIZE: NonZeroUsize = unsafe { NonZeroUsize::new_unchecked(1024) };

// Mutex<LruCache<...>> is faster that quick_cache::Cache<...>
#[cfg(feature = "hash-cache")]
static HASH_CACHE_32: LazyLock<Mutex<LruCache<[u8; 32], u256>>> =
    LazyLock::new(|| Mutex::new(LruCache::new(CACHE_SIZE)));
#[cfg(feature = "hash-cache")]
static HASH_CACHE_64: LazyLock<Mutex<LruCache<[u8; 64], u256>>> =
    LazyLock::new(|| Mutex::new(LruCache::new(CACHE_SIZE)));

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
        *HASH_CACHE_32
            .lock()
            .unwrap()
            .get_or_insert_ref(data, || sha3(data))
    } else if data.len() == 64 {
        // SAFETY:
        // data has length 64 so it is safe to cast it to &[u8; 64].
        let data = unsafe { &*(data.as_ptr() as *const [u8; 64]) };
        *HASH_CACHE_64
            .lock()
            .unwrap()
            .get_or_insert_ref(data, || sha3(data))
    } else {
        sha3(data)
    }
    #[cfg(not(feature = "hash-cache"))]
    sha3(data)
}

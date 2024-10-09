#[cfg(all(feature = "jump-cache", feature = "thread-local-cache"))]
use std::cell::RefCell;
#[cfg(feature = "hash-cache")]
use std::num::NonZeroUsize;
#[cfg(all(feature = "jump-cache", not(feature = "thread-local-cache")))]
use std::sync::{LazyLock, Mutex};

#[cfg(feature = "hash-cache")]
use lru::LruCache;
use sha3::{Digest, Keccak256};

use crate::u256;

#[cfg(feature = "hash-cache")]
const CACHE_SIZE: NonZeroUsize = unsafe { NonZeroUsize::new_unchecked(1024) };

// Mutex<LruCache<...>> is faster that quick_cache::Cache<...>
#[cfg(all(feature = "hash-cache", not(feature = "thread-local-cache")))]
static HASH_CACHE_32: LazyLock<Mutex<LruCache<[u8; 32], u256>>> =
    LazyLock::new(|| Mutex::new(LruCache::new(CACHE_SIZE)));
#[cfg(all(feature = "hash-cache", not(feature = "thread-local-cache")))]
static HASH_CACHE_64: LazyLock<Mutex<LruCache<[u8; 64], u256>>> =
    LazyLock::new(|| Mutex::new(LruCache::new(CACHE_SIZE)));

#[cfg(all(feature = "hash-cache", feature = "thread-local-cache"))]
thread_local! {
static HASH_CACHE_32: RefCell<LruCache<[u8; 32], u256>> =
    RefCell::new(LruCache::new(CACHE_SIZE));
static HASH_CACHE_64: RefCell<LruCache<[u8; 64], u256>> =
    RefCell::new(LruCache::new(CACHE_SIZE));
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
        #[cfg(all(feature = "hash-cache", not(feature = "thread-local-cache")))]
        return *HASH_CACHE_32
            .lock()
            .unwrap()
            .get_or_insert_ref(data, || sha3(data));

        #[cfg(all(feature = "hash-cache", feature = "thread-local-cache"))]
        return HASH_CACHE_32
            .with_borrow_mut(|cache| *cache.get_or_insert_ref(data, || sha3(data)));
    } else if data.len() == 64 {
        // SAFETY:
        // data has length 64 so it is safe to cast it to &[u8; 64].
        let data = unsafe { &*(data.as_ptr() as *const [u8; 64]) };
        #[cfg(all(feature = "hash-cache", not(feature = "thread-local-cache")))]
        return *HASH_CACHE_64
            .lock()
            .unwrap()
            .get_or_insert_ref(data, || sha3(data));

        #[cfg(all(feature = "hash-cache", feature = "thread-local-cache"))]
        return HASH_CACHE_64
            .with_borrow_mut(|cache| *cache.get_or_insert_ref(data, || sha3(data)));
    } else {
        sha3(data)
    }
    #[cfg(not(feature = "hash-cache"))]
    sha3(data)
}

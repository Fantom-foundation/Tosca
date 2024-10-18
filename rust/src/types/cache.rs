#[cfg(not(feature = "thread-local-cache"))]
use std::sync::{LazyLock, Mutex};
#[cfg(feature = "thread-local-cache")]
use std::{cell::RefCell, thread::LocalKey};
use std::{
    hash::{BuildHasher, Hash},
    num::NonZeroUsize,
};

use lru::{DefaultHasher, LruCache};

pub struct Cache<const S: usize, K, V, H = DefaultHasher>(
    // Mutex<LruCache<...>> is faster that quick_cache::Cache<...>
    #[cfg(not(feature = "thread-local-cache"))] LazyLock<Mutex<LruCache<K, V, H>>>,
    #[cfg(feature = "thread-local-cache")] RefCell<LruCache<K, V, H>>,
)
where
    K: Hash + Eq;

impl<const S: usize, K, V, H> Cache<S, K, V, H>
where
    K: Hash + Eq,
    H: BuildHasher + Default,
{
    #[cfg(not(feature = "thread-local-cache"))]
    pub const fn new() -> Self {
        Self(LazyLock::new(|| {
            Mutex::new(LruCache::with_hasher(
                NonZeroUsize::new(S).unwrap(),
                H::default(),
            ))
        }))
    }
    #[cfg(feature = "thread-local-cache")]
    pub fn new() -> Self {
        Self(RefCell::new(LruCache::with_hasher(
            NonZeroUsize::new(S).unwrap(),
            H::default(),
        )))
    }

    #[cfg(feature = "jump-cache")]
    pub fn get_or_insert(&self, key: K, f: impl FnOnce() -> V) -> V
    where
        V: Clone,
    {
        #[cfg(not(feature = "thread-local-cache"))]
        return self.0.lock().unwrap().get_or_insert(key, f).clone();
        #[cfg(feature = "thread-local-cache")]
        return self.0.borrow_mut().get_or_insert(key, f).clone();
    }

    #[cfg(feature = "hash-cache")]
    pub fn get_or_insert_ref<Q>(&self, key: &Q, f: impl FnOnce() -> V) -> V
    where
        K: std::borrow::Borrow<Q>,
        Q: ToOwned<Owned = K> + Hash + Eq,
        V: Clone,
    {
        #[cfg(not(feature = "thread-local-cache"))]
        return self.0.lock().unwrap().get_or_insert_ref(key, f).clone();
        #[cfg(feature = "thread-local-cache")]
        return self.0.borrow_mut().get_or_insert_ref(key, f).clone();
    }
}

#[cfg(feature = "thread-local-cache")]
pub trait LocalKeyExt<const S: usize, K, V, H> {
    #[cfg(feature = "jump-cache")]
    fn get_or_insert(&'static self, key: K, f: impl FnOnce() -> V) -> V
    where
        V: Clone;

    #[cfg(feature = "hash-cache")]
    fn get_or_insert_ref<Q>(&'static self, key: &Q, f: impl FnOnce() -> V) -> V
    where
        K: std::borrow::Borrow<Q>,
        Q: ToOwned<Owned = K> + Hash + Eq,
        V: Clone;
}

#[cfg(feature = "thread-local-cache")]
impl<const S: usize, K, V, H> LocalKeyExt<S, K, V, H> for LocalKey<Cache<S, K, V, H>>
where
    K: Hash + Eq,
    H: BuildHasher + Default,
{
    #[cfg(feature = "jump-cache")]
    fn get_or_insert(&'static self, key: K, f: impl FnOnce() -> V) -> V
    where
        V: Clone,
    {
        self.with(|c| c.get_or_insert(key, f))
    }

    #[cfg(feature = "hash-cache")]
    fn get_or_insert_ref<Q>(&'static self, key: &Q, f: impl FnOnce() -> V) -> V
    where
        K: std::borrow::Borrow<Q>,
        Q: ToOwned<Owned = K> + Hash + Eq,
        V: Clone,
    {
        self.with(|c| c.get_or_insert_ref(key, f))
    }
}

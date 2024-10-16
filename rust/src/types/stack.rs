#[cfg(feature = "stack-array")]
use std::{cmp::min, mem::MaybeUninit};

use crate::types::{u256, FailStatus};

#[cfg(feature = "stack-array")]
#[derive(Debug)]
struct StackRaw {
    data: [MaybeUninit<u256>; 1024],
    len: usize,
}

#[cfg(feature = "stack-array")]
impl StackRaw {
    pub fn new(inner: &[u256]) -> Self {
        let len = min(inner.len(), 1024);
        let mut s = Self {
            data: [const { MaybeUninit::uninit() }; 1024],
            len,
        };
        // SAFETY:
        // &[T] and &[MaybeUninit<T>] have the same layout
        // With nightly rust this could be replaced by MaybeUninit::copy_from_slice
        s.data[..len].copy_from_slice(unsafe {
            std::mem::transmute::<&[u256], &[std::mem::MaybeUninit<u256>]>(inner)
        });
        s
    }

    pub fn as_slice(&self) -> &[u256] {
        // SAFETY:
        // &[T] and &[MaybeUninit<T>] have the same layout and self.data is initialized up to
        // self.len
        unsafe {
            std::mem::transmute::<&[std::mem::MaybeUninit<u256>], &[u256]>(&self.data[..self.len])
        }
    }

    pub fn as_mut_slice(&mut self) -> &mut [u256] {
        // SAFETY:
        // &[T] and &[MaybeUninit<T>] have the same layout and self.data is initialized up to
        // self.len
        unsafe {
            std::mem::transmute::<&mut [std::mem::MaybeUninit<u256>], &mut [u256]>(
                &mut self.data[..self.len],
            )
        }
    }

    pub fn len(&self) -> usize {
        self.len
    }

    /// # Safety
    /// self.len must be < 1024
    pub unsafe fn push(&mut self, value: u256) {
        self.data[self.len] = MaybeUninit::new(value);
        self.len += 1;
    }

    /// # Safety
    /// self.len must be >= N
    pub unsafe fn pop_array<const N: usize>(&mut self) -> [u256; N] {
        self.len -= N;
        let start = self.data.as_ptr() as *const u256;
        // SAFETY:
        // This does not wrap and the whole range from start to start + self.len is valid.
        let array_start = unsafe { start.add(self.len) };
        // SAFETY:
        // The invariant of this type is that the first `self.len` elements are initialized.
        // `self.len` just got decremented by N, which means now that the first `self.len + N`
        // elements are initialized. Therefore, it is safe to read N elements starting at index
        // `self.len` as an array of length N and type u256.
        unsafe { std::ptr::read(array_start as *const [u256; N]) }
    }
}

#[cfg(feature = "stack-array")]
#[derive(Debug)]
pub struct Stack(StackRaw);

#[cfg(not(feature = "stack-array"))]
#[derive(Debug)]
pub struct Stack(Vec<u256>);

impl Stack {
    pub fn new(inner: &[u256]) -> Self {
        #[cfg(feature = "stack-array")]
        return Self(StackRaw::new(inner));
        #[cfg(not(feature = "stack-array"))]
        return Self(Vec::from(inner));
    }

    pub fn as_slice(&self) -> &[u256] {
        self.0.as_slice()
    }

    pub fn len(&self) -> usize {
        self.0.len()
    }

    pub fn push(&mut self, value: impl Into<u256>) -> Result<(), FailStatus> {
        self.check_overflow_on_push()?;

        #[cfg(feature = "stack-array")]
        // SAFETY:
        // self.0.len < 1024
        unsafe {
            self.0.push(value.into());
        }
        #[cfg(not(feature = "stack-array"))]
        self.0.push(value.into());

        Ok(())
    }

    pub fn swap_with_top(&mut self, nth: usize) -> Result<(), FailStatus> {
        self.check_underflow(nth + 1)?;

        let len = self.0.len();
        self.0.as_mut_slice().swap(len - 1, len - 1 - nth);
        Ok(())
    }

    pub fn pop<const N: usize>(&mut self) -> Result<[u256; N], FailStatus> {
        self.check_underflow(N)?;

        #[cfg(feature = "stack-array")]
        {
            // SAFETY:
            // self.0.len >= N
            Ok(unsafe { self.0.pop_array() })
        }
        #[cfg(not(feature = "stack-array"))]
        {
            let new_len = self.0.len() - N;
            let mut array = [u256::ZERO; N];
            array.copy_from_slice(&self.0[new_len..]);
            self.0.truncate(new_len);
            Ok(array)
        }
    }

    pub fn nth(&self, nth: usize) -> Result<u256, FailStatus> {
        self.check_underflow(nth + 1)?;
        Ok(self.0.as_slice()[self.0.len() - nth - 1])
    }

    #[inline(always)]
    fn check_overflow_on_push(&self) -> Result<(), FailStatus> {
        if self.0.len() >= 1024 {
            return Err(FailStatus::StackOverflow);
        }
        Ok(())
    }

    #[inline(always)]
    fn check_underflow(&self, min_len: usize) -> Result<(), FailStatus> {
        if self.0.len() < min_len {
            return Err(FailStatus::StackUnderflow);
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use crate::types::{stack::Stack, u256, FailStatus};

    #[test]
    fn internals() {
        let stack = Stack::new(&[u256::ONE]);
        assert_eq!(stack.len(), 1);
        assert_eq!(stack.as_slice(), &[u256::ONE]);
    }

    #[test]
    fn push() {
        let mut stack = Stack::new(&[]);
        assert_eq!(stack.push(u256::MAX), Ok(()));
        assert_eq!(stack.as_slice(), [u256::MAX]);

        let mut stack = Stack::new(&[u256::ZERO; 1024]);
        assert_eq!(stack.push(u256::ZERO), Err(FailStatus::StackOverflow));
    }

    #[test]
    fn pop() {
        let mut stack = Stack::new(&[u256::MAX]);
        assert_eq!(stack.pop::<1>(), Ok([u256::MAX]));

        let mut stack = Stack::new(&[]);
        assert_eq!(stack.pop::<1>(), Err(FailStatus::StackUnderflow));

        let mut stack = Stack::new(&[u256::ONE, u256::MAX]);
        assert_eq!(stack.pop::<2>(), Ok([u256::ONE, u256::MAX]));

        let mut stack = Stack::new(&[u256::MAX]);
        assert_eq!(stack.pop::<2>(), Err(FailStatus::StackUnderflow));
    }

    #[test]
    fn nth() {
        let stack = Stack::new(&[u256::MAX, u256::ZERO]);
        assert_eq!(stack.nth(0), Ok(u256::ZERO));
        assert_eq!(stack.nth(1), Ok(u256::MAX));
        assert_eq!(stack.nth(2), Err(FailStatus::StackUnderflow));
    }

    #[test]
    fn swap_with_top() {
        let mut stack = Stack::new(&[u256::MAX, u256::ONE]);
        assert_eq!(stack.swap_with_top(0), Ok(()));
        assert_eq!(stack.as_slice(), &[u256::MAX, u256::ONE]);

        let mut stack = Stack::new(&[u256::MAX, u256::ONE]);
        assert_eq!(stack.swap_with_top(1), Ok(()));
        assert_eq!(stack.as_slice(), [u256::ONE, u256::MAX]);

        let mut stack = Stack::new(&[u256::MAX, u256::ONE]);
        assert_eq!(stack.swap_with_top(2), Err(FailStatus::StackUnderflow));
    }

    #[test]
    fn check_overflow_on_push() {
        let stack = Stack::new(&[u256::MAX; 1023]);
        assert_eq!(stack.check_overflow_on_push(), Ok(()));
        let stack = Stack::new(&[u256::MAX; 1024]);
        assert_eq!(
            stack.check_overflow_on_push(),
            Err(FailStatus::StackOverflow)
        );
    }

    #[test]
    fn check_underflow() {
        let stack = Stack::new(&[]);
        assert_eq!(stack.check_underflow(0), Ok(()));
        let stack = Stack::new(&[u256::ZERO]);
        assert_eq!(stack.check_underflow(1), Ok(()));
        assert_eq!(stack.check_underflow(2), Err(FailStatus::StackUnderflow));
    }
}

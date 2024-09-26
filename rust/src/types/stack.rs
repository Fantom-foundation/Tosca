#[cfg(feature = "stack-array")]
use std::{cmp::min, hint::assert_unchecked, mem::MaybeUninit};

use evmc_vm::StatusCode;

use crate::types::u256;

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

    pub fn push(&mut self, value: u256) {
        if self.len < 1024 {
            self.data[self.len] = MaybeUninit::new(value);
            self.len += 1;
        }
    }

    pub fn pop_array<const N: usize>(&mut self) -> Option<[u256; N]> {
        if self.len < N {
            return None;
        }
        self.len -= N;
        // SAFETY:
        // The only methods modifying self.len are new, push and this method. In new self.len is set
        // to a value <= 1024. In push the length is only incremented if it was < 1024 beforehand.
        // An here we have decremented self.len by N. This means self.len + N is always <= 1024;
        unsafe {
            assert_unchecked(self.len + N <= 1024);
        }
        // SAFETY:
        // Now that we subtracted N from self.len all elements until index self.len + N are
        // initialized. This means that an array of length N starting at index self.len is fully
        // initialized.
        unsafe {
            let start = (&self.data[self.len]) as *const MaybeUninit<u256> as *const u256;
            Some(std::ptr::read(start as *const [u256; N]))
        }
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

    pub fn push(&mut self, value: impl Into<u256>) -> Result<(), StatusCode> {
        self.check_overflow_on_push()?;
        self.0.push(value.into());
        Ok(())
    }

    pub fn swap_with_top(&mut self, nth: usize) -> Result<(), StatusCode> {
        self.check_underflow(nth + 1)?;

        let len = self.0.len();
        self.0.as_mut_slice().swap(len - 1, len - 1 - nth);
        Ok(())
    }

    pub fn pop<const N: usize>(&mut self) -> Result<[u256; N], StatusCode> {
        self.check_underflow(N)?;

        #[cfg(feature = "stack-array")]
        {
            Ok(self.0.pop_array().unwrap())
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

    pub fn nth(&self, nth: usize) -> Result<u256, StatusCode> {
        self.check_underflow(nth + 1)?;
        Ok(self.0.as_slice()[self.0.len() - nth - 1])
    }

    #[inline(always)]
    fn check_overflow_on_push(&self) -> Result<(), StatusCode> {
        if self.0.len() >= 1024 {
            return Err(StatusCode::EVMC_STACK_OVERFLOW);
        }
        Ok(())
    }

    #[inline(always)]
    fn check_underflow(&self, min_len: usize) -> Result<(), StatusCode> {
        if self.0.len() < min_len {
            return Err(StatusCode::EVMC_STACK_UNDERFLOW);
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use evmc_vm::StatusCode;

    use crate::types::{stack::Stack, u256};

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
        assert_eq!(stack.push(u256::ZERO), Err(StatusCode::EVMC_STACK_OVERFLOW));
    }

    #[test]
    fn pop() {
        let mut stack = Stack::new(&[u256::MAX]);
        assert_eq!(stack.pop::<1>(), Ok([u256::MAX]));

        let mut stack = Stack::new(&[]);
        assert_eq!(stack.pop::<1>(), Err(StatusCode::EVMC_STACK_UNDERFLOW));

        let mut stack = Stack::new(&[u256::ONE, u256::MAX]);
        assert_eq!(stack.pop::<2>(), Ok([u256::ONE, u256::MAX]));

        let mut stack = Stack::new(&[u256::MAX]);
        assert_eq!(stack.pop::<2>(), Err(StatusCode::EVMC_STACK_UNDERFLOW));
    }

    #[test]
    fn nth() {
        let stack = Stack::new(&[u256::MAX, u256::ZERO]);
        assert_eq!(stack.nth(0), Ok(u256::ZERO));
        assert_eq!(stack.nth(1), Ok(u256::MAX));
        assert_eq!(stack.nth(2), Err(StatusCode::EVMC_STACK_UNDERFLOW));
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
        assert_eq!(
            stack.swap_with_top(2),
            Err(StatusCode::EVMC_STACK_UNDERFLOW)
        );
    }

    #[test]
    fn check_overflow_on_push() {
        let stack = Stack::new(&[u256::MAX; 1023]);
        assert_eq!(stack.check_overflow_on_push(), Ok(()));
        let stack = Stack::new(&[u256::MAX; 1024]);
        assert_eq!(
            stack.check_overflow_on_push(),
            Err(StatusCode::EVMC_STACK_OVERFLOW)
        );
    }

    #[test]
    fn check_underflow() {
        let stack = Stack::new(&[]);
        assert_eq!(stack.check_underflow(0), Ok(()));
        let stack = Stack::new(&[u256::ZERO]);
        assert_eq!(stack.check_underflow(1), Ok(()));
        assert_eq!(
            stack.check_underflow(2),
            Err(StatusCode::EVMC_STACK_UNDERFLOW)
        );
    }
}

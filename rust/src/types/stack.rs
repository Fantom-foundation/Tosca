use std::{cmp::min, sync::Mutex};

use crate::types::{u256, FailStatus};

static REUSABLE_STACK: Mutex<Option<Vec<u256>>> = Mutex::new(None);

#[derive(Debug)]
pub struct Stack(Vec<u256>);

impl Drop for Stack {
    fn drop(&mut self) {
        let mut stack = Vec::new();
        std::mem::swap(&mut stack, &mut self.0);
        *REUSABLE_STACK.lock().unwrap() = Some(stack);
    }
}

impl Stack {
    const CAPACITY: usize = 1024;

    #[inline(never)]
    pub fn new(inner: &[u256]) -> Self {
        let len = min(inner.len(), Self::CAPACITY);
        let inner = &inner[..len];
        let mut v = REUSABLE_STACK
            .lock()
            .unwrap()
            .take()
            .unwrap_or_else(|| Vec::with_capacity(Self::CAPACITY));
        v.clear();
        #[cfg(feature = "unsafe-stack")]
        // SAFETY:
        // inner was shorted to the minimum of its original length and Self::CAPACITY.
        // v was taken from REUSABLE_STACK which was put there by Stack::drop or was created with
        // capacity Self::CAPACITY. Therefore it always has capacity Self::CAPACITY.
        unsafe {
            std::hint::assert_unchecked(inner.len() <= v.capacity());
        }
        v.extend_from_slice(inner);
        Self(v)
    }

    pub fn as_slice(&self) -> &[u256] {
        self.0.as_slice()
    }

    pub fn len(&self) -> usize {
        self.0.len()
    }

    pub fn push(&mut self, value: impl Into<u256>) -> Result<(), FailStatus> {
        if self.0.len() >= Self::CAPACITY {
            return Err(FailStatus::StackOverflow);
        }
        #[cfg(feature = "unsafe-stack")]
        // SAFETY:
        // self.0 is initialized with capacity Self::CAPACITY and never shrunk.
        unsafe {
            std::hint::assert_unchecked(self.0.capacity() == Self::CAPACITY);
        }
        self.0.push(value.into());
        Ok(())
    }

    pub fn swap_with_top(&mut self, nth: usize) -> Result<(), FailStatus> {
        self.check_underflow(nth + 1)?;

        #[cfg(not(feature = "unsafe-stack"))]
        {
            let len = self.0.len();
            self.0.swap(len - 1, len - 1 - nth);
        }
        #[cfg(feature = "unsafe-stack")]
        {
            let start = self.0.as_mut_ptr();
            // SAFETY:
            // This does not wrap and the whole range is valid.
            let top = unsafe { start.add(self.len() - 1) };
            // SAFETY:
            // This does not wrap and the whole range is valid.
            let nth = unsafe { top.sub(nth) };
            // SAFETY:
            // top and nth are valid pointers into the initialized part of the vector.
            unsafe {
                std::ptr::swap(top, nth);
            }
        }

        Ok(())
    }

    pub fn pop<const N: usize>(&mut self) -> Result<[u256; N], FailStatus> {
        self.check_underflow(N)?;

        let new_len = self.0.len() - N;
        let mut array = [u256::ZERO; N];
        array.copy_from_slice(&self.0[new_len..]);
        self.0.truncate(new_len);
        Ok(array)
    }

    pub fn peek(&self) -> Option<&u256> {
        self.0.last()
    }

    pub fn nth(&self, nth: usize) -> Result<u256, FailStatus> {
        self.check_underflow(nth + 1)?;
        #[cfg(not(feature = "unsafe-stack"))]
        return Ok(self.0[self.0.len() - 1 - nth]);
        #[cfg(feature = "unsafe-stack")]
        // SAFETY:
        // self.0.len() >= nth + 1 was checked in check_underflow.
        // Therefore self.0.len() - 1 - nth is in bounds.
        return Ok(*unsafe { self.0.get_unchecked(self.0.len() - 1 - nth) });
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

        let mut stack = Stack::new(&[u256::ZERO; Stack::CAPACITY]);
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
    fn check_underflow() {
        let stack = Stack::new(&[]);
        assert_eq!(stack.check_underflow(0), Ok(()));
        let stack = Stack::new(&[u256::ZERO]);
        assert_eq!(stack.check_underflow(1), Ok(()));
        assert_eq!(stack.check_underflow(2), Err(FailStatus::StackUnderflow));
    }
}

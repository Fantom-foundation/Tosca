#[cfg(feature = "stack-array")]
use std::{cmp::min, mem::MaybeUninit};

use crate::types::{u256, FailStatus};

#[cfg(feature = "stack-array")]
#[derive(Debug)]
pub struct Stack {
    data: [MaybeUninit<u256>; 1024],
    len: usize,
}

#[cfg(feature = "stack-array")]
impl Stack {
    pub fn new(inner: &[u256]) -> Self {
        let len = min(inner.len(), 1024);
        let mut s = Self {
            data: [MaybeUninit::uninit(); 1024],
            len,
        };
        // SAFETY:
        // &[T] and &[MaybeUninit<T>] have the same layout.
        // With nightly rust this could be replaced by MaybeUninit::copy_from_slice
        s.data[..len].copy_from_slice(unsafe {
            std::mem::transmute::<&[u256], &[std::mem::MaybeUninit<u256>]>(inner)
        });
        s
    }

    pub fn as_slice(&self) -> &[u256] {
        // SAFETY:
        // &[T] and &[MaybeUninit<T>] have the same layout and the first self.len elements are
        // initialized.
        unsafe {
            std::mem::transmute::<&[std::mem::MaybeUninit<u256>], &[u256]>(&self.data[..self.len])
        }
    }

    pub fn len(&self) -> usize {
        self.len
    }

    pub fn push(&mut self, value: impl Into<u256>) -> Result<(), FailStatus> {
        if self.len >= 1024 {
            return Err(FailStatus::StackOverflow);
        }

        self.data[self.len] = MaybeUninit::new(value.into());
        self.len += 1;
        Ok(())
    }

    pub fn pop<const N: usize>(&mut self) -> Result<[u256; N], FailStatus> {
        self.check_underflow(N)?;

        self.len -= N;
        let start = self.data.as_ptr() as *const u256;
        // SAFETY:
        // This does not wrap and the whole range from start to start + self.len is valid.
        let pop_start = unsafe { start.add(self.len) };
        // SAFETY:
        // The the first self.len elements are initialized (invariant).
        // `self.len` just got decremented by N, which means now that the first `self.len + N`
        // elements are initialized. Therefore, it is safe to read N elements starting at index
        // `self.len` as an array of length N and type u256.
        Ok(unsafe { std::ptr::read(pop_start as *const [u256; N]) })
    }

    pub fn peek(&self) -> Option<&u256> {
        if self.len == 0 {
            None
        } else {
            let top = &self.data[self.len - 1];
            // SAFETY:
            // The first self.len elements are initialized.
            let top = unsafe { std::mem::transmute::<&MaybeUninit<u256>, &u256>(top) };
            Some(top)
        }
    }

    pub fn swap_with_top(&mut self, nth: usize) -> Result<(), FailStatus> {
        self.check_underflow(nth + 1)?;

        let start = self.data.as_mut_ptr();
        // SAFETY:
        // This does not wrap and the whole range is valid.
        let top = unsafe { start.add(self.len - 1) };
        // SAFETY:
        // This does not wrap and the whole range is valid.
        let nth = unsafe { top.sub(nth) };
        // SAFETY:
        // top and nth are valid pointers into self.data.
        unsafe {
            std::ptr::swap(top, nth);
        }
        Ok(())
    }

    pub fn nth(&self, nth: usize) -> Result<u256, FailStatus> {
        self.check_underflow(nth + 1)?;
        let start = self.data.as_ptr() as *const u256;
        // SAFETY:
        // This does not wrap and the whole range is valid.
        let nth = unsafe { start.add(self.len - 1 - nth) };
        // SAFETY:
        // nth is a valid pointer into self.data which points to one of the first self.len elements,
        // which are all initialized, so the access is in bounds and it is safe so read the element
        // as u256.
        Ok(unsafe { *nth })
    }

    #[inline(always)]
    fn check_underflow(&self, min_len: usize) -> Result<(), FailStatus> {
        if self.len < min_len {
            return Err(FailStatus::StackUnderflow);
        }
        Ok(())
    }
}

#[cfg(not(feature = "stack-array"))]
#[derive(Debug)]
pub struct Stack(Vec<u256>);

#[cfg(not(feature = "stack-array"))]
impl Stack {
    pub fn new(inner: &[u256]) -> Self {
        Self(Vec::from(inner))
    }

    pub fn as_slice(&self) -> &[u256] {
        self.0.as_slice()
    }

    pub fn len(&self) -> usize {
        self.0.len()
    }

    pub fn push(&mut self, value: impl Into<u256>) -> Result<(), FailStatus> {
        if self.0.len() >= 1024 {
            return Err(FailStatus::StackOverflow);
        }
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
        Ok(self.0[self.0.len() - nth - 1])
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
    fn check_underflow() {
        let stack = Stack::new(&[]);
        assert_eq!(stack.check_underflow(0), Ok(()));
        let stack = Stack::new(&[u256::ZERO]);
        assert_eq!(stack.check_underflow(1), Ok(()));
        assert_eq!(stack.check_underflow(2), Err(FailStatus::StackUnderflow));
    }
}

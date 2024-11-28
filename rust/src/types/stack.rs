use std::cmp::min;
#[cfg(feature = "alloc-reuse")]
use std::sync::Mutex;

use crate::types::{u256, FailStatus};

struct NonZero<const N: usize>;

impl<const N: usize> NonZero<N> {
    const VALID: () = assert!(N > 0);
}

/// Wrapper around [`&mut u256`] that ensures that the only possible operation is to write once to
/// this memory location.
pub struct PushGuard<'p>(&'p mut u256);

impl PushGuard<'_> {
    pub fn push(self, value: impl Into<u256>) {
        *self.0 = value.into();
    }
}

#[cfg(feature = "alloc-reuse")]
static REUSABLE_STACK: Mutex<Vec<Vec<u256>>> = Mutex::new(Vec::new());

#[derive(Debug)]
pub struct Stack(Vec<u256>);

#[cfg(feature = "alloc-reuse")]
impl Drop for Stack {
    fn drop(&mut self) {
        let mut stack = Vec::new();
        std::mem::swap(&mut stack, &mut self.0);
        REUSABLE_STACK.lock().unwrap().push(stack);
    }
}

impl Stack {
    const CAPACITY: usize = 1024;

    #[inline(never)]
    pub fn new(inner: &[u256]) -> Self {
        let len = min(inner.len(), Self::CAPACITY);
        let inner = &inner[..len];
        #[cfg(not(feature = "alloc-reuse"))]
        let mut v = Vec::with_capacity(Self::CAPACITY);
        #[cfg(feature = "alloc-reuse")]
        let mut v = REUSABLE_STACK
            .lock()
            .unwrap()
            .pop()
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

    pub fn swap_with_top<const N: usize>(&mut self) -> Result<(), FailStatus> {
        let () = const { NonZero::<N>::VALID };

        self.check_underflow(N + 1)?;

        #[cfg(not(feature = "unsafe-stack"))]
        {
            let len = self.0.len();
            self.0.swap(len - 1, len - 1 - N);
        }
        #[cfg(feature = "unsafe-stack")]
        {
            let start = self.0.as_mut_ptr();
            // SAFETY:
            // This does not wrap and the whole range is valid.
            let top = unsafe { start.add(self.len() - 1) };
            // SAFETY:
            // This does not wrap and the whole range is valid.
            let nth = unsafe { top.sub(N) };
            // SAFETY:
            // top and nth are valid pointers into the initialized part of the vector.
            unsafe {
                std::ptr::swap_nonoverlapping(top, nth, 1);
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

    pub fn pop_with_guard<const N: usize>(&mut self) -> Result<(PushGuard, [u256; N]), FailStatus> {
        self.check_underflow(N)?;

        unsafe {
            self.0.set_len(self.len() - N + 1);
        }
        let start = self.0.as_ptr() as *mut u256;
        // SAFETY:
        // This does not wrap and the whole range from start to start + self.len is valid.
        let pop_start = unsafe { start.add(self.len() - 1) };
        // SAFETY:
        // The the first self.len elements are initialized (invariant).
        // `self.len` just got decremented by N - 1, which means now that the first `self.len - 1 +
        // (N + 1)` elements are initialized. Therefore, it is safe to read N elements
        // starting at index `self.len - 1` as an array of length N and type u256.
        let pop_data = unsafe { *(pop_start as *const [u256; N]) };
        // SAFETY:
        // The data for pop_data is copied out so there are no other references to this data.
        // The validity of the data is the same as for pop_data. Because the pointer is valid and no
        // one else holds a reference to it, it is safe to cast it to a mutable reference.
        let push_guard = PushGuard(unsafe { &mut *pop_start });
        Ok((push_guard, pop_data))
    }

    pub fn peek(&self) -> Option<&u256> {
        self.0.last()
    }

    pub fn dup<const N: usize>(&mut self) -> Result<(), FailStatus> {
        // Note: N is 1 based (N = x -> duplicate element at index x-1)
        let () = const { NonZero::<N>::VALID };

        self.check_underflow(N)?;
        #[cfg(not(feature = "unsafe-stack"))]
        let element = self.0[self.0.len() - N];
        #[cfg(feature = "unsafe-stack")]
        // SAFETY:
        // self.0.len() >= nth + 1 was checked in check_underflow.
        // Therefore self.0.len() - 1 - nth is in bounds.
        let element = *unsafe { self.0.get_unchecked(self.0.len() - N) };
        self.push(element)
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
    fn dup() {
        let mut stack = Stack::new(&[u256::MAX, u256::ZERO]);
        stack.dup::<1>().unwrap();
        assert_eq!(stack.as_slice(), [u256::MAX, u256::ZERO, u256::ZERO]);

        let mut stack = Stack::new(&[u256::MAX, u256::ZERO]);
        stack.dup::<2>().unwrap();
        assert_eq!(stack.as_slice(), [u256::MAX, u256::ZERO, u256::MAX]);

        let mut stack = Stack::new(&[u256::MAX, u256::ZERO]);
        assert_eq!(stack.dup::<3>(), Err(FailStatus::StackUnderflow));

        let mut stack = Stack::new(&[u256::ZERO; 1024]);
        assert_eq!(stack.dup::<1>(), Err(FailStatus::StackOverflow));
    }

    #[test]
    fn swap_with_top() {
        let mut stack = Stack::new(&[u256::MAX, u256::ONE]);
        assert_eq!(stack.swap_with_top::<1>(), Ok(()));
        assert_eq!(stack.as_slice(), [u256::ONE, u256::MAX]);

        let mut stack = Stack::new(&[u256::MAX, u256::ONE]);
        assert_eq!(stack.swap_with_top::<2>(), Err(FailStatus::StackUnderflow));
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

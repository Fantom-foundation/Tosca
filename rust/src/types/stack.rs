use crate::types::{u256, FailStatus};

#[derive(Debug)]
pub struct Stack(Vec<u256>);

impl Stack {
    pub fn new(inner: Vec<u256>) -> Self {
        Self(inner)
    }

    pub fn into_inner(self) -> Vec<u256> {
        self.0
    }

    pub fn push(&mut self, value: impl Into<u256>) -> Result<(), FailStatus> {
        self.check_overflow_on_push()?;
        self.0.push(value.into());
        Ok(())
    }

    pub fn swap_with_top(&mut self, nth: usize) -> Result<(), FailStatus> {
        self.check_underflow(nth + 1)?;

        let len = self.0.len();
        self.0.swap(len - 1, len - 1 - nth);
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

    pub fn nth(&self, nth: usize) -> Result<u256, FailStatus> {
        self.check_underflow(nth + 1)?;
        Ok(self.0[self.0.len() - nth - 1])
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
    fn push() {
        let mut stack = Stack::new(Vec::new());
        assert_eq!(stack.push(u256::MAX), Ok(()));
        assert_eq!(stack.into_inner().pop(), Some(u256::MAX));

        let mut stack = Stack::new(vec![u256::ZERO; 1024]);
        assert_eq!(stack.push(u256::ZERO), Err(FailStatus::StackOverflow));
    }

    #[test]
    fn pop() {
        let mut stack = Stack::new(vec![u256::MAX]);
        assert_eq!(stack.pop::<1>(), Ok([u256::MAX]));

        let mut stack = Stack::new(vec![]);
        assert_eq!(stack.pop::<1>(), Err(FailStatus::StackUnderflow));

        let mut stack = Stack::new(vec![u256::ONE, u256::MAX]);
        assert_eq!(stack.pop::<2>(), Ok([u256::ONE, u256::MAX]));

        let mut stack = Stack::new(vec![u256::MAX]);
        assert_eq!(stack.pop::<2>(), Err(FailStatus::StackUnderflow));
    }

    #[test]
    fn nth() {
        let stack = Stack::new(vec![u256::MAX, u256::ZERO]);
        assert_eq!(stack.nth(0), Ok(u256::ZERO));
        assert_eq!(stack.nth(1), Ok(u256::MAX));
        assert_eq!(stack.nth(2), Err(FailStatus::StackUnderflow));
    }

    #[test]
    fn swap_with_top() {
        let mut stack = Stack::new(vec![u256::MAX, u256::ZERO]);
        assert_eq!(stack.swap_with_top(0), Ok(()));
        assert_eq!(stack.into_inner(), [u256::MAX, u256::ZERO]);

        let mut stack = Stack::new(vec![u256::MAX, u256::ZERO]);
        assert_eq!(stack.swap_with_top(1), Ok(()));
        assert_eq!(stack.into_inner(), [u256::ZERO, u256::MAX]);

        let mut stack = Stack::new(vec![u256::MAX, u256::ZERO]);
        assert_eq!(stack.swap_with_top(2), Err(FailStatus::StackUnderflow));
    }

    #[test]
    fn check_overflow_on_push() {
        let stack = Stack::new(vec![u256::MAX; 1023]);
        assert_eq!(stack.check_overflow_on_push(), Ok(()));
        let stack = Stack::new(vec![u256::MAX; 1024]);
        assert_eq!(
            stack.check_overflow_on_push(),
            Err(FailStatus::StackOverflow)
        );
    }

    #[test]
    fn check_underflow() {
        let stack = Stack::new(vec![]);
        assert_eq!(stack.check_underflow(0), Ok(()));
        let stack = Stack::new(vec![u256::ZERO]);
        assert_eq!(stack.check_underflow(1), Ok(()));
        assert_eq!(stack.check_underflow(2), Err(FailStatus::StackUnderflow));
    }
}

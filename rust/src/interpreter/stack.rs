use evmc_vm::{StatusCode, StepStatusCode};

use crate::types::u256;

#[derive(Debug)]
pub struct Stack(Vec<u256>);

impl Stack {
    pub fn new(inner: Vec<u256>) -> Self {
        Self(inner)
    }

    pub fn into_inner(self) -> Vec<u256> {
        self.0
    }

    pub fn push(&mut self, value: impl Into<u256>) -> Result<(), (StepStatusCode, StatusCode)> {
        self.check_overflow_on_push()?;
        self.0.push(value.into());
        Ok(())
    }

    pub fn swap_with_top(&mut self, nth: usize) -> Result<(), (StepStatusCode, StatusCode)> {
        self.check_underflow(nth + 1)?;

        let len = self.0.len();
        self.0.swap(len - 1, len - 1 - nth);
        Ok(())
    }

    pub fn pop<const N: usize>(&mut self) -> Result<[u256; N], (StepStatusCode, StatusCode)> {
        self.check_underflow(N)?;

        let mut array = [u256::ZERO; N];
        for element in &mut array {
            *element = self.0.pop().unwrap();
        }
        Ok(array)
    }

    pub fn nth(&self, nth: usize) -> Result<u256, (StepStatusCode, StatusCode)> {
        self.check_underflow(nth)?;
        Ok(self.0[self.0.len() - nth])
    }

    #[inline(always)]
    fn check_overflow_on_push(&self) -> Result<(), (StepStatusCode, StatusCode)> {
        if self.0.len() + 1 > 1024 {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_STACK_OVERFLOW,
            ));
        }
        Ok(())
    }

    #[inline(always)]
    fn check_underflow(&self, nth: usize) -> Result<(), (StepStatusCode, StatusCode)> {
        if self.0.len() < nth {
            return Err((
                StepStatusCode::EVMC_STEP_FAILED,
                StatusCode::EVMC_STACK_UNDERFLOW,
            ));
        }
        Ok(())
    }
}

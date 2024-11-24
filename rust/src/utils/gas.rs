use evmc_vm::{AccessStatus, Address, Revision};

use crate::{
    types::{u256, ExecutionContextTrait, FailStatus},
    utils::word_size,
};

#[derive(Debug)]
pub struct GasRefund(i64);

impl GasRefund {
    pub fn new(gas: i64) -> Self {
        Self(gas)
    }

    pub fn as_i64(&self) -> i64 {
        self.0
    }

    #[inline(always)]
    pub fn add(&mut self, gas: i64) -> Result<(), FailStatus> {
        let (gas, overflow) = self.0.overflowing_add(gas);
        if overflow {
            return Err(FailStatus::OutOfGas);
        }
        self.0 = gas;
        Ok(())
    }
}

// Invariant: gas <= i64::MAX
#[derive(Debug)]
pub struct Gas(u64);

impl PartialEq<u64> for Gas {
    fn eq(&self, other: &u64) -> bool {
        self.0.eq(other)
    }
}

impl PartialOrd<u64> for Gas {
    fn partial_cmp(&self, other: &u64) -> Option<std::cmp::Ordering> {
        Some(self.0.cmp(other))
    }
}

impl Gas {
    pub fn new(gas: i64) -> Self {
        if gas < 0 {
            Self(0)
        } else {
            Self(gas as u64)
        }
    }

    pub fn as_u64(&self) -> u64 {
        self.0
    }

    #[inline(always)]
    pub fn add(&mut self, gas: i64) -> Result<(), FailStatus> {
        let (gas, overflow) = (self.0 as i64).overflowing_add(gas);
        if gas < 0 || overflow {
            return Err(FailStatus::OutOfGas);
        }
        self.0 = gas as u64;
        Ok(())
    }

    #[inline(always)]
    pub fn consume(&mut self, gas: u64) -> Result<(), FailStatus> {
        if self.0 < gas {
            return Err(FailStatus::OutOfGas);
        }
        self.0 -= gas;
        Ok(())
    }

    #[inline(always)]
    pub fn consume_positive_value_cost(&mut self, value: &u256) -> Result<(), FailStatus> {
        if *value != u256::ZERO {
            self.consume(9_000)?;
        }
        Ok(())
    }

    #[inline(always)]
    pub fn consume_value_to_empty_account_cost(
        &mut self,
        value: &u256,
        addr: &Address,
        context: &mut dyn ExecutionContextTrait,
    ) -> Result<(), FailStatus> {
        if *value != u256::ZERO && !context.account_exists(addr) {
            self.consume(25_000)?;
        }
        Ok(())
    }

    #[inline(always)]
    pub fn consume_address_access_cost(
        &mut self,
        addr: &Address,
        revision: Revision,
        context: &mut dyn ExecutionContextTrait,
    ) -> Result<(), FailStatus> {
        if revision < Revision::EVMC_BERLIN {
            return Ok(());
        }
        if context.access_account(addr) == AccessStatus::EVMC_ACCESS_COLD {
            self.consume(2_600)
        } else {
            self.consume(100)
        }
    }

    #[inline(always)]
    pub fn consume_copy_cost(&mut self, len: u64) -> Result<(), FailStatus> {
        let cost = word_size(len)? * 3; // does not overflow because word_size divides by 32
        self.consume(cost)
    }
}

#[cfg(test)]
mod tests {
    use evmc_vm::{AccessStatus, Address, Revision};
    use mockall::predicate;

    use crate::{
        interpreter::Interpreter,
        types::{u256, FailStatus, MockExecutionContextTrait, MockExecutionMessage, Opcode},
        utils::Gas,
    };

    #[test]
    fn consume_gas() {
        let mut gas_left = Gas::new(1);
        assert_eq!(gas_left.consume(0), Ok(()));
        assert_eq!(gas_left, 1);

        let mut gas_left = Gas::new(1);
        assert_eq!(gas_left.consume(1), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = Gas::new(1);
        assert_eq!(gas_left.consume(2), Err(FailStatus::OutOfGas));
        assert_eq!(gas_left, 1);
    }

    #[test]
    fn consume_positive_value_cost() {
        let mut gas_left = Gas::new(1);
        assert_eq!(gas_left.consume_positive_value_cost(&u256::ZERO), Ok(()));
        assert_eq!(gas_left, 1);

        let mut gas_left = Gas::new(9_000);
        assert_eq!(gas_left.consume_positive_value_cost(&u256::ONE), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = Gas::new(1);
        assert_eq!(
            gas_left.consume_positive_value_cost(&u256::ONE),
            Err(FailStatus::OutOfGas)
        );
        assert_eq!(gas_left, 1);
    }

    #[test]
    fn consume_value_to_empty_account_cost() {
        let cases = [
            (true, u256::ZERO, false),
            (true, u256::ONE, false),
            (false, u256::ZERO, false),
            (false, u256::ONE, true),
        ];

        for (exists, value, consume) in cases {
            let addr = Address::from(u256::ONE);
            let message = MockExecutionMessage::default().into();

            let mut context = MockExecutionContextTrait::new();
            context
                .expect_account_exists()
                .times(if value == u256::ZERO {
                    0
                } else if consume {
                    2
                } else {
                    1
                })
                .with(predicate::eq(addr))
                .return_const(exists);

            let mut interpreter = Interpreter::new(
                Revision::EVMC_ISTANBUL,
                &message,
                &mut context,
                &[Opcode::Call as u8],
            );
            interpreter.gas_left = Gas::new(if consume { 25_000 } else { 0 });

            assert_eq!(
                interpreter.gas_left.consume_value_to_empty_account_cost(
                    &value,
                    &addr,
                    interpreter.context
                ),
                Ok(())
            );
            assert_eq!(interpreter.gas_left, 0);

            if consume {
                interpreter.gas_left = Gas::new(0);

                assert_eq!(
                    interpreter.gas_left.consume_value_to_empty_account_cost(
                        &value,
                        &addr,
                        interpreter.context
                    ),
                    Err(FailStatus::OutOfGas)
                );
            }
        }
    }

    #[test]
    fn consume_address_access_cost() {
        let cases = [
            (
                Revision::EVMC_ISTANBUL,
                AccessStatus::EVMC_ACCESS_COLD,
                Gas::new(0),
            ),
            (
                Revision::EVMC_BERLIN,
                AccessStatus::EVMC_ACCESS_COLD,
                Gas::new(2_600),
            ),
            (
                Revision::EVMC_BERLIN,
                AccessStatus::EVMC_ACCESS_WARM,
                Gas::new(100),
            ),
        ];
        for (revision, access_status, gas) in cases {
            let addr = Address::from(u256::ONE);
            let message = MockExecutionMessage::default().into();

            let mut context = MockExecutionContextTrait::new();
            context
                .expect_access_account()
                .times(if revision < Revision::EVMC_BERLIN {
                    0
                } else {
                    1
                })
                .with(predicate::eq(addr))
                .return_const(access_status);

            let mut interpreter =
                Interpreter::new(revision, &message, &mut context, &[Opcode::Call as u8]);
            interpreter.gas_left = gas;

            assert_eq!(
                interpreter.gas_left.consume_address_access_cost(
                    &addr,
                    interpreter.revision,
                    interpreter.context
                ),
                Ok(())
            );
            assert_eq!(interpreter.gas_left, 0);
        }
    }

    #[test]
    fn consume_copy_cost() {
        let mut gas_left = Gas::new(1);
        assert_eq!(gas_left.consume_copy_cost(0), Ok(()));
        assert_eq!(gas_left, 1);

        let mut gas_left = Gas::new(3);
        assert_eq!(gas_left.consume_copy_cost(1), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = Gas::new(3);
        assert_eq!(gas_left.consume_copy_cost(32), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = Gas::new(6);
        assert_eq!(gas_left.consume_copy_cost(33), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = Gas::new(2);
        assert_eq!(gas_left.consume_copy_cost(1), Err(FailStatus::OutOfGas));
        assert_eq!(gas_left, 2);

        let mut gas_left = Gas::new(2);
        assert_eq!(
            gas_left.consume_copy_cost(u64::MAX),
            Err(FailStatus::OutOfGas)
        );
        assert_eq!(gas_left, 2);
    }
}

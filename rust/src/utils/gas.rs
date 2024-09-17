use evmc_vm::{AccessStatus, Address, Revision, StatusCode};

use crate::{
    interpreter::Interpreter,
    types::{u256, ExecutionContextTrait},
    utils::word_size,
};

#[inline(always)]
pub fn consume_gas(gas: u64, gas_left: &mut u64) -> Result<(), StatusCode> {
    if *gas_left < gas {
        return Err(StatusCode::EVMC_OUT_OF_GAS);
    }
    *gas_left -= gas;
    Ok(())
}

#[inline(always)]
pub fn consume_positive_value_cost(value: &u256, gas_left: &mut u64) -> Result<(), StatusCode> {
    if *value != u256::ZERO {
        consume_gas(9_000, gas_left)?;
    }
    Ok(())
}

#[inline(always)]
pub fn consume_value_to_empty_account_cost<E: ExecutionContextTrait>(
    value: &u256,
    addr: &Address,
    interpreter: &mut Interpreter<E>,
) -> Result<(), StatusCode> {
    if *value != u256::ZERO && !interpreter.context.account_exists(addr) {
        consume_gas(25_000, &mut interpreter.gas_left)?;
    }
    Ok(())
}

#[inline(always)]
pub fn consume_address_access_cost<E: ExecutionContextTrait>(
    addr: &Address,
    interpreter: &mut Interpreter<E>,
) -> Result<(), StatusCode> {
    if interpreter.revision < Revision::EVMC_BERLIN {
        return Ok(());
    }
    if interpreter.context.access_account(addr) == AccessStatus::EVMC_ACCESS_COLD {
        consume_gas(2_600, &mut interpreter.gas_left)
    } else {
        consume_gas(100, &mut interpreter.gas_left)
    }
}

#[inline(always)]
pub fn consume_copy_cost(len: u64, gas_left: &mut u64) -> Result<(), StatusCode> {
    let cost = word_size(len)? * 3; // does not overflow because word_size divides by 32
    consume_gas(cost, gas_left)
}

#[cfg(test)]
mod tests {
    use evmc_vm::{AccessStatus, Address, Revision, StatusCode};
    use mockall::predicate;

    use crate::{
        interpreter::Interpreter,
        types::{u256, MockExecutionContextTrait, MockExecutionMessage, Opcode},
    };

    #[test]
    fn consume_gas() {
        let mut gas_left = 1;
        assert_eq!(super::consume_gas(0, &mut gas_left,), Ok(()));
        assert_eq!(gas_left, 1);

        let mut gas_left = 1;
        assert_eq!(super::consume_gas(1, &mut gas_left,), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = 1;
        assert_eq!(
            super::consume_gas(2, &mut gas_left,),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );
        assert_eq!(gas_left, 1);
    }

    #[test]
    fn consume_positive_value_cost() {
        let mut gas_left = 1;
        assert_eq!(
            super::consume_positive_value_cost(&u256::ZERO, &mut gas_left),
            Ok(())
        );
        assert_eq!(gas_left, 1);

        let mut gas_left = 9_000;
        assert_eq!(
            super::consume_positive_value_cost(&u256::ONE, &mut gas_left),
            Ok(())
        );
        assert_eq!(gas_left, 0);

        let mut gas_left = 1;
        assert_eq!(
            super::consume_positive_value_cost(&u256::ONE, &mut gas_left),
            Err(StatusCode::EVMC_OUT_OF_GAS)
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
                Revision::EVMC_FRONTIER,
                &message,
                &mut context,
                &[Opcode::Call as u8],
            );
            interpreter.gas_left = if consume { 25_000 } else { 0 };

            assert_eq!(
                super::consume_value_to_empty_account_cost(&value, &addr, &mut interpreter),
                Ok(())
            );
            assert_eq!(interpreter.gas_left, 0);

            if consume {
                interpreter.gas_left = 0;

                assert_eq!(
                    super::consume_value_to_empty_account_cost(&value, &addr, &mut interpreter),
                    Err(StatusCode::EVMC_OUT_OF_GAS)
                );
            }
        }
    }

    #[test]
    fn consume_address_access_cost() {
        let cases = [
            (Revision::EVMC_FRONTIER, AccessStatus::EVMC_ACCESS_COLD, 0),
            (Revision::EVMC_BERLIN, AccessStatus::EVMC_ACCESS_COLD, 2_600),
            (Revision::EVMC_BERLIN, AccessStatus::EVMC_ACCESS_WARM, 100),
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
                super::consume_address_access_cost(&addr, &mut interpreter),
                Ok(())
            );
            assert_eq!(interpreter.gas_left, 0);
        }
    }

    #[test]
    fn consume_copy_cost() {
        let mut gas_left = 1;
        assert_eq!(super::consume_copy_cost(0, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 1);

        let mut gas_left = 3;
        assert_eq!(super::consume_copy_cost(1, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = 3;
        assert_eq!(super::consume_copy_cost(32, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = 6;
        assert_eq!(super::consume_copy_cost(33, &mut gas_left), Ok(()));
        assert_eq!(gas_left, 0);

        let mut gas_left = 2;
        assert_eq!(
            super::consume_copy_cost(1, &mut gas_left),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );
        assert_eq!(gas_left, 2);

        let mut gas_left = 2;
        assert_eq!(
            super::consume_copy_cost(u64::MAX, &mut gas_left),
            Err(StatusCode::EVMC_OUT_OF_GAS)
        );
        assert_eq!(gas_left, 2);
    }
}

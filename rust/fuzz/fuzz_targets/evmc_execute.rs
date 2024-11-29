#![no_main]

use core::slice;
use std::fmt::Debug;

use arbitrary::Arbitrary;
use driver::{host_interface::mocked_host_interface, Instance};
use evmrs::{
    evmc_vm::{
        ffi::{evmc_host_interface, evmc_message},
        AccessStatus, ExecutionResult, ExecutionTxContext, MessageKind, Revision, StatusCode,
        StorageStatus, Uint256,
    },
    u256, MockExecutionContextTrait,
};
use libfuzzer_sys::fuzz_target;

struct InterpreterArgs<'a> {
    instance: Instance,
    host: evmc_host_interface,
    context: MockExecutionContextTrait,
    revision: Revision,
    message: evmc_message,
    code: &'a [u8],
}

impl Debug for InterpreterArgs<'_> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("InterpreterArgs")
            .field("host", &self.host)
            .field("context", &self.context)
            .field("revision", &self.revision)
            .field("message", &self.message)
            .field("code", &self.code)
            .finish()
    }
}

impl<'a> Arbitrary<'a> for InterpreterArgs<'a> {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        let input = <&[u8]>::arbitrary(u)?;
        let code = <&[u8]>::arbitrary(u)?;
        let message = evmc_message {
            kind: *u.choose(&[
                MessageKind::EVMC_CALL,
                MessageKind::EVMC_CALLCODE,
                MessageKind::EVMC_CREATE,
                MessageKind::EVMC_CREATE2,
                MessageKind::EVMC_DELEGATECALL,
                MessageKind::EVMC_EOFCREATE,
            ])?,
            flags: u32::arbitrary(u)?,
            depth: i32::arbitrary(u)?,
            gas: u.int_in_range(0..=5_000_000_000)?, // see go/ct/evm_fuzz_test.go
            recipient: u256::arbitrary(u)?.into(),
            sender: u256::arbitrary(u)?.into(),
            input_data: input.as_ptr(),
            input_size: input.len(),
            value: u256::arbitrary(u)?.into(),
            create2_salt: u256::arbitrary(u)?.into(),
            code_address: u256::arbitrary(u)?.into(),
            code: code.as_ptr(),
            code_size: code.len(),
            code_hash: std::ptr::null(),
        };

        let mut context = MockExecutionContextTrait::new();
        let len = u.arbitrary_len::<[u8; 32]>()?;
        let bytes = u.bytes(len * 32)?;
        let blob_hashes: &[Uint256] =
            unsafe { slice::from_raw_parts(bytes.as_ptr() as *const Uint256, len) };
        let txcontext = ExecutionTxContext {
            tx_gas_price: u256::arbitrary(u)?.into(),
            tx_origin: u256::arbitrary(u)?.into(),
            block_coinbase: u256::arbitrary(u)?.into(),
            block_number: Arbitrary::arbitrary(u)?,
            block_timestamp: Arbitrary::arbitrary(u)?,
            block_gas_limit: Arbitrary::arbitrary(u)?,
            block_prev_randao: u256::arbitrary(u)?.into(),
            chain_id: u256::arbitrary(u)?.into(),
            block_base_fee: u256::arbitrary(u)?.into(),
            blob_base_fee: u256::arbitrary(u)?.into(),
            blob_hashes: blob_hashes.as_ptr(),
            blob_hashes_count: blob_hashes.len(),
            initcodes: std::ptr::null(),
            initcodes_count: 0,
        };
        context.expect_get_tx_context().return_const(txcontext);
        context
            .expect_account_exists()
            .return_const(bool::arbitrary(u)?);
        context
            .expect_get_storage()
            .return_const(Uint256::from(u256::arbitrary(u)?));
        context.expect_set_storage().return_const(*u.choose(&[
            StorageStatus::EVMC_STORAGE_ASSIGNED,
            StorageStatus::EVMC_STORAGE_ADDED,
            StorageStatus::EVMC_STORAGE_DELETED,
            StorageStatus::EVMC_STORAGE_MODIFIED,
            StorageStatus::EVMC_STORAGE_DELETED_ADDED,
            StorageStatus::EVMC_STORAGE_MODIFIED_DELETED,
            StorageStatus::EVMC_STORAGE_DELETED_RESTORED,
            StorageStatus::EVMC_STORAGE_ADDED_DELETED,
            StorageStatus::EVMC_STORAGE_MODIFIED_RESTORED,
        ])?);
        context
            .expect_get_balance()
            .return_const(Uint256::from(u256::arbitrary(u)?));
        context
            .expect_get_code_size()
            .return_const(usize::arbitrary(u)?);
        context
            .expect_get_code_hash()
            .return_const(Uint256::from(u256::arbitrary(u)?));
        context
            .expect_copy_code()
            .return_const(usize::arbitrary(u)?);
        context
            .expect_selfdestruct()
            .return_const(bool::arbitrary(u)?);
        let execution_result = ExecutionResult {
            status_code: *u.choose(&[
                StatusCode::EVMC_SUCCESS,
                StatusCode::EVMC_FAILURE,
                StatusCode::EVMC_REVERT,
                StatusCode::EVMC_OUT_OF_GAS,
                StatusCode::EVMC_INVALID_INSTRUCTION,
                StatusCode::EVMC_UNDEFINED_INSTRUCTION,
                StatusCode::EVMC_STACK_OVERFLOW,
                StatusCode::EVMC_STACK_UNDERFLOW,
                StatusCode::EVMC_BAD_JUMP_DESTINATION,
                StatusCode::EVMC_INVALID_MEMORY_ACCESS,
                StatusCode::EVMC_CALL_DEPTH_EXCEEDED,
                StatusCode::EVMC_STATIC_MODE_VIOLATION,
                StatusCode::EVMC_PRECOMPILE_FAILURE,
                StatusCode::EVMC_CONTRACT_VALIDATION_FAILURE,
                StatusCode::EVMC_ARGUMENT_OUT_OF_RANGE,
                StatusCode::EVMC_WASM_UNREACHABLE_INSTRUCTION,
                StatusCode::EVMC_WASM_TRAP,
                StatusCode::EVMC_INSUFFICIENT_BALANCE,
                StatusCode::EVMC_INTERNAL_ERROR,
                StatusCode::EVMC_REJECTED,
                StatusCode::EVMC_OUT_OF_MEMORY,
            ])?,
            gas_left: Arbitrary::arbitrary(u)?,
            gas_refund: Arbitrary::arbitrary(u)?,
            output: Arbitrary::arbitrary(u)?,
            create_address: Option::<u256>::arbitrary(u)?.map(Into::into),
        };
        let clone_result = move || ExecutionResult {
            status_code: execution_result.status_code,
            gas_left: execution_result.gas_left,
            gas_refund: execution_result.gas_refund,
            output: execution_result.output.clone(),
            create_address: execution_result.create_address,
        };
        context.expect_call().returning(move |_| clone_result());
        context
            .expect_get_block_hash()
            .return_const(Uint256::from(u256::arbitrary(u)?));
        context.expect_emit_log().return_const(());
        context.expect_access_account().return_const(*u.choose(&[
            AccessStatus::EVMC_ACCESS_COLD,
            AccessStatus::EVMC_ACCESS_WARM,
        ])?);
        context.expect_access_storage().return_const(*u.choose(&[
            AccessStatus::EVMC_ACCESS_COLD,
            AccessStatus::EVMC_ACCESS_WARM,
        ])?);
        context
            .expect_get_transient_storage()
            .return_const(Uint256::from(u256::arbitrary(u)?));
        context.expect_set_transient_storage().return_const(());

        let revision = *u.choose(&[
            Revision::EVMC_FRONTIER,
            Revision::EVMC_HOMESTEAD,
            Revision::EVMC_TANGERINE_WHISTLE,
            Revision::EVMC_SPURIOUS_DRAGON,
            Revision::EVMC_BYZANTIUM,
            Revision::EVMC_CONSTANTINOPLE,
            Revision::EVMC_PETERSBURG,
            Revision::EVMC_ISTANBUL,
            Revision::EVMC_BERLIN,
            Revision::EVMC_LONDON,
            Revision::EVMC_PARIS,
            Revision::EVMC_SHANGHAI,
            Revision::EVMC_CANCUN,
            Revision::EVMC_PRAGUE,
            Revision::EVMC_OSAKA,
        ])?;
        let args = Self {
            instance: Instance::default(),
            host: mocked_host_interface(),
            context,
            revision,
            message,
            code: Arbitrary::arbitrary(u)?,
        };
        Ok(args)
    }
}

fuzz_target!(|args: InterpreterArgs| {
    let mut args = args; // fuzz_target does not accept mutable arguments

    // Note: cargo-fuzz compiles with -Cpanic=abort so the catch_unwind in evmrs::ffi no longer
    // catches panics.
    let _result = args.instance.run(
        &args.host,
        &mut args.context,
        args.revision,
        &args.message,
        args.code,
    );
});

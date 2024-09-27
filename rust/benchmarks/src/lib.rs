#![allow(unused_crate_dependencies)]

use core::slice;

use driver::{self, get_tx_context_zeroed, host_interface::null_ptr_host_interface, Instance};
use evmc_vm::{
    ffi::{evmc_host_interface, evmc_message},
    Revision, StatusCode,
};
use evmrs::{MockExecutionMessage, Opcode};

pub struct RunArgs {
    instance: Instance,
    host: evmc_host_interface,
    revision: Revision,
    message: evmc_message,
    code: &'static [u8],
}

impl RunArgs {
    /// Create arguments for the interpreter that outline the FFI overhead.
    /// In particular those arguments try to trigger all possible allocations that happen because
    /// pointers get passed via the FFI interface and then the data is copied into a fresh
    /// allocation on the other side.
    /// - `ExecutionMessage` contains non-empty input
    /// - `code` is non-empty so that `CodeReader` must allocate memory to store the jump analysis
    ///   results
    /// - `code` contains `Opcode::MStore` so that `memory` is non-empty
    /// - `code` returns a single word so that `output` is non-empty
    pub fn ffi_overhead() -> Self {
        let instance = Instance::default();
        let mut host = null_ptr_host_interface();
        host.get_tx_context = Some(get_tx_context_zeroed);
        let message = MockExecutionMessage {
            input: Some(&[0]),
            ..Default::default()
        };

        Self {
            instance,
            host,
            revision: Revision::EVMC_CANCUN,
            message: message.to_evmc_message(),
            code: &[
                Opcode::Push1 as u8,
                u8::MAX, // value
                Opcode::Push1 as u8,
                0, // offset
                Opcode::MStore as u8,
                Opcode::Push1 as u8,
                32, // len
                Opcode::Push1 as u8,
                0, // offset
                Opcode::Return as u8,
            ],
        }
    }
}

pub fn run(args: &mut RunArgs) {
    // SAFETY:
    // `host` and `message` are valid pointers since they are created from references. `context`
    // is a null pointer but this is allowed by the evmc interface as long as the host
    // interface does not require a valid pointer, which is not the case here.
    let result =
        args.instance
            .run_with_null_context(&args.host, args.revision, &args.message, args.code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
    let output = unsafe { slice::from_raw_parts(result.output_data, result.output_size) };
    assert_eq!(output[..31], [0; 31]);
    assert_eq!(output[31..], [u8::MAX; 1]);
}

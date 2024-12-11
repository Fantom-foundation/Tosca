use common::{
    evmc_vm::{
        ffi::{evmc_host_interface, evmc_message},
        Revision, StatusCode, Uint256,
    },
    opcode::*,
    MockExecutionMessage,
};
use driver::{get_tx_context_zeroed, host_interface::null_ptr_host_interface, Instance};
use sha3::{Digest, Keccak256};

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
    /// - `code` is non-empty so that `CodeReader` must allocate memory to store the code analysis
    ///   results
    /// - `code` contains `MStore` so that `memory` is non-empty
    /// - `code` returns a single word so that `output` is non-empty
    pub fn ffi_overhead(size: u32) -> (Self, u32) {
        fn ffi_overhead_ref(input: u32) -> u32 {
            input
        }
        const CODE: [u8; 11] = [
            PUSH1,
            4, // offset
            CALLDATALOAD,
            PUSH1,
            0, // offset
            MSTORE,
            PUSH1,
            32, // len
            PUSH1,
            0, // offset
            RETURN,
        ];

        (Self::new(&CODE, size, None), ffi_overhead_ref(size))
    }

    // See go/examples/inc.go
    pub fn inc(size: u32) -> (Self, u32) {
        fn inc_ref(input: u32) -> u32 {
            input + 1
        }

        const CODE: [u8; 464] = [
            96, 128, 96, 64, 82, 52, 128, 21, 97, 0, 16, 87, 96, 0, 128, 253, 91, 80, 96, 4, 54,
            16, 97, 0, 43, 87, 96, 0, 53, 96, 224, 28, 128, 99, 221, 93, 82, 17, 20, 97, 0, 48, 87,
            91, 96, 0, 128, 253, 91, 97, 0, 74, 96, 4, 128, 54, 3, 129, 1, 144, 97, 0, 69, 145,
            144, 97, 0, 183, 86, 91, 97, 0, 96, 86, 91, 96, 64, 81, 97, 0, 87, 145, 144, 97, 0,
            243, 86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 96, 0, 96, 1, 130, 97, 0, 111, 145,
            144, 97, 1, 61, 86, 91, 144, 80, 145, 144, 80, 86, 91, 96, 0, 128, 253, 91, 96, 0, 99,
            255, 255, 255, 255, 130, 22, 144, 80, 145, 144, 80, 86, 91, 97, 0, 148, 129, 97, 0,
            123, 86, 91, 129, 20, 97, 0, 159, 87, 96, 0, 128, 253, 91, 80, 86, 91, 96, 0, 129, 53,
            144, 80, 97, 0, 177, 129, 97, 0, 139, 86, 91, 146, 145, 80, 80, 86, 91, 96, 0, 96, 32,
            130, 132, 3, 18, 21, 97, 0, 205, 87, 97, 0, 204, 97, 0, 118, 86, 91, 91, 96, 0, 97, 0,
            219, 132, 130, 133, 1, 97, 0, 162, 86, 91, 145, 80, 80, 146, 145, 80, 80, 86, 91, 97,
            0, 237, 129, 97, 0, 123, 86, 91, 130, 82, 80, 80, 86, 91, 96, 0, 96, 32, 130, 1, 144,
            80, 97, 1, 8, 96, 0, 131, 1, 132, 97, 0, 228, 86, 91, 146, 145, 80, 80, 86, 91, 127,
            78, 72, 123, 113, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 96, 0, 82, 96, 17, 96, 4, 82, 96, 36, 96, 0, 253, 91, 96, 0, 97, 1, 72,
            130, 97, 0, 123, 86, 91, 145, 80, 97, 1, 83, 131, 97, 0, 123, 86, 91, 146, 80, 130,
            130, 1, 144, 80, 99, 255, 255, 255, 255, 129, 17, 21, 97, 1, 111, 87, 97, 1, 110, 97,
            1, 14, 86, 91, 91, 146, 145, 80, 80, 86, 254, 162, 100, 105, 112, 102, 115, 88, 34, 18,
            32, 157, 56, 173, 41, 32, 104, 86, 178, 225, 128, 173, 18, 209, 171, 78, 95, 83, 125,
            108, 250, 78, 111, 65, 56, 240, 233, 227, 255, 7, 247, 146, 86, 100, 115, 111, 108, 99,
            120, 39, 48, 46, 56, 46, 49, 55, 45, 100, 101, 118, 101, 108, 111, 112, 46, 50, 48, 50,
            50, 46, 56, 46, 57, 43, 99, 111, 109, 109, 105, 116, 46, 98, 98, 49, 97, 56, 100, 102,
            57, 0, 88,
        ];

        (
            Self::new(&CODE, size, Some([221, 93, 82, 17])),
            inc_ref(size),
        )
    }

    // See go/examples/fib.go
    pub fn fib(size: u32) -> (Self, u32) {
        fn fib_ref(input: u32) -> u32 {
            if input <= 1 {
                1
            } else {
                fib_ref(input - 1) + fib_ref(input - 2)
            }
        }

        const CODE: [u8; 583] = [
            96, 128, 96, 64, 82, 52, 128, 21, 97, 0, 16, 87, 96, 0, 128, 253, 91, 80, 96, 4, 54,
            16, 97, 0, 43, 87, 96, 0, 53, 96, 224, 28, 128, 99, 249, 183, 199, 229, 20, 97, 0, 48,
            87, 91, 96, 0, 128, 253, 91, 97, 0, 74, 96, 4, 128, 54, 3, 129, 1, 144, 97, 0, 69, 145,
            144, 97, 0, 246, 86, 91, 97, 0, 96, 86, 91, 96, 64, 81, 97, 0, 87, 145, 144, 97, 1, 50,
            86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 96, 0, 96, 1, 130, 99, 255, 255, 255,
            255, 22, 17, 97, 0, 121, 87, 96, 1, 144, 80, 97, 0, 176, 86, 91, 97, 0, 142, 96, 2,
            131, 97, 0, 137, 145, 144, 97, 1, 124, 86, 91, 97, 0, 96, 86, 91, 97, 0, 163, 96, 1,
            132, 97, 0, 158, 145, 144, 97, 1, 124, 86, 91, 97, 0, 96, 86, 91, 97, 0, 173, 145, 144,
            97, 1, 180, 86, 91, 144, 80, 91, 145, 144, 80, 86, 91, 96, 0, 128, 253, 91, 96, 0, 99,
            255, 255, 255, 255, 130, 22, 144, 80, 145, 144, 80, 86, 91, 97, 0, 211, 129, 97, 0,
            186, 86, 91, 129, 20, 97, 0, 222, 87, 96, 0, 128, 253, 91, 80, 86, 91, 96, 0, 129, 53,
            144, 80, 97, 0, 240, 129, 97, 0, 202, 86, 91, 146, 145, 80, 80, 86, 91, 96, 0, 96, 32,
            130, 132, 3, 18, 21, 97, 1, 12, 87, 97, 1, 11, 97, 0, 181, 86, 91, 91, 96, 0, 97, 1,
            26, 132, 130, 133, 1, 97, 0, 225, 86, 91, 145, 80, 80, 146, 145, 80, 80, 86, 91, 97, 1,
            44, 129, 97, 0, 186, 86, 91, 130, 82, 80, 80, 86, 91, 96, 0, 96, 32, 130, 1, 144, 80,
            97, 1, 71, 96, 0, 131, 1, 132, 97, 1, 35, 86, 91, 146, 145, 80, 80, 86, 91, 127, 78,
            72, 123, 113, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 96, 0, 82, 96, 17, 96, 4, 82, 96, 36, 96, 0, 253, 91, 96, 0, 97, 1, 135,
            130, 97, 0, 186, 86, 91, 145, 80, 97, 1, 146, 131, 97, 0, 186, 86, 91, 146, 80, 130,
            130, 3, 144, 80, 99, 255, 255, 255, 255, 129, 17, 21, 97, 1, 174, 87, 97, 1, 173, 97,
            1, 77, 86, 91, 91, 146, 145, 80, 80, 86, 91, 96, 0, 97, 1, 191, 130, 97, 0, 186, 86,
            91, 145, 80, 97, 1, 202, 131, 97, 0, 186, 86, 91, 146, 80, 130, 130, 1, 144, 80, 99,
            255, 255, 255, 255, 129, 17, 21, 97, 1, 230, 87, 97, 1, 229, 97, 1, 77, 86, 91, 91,
            146, 145, 80, 80, 86, 254, 162, 100, 105, 112, 102, 115, 88, 34, 18, 32, 127, 211, 62,
            71, 233, 124, 229, 135, 27, 176, 84, 1, 230, 113, 2, 56, 175, 83, 90, 232, 174, 170,
            176, 19, 202, 154, 156, 41, 21, 43, 138, 27, 100, 115, 111, 108, 99, 120, 39, 48, 46,
            56, 46, 49, 55, 45, 100, 101, 118, 101, 108, 111, 112, 46, 50, 48, 50, 50, 46, 56, 46,
            57, 43, 99, 111, 109, 109, 105, 116, 46, 98, 98, 49, 97, 56, 100, 102, 57, 0, 88,
        ];

        (
            Self::new(&CODE, size, Some([249, 183, 199, 229])),
            fib_ref(size),
        )
    }

    // See go/examples/sha3.go
    pub fn sha3(size: u32) -> (Self, u32) {
        fn sha3_ref(input: u32) -> u32 {
            let mut data = [0; 32];
            for _ in 0..input {
                let mut hasher = Keccak256::new();
                hasher.update(data);
                let mut new_data = [0; 32];
                hasher.finalize_into((&mut new_data).into());
                data = new_data;
            }

            data[31] as u32
        }

        const CODE: [u8; 39] = [
            96, 4, 53, 91, 128, 21, 96, 24, 87, 96, 32, 96, 0, 32, 96, 0, 82, 96, 1, 144, 3, 96, 3,
            86, 91, 96, 0, 81, 96, 255, 22, 96, 0, 82, 96, 32, 96, 0, 243,
        ];

        (Self::new(&CODE, size, None), sha3_ref(size))
    }

    // See go/examples/arithmetic.go
    pub fn arithmetic(size: u32) -> (Self, u32) {
        fn arithmetic_ref(input: u32) -> u32 {
            let iterations = input;
            let mut result = 0;
            let mut i = 1;
            while i <= iterations {
                let i_squared = i * i;
                let i_cubed = i_squared * i;
                let i_mod3 = i % 3;
                result += i;
                result *= i;
                result += i_squared;
                result -= i;
                result /= i;
                result *= i_mod3 + 1;
                result += i_cubed;

                i += 1;
            }
            let max_u32 = i32::MAX as u32;
            result %= max_u32;
            result
        }

        const CODE: [u8; 483] = [
            96, 128, 96, 64, 82, 52, 128, 21, 97, 0, 16, 87, 96, 0, 128, 253, 91, 80, 96, 4, 54,
            16, 97, 0, 43, 87, 96, 0, 53, 96, 224, 28, 128, 99, 204, 130, 28, 9, 20, 97, 0, 48, 87,
            91, 96, 0, 128, 253, 91, 97, 0, 74, 96, 4, 128, 54, 3, 129, 1, 144, 97, 0, 69, 145,
            144, 97, 1, 39, 86, 91, 97, 0, 96, 86, 91, 96, 64, 81, 97, 0, 87, 145, 144, 97, 1, 99,
            86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 96, 0, 128, 96, 0, 144, 80, 96, 0, 96,
            1, 144, 80, 91, 131, 129, 17, 97, 0, 203, 87, 128, 130, 1, 145, 80, 128, 130, 2, 145,
            80, 128, 129, 2, 130, 1, 145, 80, 128, 130, 3, 145, 80, 128, 130, 129, 97, 0, 155, 87,
            97, 0, 154, 97, 1, 126, 86, 91, 91, 4, 145, 80, 96, 1, 96, 3, 130, 129, 97, 0, 177, 87,
            97, 0, 176, 97, 1, 126, 86, 91, 91, 6, 1, 130, 2, 145, 80, 128, 129, 130, 2, 2, 130, 1,
            145, 80, 128, 96, 1, 1, 144, 80, 97, 0, 110, 86, 91, 80, 99, 127, 255, 255, 255, 96, 3,
            11, 129, 129, 97, 0, 227, 87, 97, 0, 226, 97, 1, 126, 86, 91, 91, 6, 145, 80, 80, 145,
            144, 80, 86, 91, 96, 0, 128, 253, 91, 96, 0, 129, 144, 80, 145, 144, 80, 86, 91, 97, 1,
            4, 129, 97, 0, 241, 86, 91, 129, 20, 97, 1, 15, 87, 96, 0, 128, 253, 91, 80, 86, 91,
            96, 0, 129, 53, 144, 80, 97, 1, 33, 129, 97, 0, 251, 86, 91, 146, 145, 80, 80, 86, 91,
            96, 0, 96, 32, 130, 132, 3, 18, 21, 97, 1, 61, 87, 97, 1, 60, 97, 0, 236, 86, 91, 91,
            96, 0, 97, 1, 75, 132, 130, 133, 1, 97, 1, 18, 86, 91, 145, 80, 80, 146, 145, 80, 80,
            86, 91, 97, 1, 93, 129, 97, 0, 241, 86, 91, 130, 82, 80, 80, 86, 91, 96, 0, 96, 32,
            130, 1, 144, 80, 97, 1, 120, 96, 0, 131, 1, 132, 97, 1, 84, 86, 91, 146, 145, 80, 80,
            86, 91, 127, 78, 72, 123, 113, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 82, 96, 18, 96, 4, 82, 96, 36, 96, 0, 253, 254, 162,
            100, 105, 112, 102, 115, 88, 34, 18, 32, 71, 91, 29, 242, 120, 151, 218, 100, 32, 45,
            85, 243, 156, 193, 51, 53, 120, 218, 93, 130, 207, 72, 235, 54, 95, 224, 186, 245, 76,
            0, 227, 25, 100, 115, 111, 108, 99, 67, 0, 8, 20, 0, 51,
        ];

        (
            Self::new(&CODE, size, Some([204, 130, 28, 9])),
            arithmetic_ref(size),
        )
    }

    // See go/examples/memory.go
    pub fn memory(size: u32) -> (Self, u32) {
        fn memory_ref(input: u32) -> u32 {
            let mut values = vec![0; input as usize];
            #[allow(clippy::needless_range_loop)]
            for i in 0..values.len() {
                values[i] = i as u32;
            }
            let mut values_copy = vec![0; input as usize];
            #[allow(clippy::manual_memcpy)]
            for i in 0..values.len() {
                values_copy[i] = values[i];
            }
            if input > 0 {
                return values_copy[input as usize - 1];
            }
            input
        }

        const CODE: [u8; 741] = [
            96, 128, 96, 64, 82, 52, 128, 21, 97, 0, 16, 87, 96, 0, 128, 253, 91, 80, 96, 4, 54,
            16, 97, 0, 43, 87, 96, 0, 53, 96, 224, 28, 128, 99, 232, 138, 231, 129, 20, 97, 0, 48,
            87, 91, 96, 0, 128, 253, 91, 97, 0, 74, 96, 4, 128, 54, 3, 129, 1, 144, 97, 0, 69, 145,
            144, 97, 1, 250, 86, 91, 97, 0, 96, 86, 91, 96, 64, 81, 97, 0, 87, 145, 144, 97, 2, 54,
            86, 91, 96, 64, 81, 128, 145, 3, 144, 243, 91, 96, 0, 128, 130, 144, 80, 96, 0, 129,
            103, 255, 255, 255, 255, 255, 255, 255, 255, 129, 17, 21, 97, 0, 130, 87, 97, 0, 129,
            97, 2, 81, 86, 91, 91, 96, 64, 81, 144, 128, 130, 82, 128, 96, 32, 2, 96, 32, 1, 130,
            1, 96, 64, 82, 128, 21, 97, 0, 176, 87, 129, 96, 32, 1, 96, 32, 130, 2, 128, 54, 131,
            55, 128, 130, 1, 145, 80, 80, 144, 80, 91, 80, 144, 80, 96, 0, 91, 130, 129, 16, 21,
            97, 0, 233, 87, 128, 130, 130, 129, 81, 129, 16, 97, 0, 210, 87, 97, 0, 209, 97, 2,
            128, 86, 91, 91, 96, 32, 2, 96, 32, 1, 1, 129, 129, 82, 80, 80, 128, 96, 1, 1, 144, 80,
            97, 0, 182, 86, 91, 80, 96, 0, 130, 103, 255, 255, 255, 255, 255, 255, 255, 255, 129,
            17, 21, 97, 1, 6, 87, 97, 1, 5, 97, 2, 81, 86, 91, 91, 96, 64, 81, 144, 128, 130, 82,
            128, 96, 32, 2, 96, 32, 1, 130, 1, 96, 64, 82, 128, 21, 97, 1, 52, 87, 129, 96, 32, 1,
            96, 32, 130, 2, 128, 54, 131, 55, 128, 130, 1, 145, 80, 80, 144, 80, 91, 80, 144, 80,
            96, 0, 91, 131, 129, 16, 21, 97, 1, 135, 87, 130, 129, 129, 81, 129, 16, 97, 1, 85, 87,
            97, 1, 84, 97, 2, 128, 86, 91, 91, 96, 32, 2, 96, 32, 1, 1, 81, 130, 130, 129, 81, 129,
            16, 97, 1, 112, 87, 97, 1, 111, 97, 2, 128, 86, 91, 91, 96, 32, 2, 96, 32, 1, 1, 129,
            129, 82, 80, 80, 128, 96, 1, 1, 144, 80, 97, 1, 58, 86, 91, 80, 96, 0, 133, 19, 97, 1,
            150, 87, 132, 97, 1, 181, 86, 91, 128, 96, 1, 132, 3, 129, 81, 129, 16, 97, 1, 172, 87,
            97, 1, 171, 97, 2, 128, 86, 91, 91, 96, 32, 2, 96, 32, 1, 1, 81, 91, 147, 80, 80, 80,
            80, 145, 144, 80, 86, 91, 96, 0, 128, 253, 91, 96, 0, 129, 144, 80, 145, 144, 80, 86,
            91, 97, 1, 215, 129, 97, 1, 196, 86, 91, 129, 20, 97, 1, 226, 87, 96, 0, 128, 253, 91,
            80, 86, 91, 96, 0, 129, 53, 144, 80, 97, 1, 244, 129, 97, 1, 206, 86, 91, 146, 145, 80,
            80, 86, 91, 96, 0, 96, 32, 130, 132, 3, 18, 21, 97, 2, 16, 87, 97, 2, 15, 97, 1, 191,
            86, 91, 91, 96, 0, 97, 2, 30, 132, 130, 133, 1, 97, 1, 229, 86, 91, 145, 80, 80, 146,
            145, 80, 80, 86, 91, 97, 2, 48, 129, 97, 1, 196, 86, 91, 130, 82, 80, 80, 86, 91, 96,
            0, 96, 32, 130, 1, 144, 80, 97, 2, 75, 96, 0, 131, 1, 132, 97, 2, 39, 86, 91, 146, 145,
            80, 80, 86, 91, 127, 78, 72, 123, 113, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 82, 96, 65, 96, 4, 82, 96, 36, 96, 0, 253,
            91, 127, 78, 72, 123, 113, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
            0, 0, 0, 0, 0, 0, 0, 0, 96, 0, 82, 96, 50, 96, 4, 82, 96, 36, 96, 0, 253, 254, 162,
            100, 105, 112, 102, 115, 88, 34, 18, 32, 55, 22, 201, 17, 58, 164, 66, 83, 208, 141,
            239, 132, 60, 85, 127, 183, 93, 69, 47, 235, 22, 145, 238, 189, 134, 43, 116, 249, 59,
            152, 152, 104, 100, 115, 111, 108, 99, 67, 0, 8, 20, 0, 51,
        ];

        (
            Self::new(&CODE, size, Some([232, 138, 231, 129])),
            memory_ref(size),
        )
    }

    const fn analysis_code_len(max_len: usize, filler_len: usize) -> usize {
        let code_start_len = 10;
        let code_end_len = 6;
        let filler_repetitions = (max_len - code_start_len - code_end_len) / filler_len;
        code_start_len + filler_len * filler_repetitions + code_end_len
    }

    const fn build_analysis_code<const N: usize>(filler: &[u8]) -> [u8; N] {
        let code_start = [96, 4, 53, 96, 0, 82, 97, 255, 255, 86];
        let code_end = [91, 96, 32, 96, 0, 243];
        let mut code = [0; N];

        let mut i = 0;
        while i < code_start.len() {
            code[i] = code_start[i];
            i += 1;
        }

        let jumpdest_idx = ((N - code_end.len()) as u16).to_be_bytes();
        code[7] = jumpdest_idx[0];
        code[8] = jumpdest_idx[1];

        i = code_start.len();
        while i < N - code_end.len() {
            let mut j = 0;
            while j < filler.len() {
                code[i + j] = filler[j];
                j += 1;
            }
            i += filler.len();
        }

        i = 0;
        while i < code_end.len() {
            code[N - code_end.len() + i] = code_end[i];
            i += 1;
        }

        code
    }

    const LONG_MAX_LEN: usize = 0x6000;
    const SHORT_MAX_LEN: usize = 100;

    // See go/examples/analysis.go
    pub fn analysis(size: u32, short_code: &'static [u8], long_code: &'static [u8]) -> (Self, u32) {
        fn analysis_ref(input: u32) -> u32 {
            input
        }

        let code: &[u8] = match size as usize {
            Self::LONG_MAX_LEN => long_code,
            Self::SHORT_MAX_LEN => short_code,
            _ => panic!(
                "analysis only supports size {} or {}",
                Self::LONG_MAX_LEN,
                Self::SHORT_MAX_LEN
            ),
        };

        (
            Self::new(code, size, Some([204, 130, 28, 9])),
            analysis_ref(size),
        )
    }

    pub fn jumpdest_analysis(size: u32) -> (Self, u32) {
        const FILLER: [u8; 1] = [JUMPDEST];

        const LONG_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::LONG_MAX_LEN, FILLER.len());
        const LONG_CODE: [u8; LONG_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        const SHORT_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::SHORT_MAX_LEN, FILLER.len());
        const SHORT_CODE: [u8; SHORT_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        Self::analysis(size, &SHORT_CODE, &LONG_CODE)
    }

    pub fn stop_analysis(size: u32) -> (Self, u32) {
        const FILLER: [u8; 1] = [STOP];

        const LONG_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::LONG_MAX_LEN, FILLER.len());
        const LONG_CODE: [u8; LONG_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        const SHORT_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::SHORT_MAX_LEN, FILLER.len());
        const SHORT_CODE: [u8; SHORT_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        Self::analysis(size, &SHORT_CODE, &LONG_CODE)
    }

    pub fn push1_analysis(size: u32) -> (Self, u32) {
        const FILLER: [u8; 2] = [PUSH1, 0];

        const LONG_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::LONG_MAX_LEN, FILLER.len());
        const LONG_CODE: [u8; LONG_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        const SHORT_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::SHORT_MAX_LEN, FILLER.len());
        const SHORT_CODE: [u8; SHORT_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        Self::analysis(size, &SHORT_CODE, &LONG_CODE)
    }

    pub fn push32_analysis(size: u32) -> (Self, u32) {
        const FILLER: [u8; 33] = {
            let mut code = [0; 33];
            code[0] = PUSH32;
            code
        };

        const LONG_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::LONG_MAX_LEN, FILLER.len());
        const LONG_CODE: [u8; LONG_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        const SHORT_CODE_LEN: usize =
            RunArgs::analysis_code_len(RunArgs::SHORT_MAX_LEN, FILLER.len());
        const SHORT_CODE: [u8; SHORT_CODE_LEN] = RunArgs::build_analysis_code(&FILLER);

        Self::analysis(size, &SHORT_CODE, &LONG_CODE)
    }

    fn new(code: &'static [u8], size: u32, func: Option<[u8; 4]>) -> Self {
        let instance = Instance::default();
        let mut host = null_ptr_host_interface();
        host.get_tx_context = Some(get_tx_context_zeroed);

        let mut input = [0; 36];
        if let Some(func) = func {
            input[..4].copy_from_slice(&func);
        }
        input[32..].copy_from_slice(&size.to_be_bytes());

        let mut hasher = Keccak256::new();
        hasher.update(code);
        let mut code_hash = [0; 32];
        hasher.finalize_into((&mut code_hash).into());

        let message = MockExecutionMessage {
            input: Some(Box::leak(Box::from(input.as_slice()))),
            code_hash: Some(Box::leak(Box::new(Uint256 { bytes: code_hash }))),
            ..Default::default()
        };

        Self {
            instance,
            host,
            revision: Revision::EVMC_CANCUN,
            message: message.to_evmc_message(),
            code,
        }
    }
}

impl Drop for RunArgs {
    fn drop(&mut self) {
        if !self.message.input_data.is_null() {
            unsafe {
                let _ = Box::from_raw(std::slice::from_raw_parts_mut(
                    self.message.input_data as *mut u8,
                    self.message.input_size,
                ));
            }
        }
        if !self.message.code_hash.is_null() {
            unsafe {
                let _ = Box::from_raw(self.message.code_hash as *mut Uint256);
            }
        }
    }
}

pub fn run(args: &mut RunArgs) -> u32 {
    // SAFETY:
    // `host` and `message` are valid pointers since they are created from references. `context`
    // is a null pointer but this is allowed by the evmc interface as long as the host
    // interface does not require a valid pointer, which is not the case here.
    let result =
        args.instance
            .run_with_null_context(&args.host, args.revision, &args.message, args.code);
    assert_eq!(result.status_code, StatusCode::EVMC_SUCCESS);
    let output = result.output.unwrap();
    assert_eq!(output.len(), 32);
    u32::from_be_bytes(output[28..32].try_into().unwrap())
}

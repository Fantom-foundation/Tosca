use crate::EvmRs;

static EVM_RS_NAME: &'static str = "evmrs\0";
static EVM_RS_VERSION: &'static str = "0.1.0\0";

extern "C" fn __evmc_get_capabilities(
    instance: *mut ::evmc_vm::ffi::evmc_vm,
) -> ::evmc_vm::ffi::evmc_capabilities_flagset {
    3
}

extern "C" fn __evmc_set_option(
    instance: *mut ::evmc_vm::ffi::evmc_vm,
    key: *const std::os::raw::c_char,
    value: *const std::os::raw::c_char,
) -> ::evmc_vm::ffi::evmc_set_option_result {
    use std::ffi::CStr;

    use evmc_vm::{EvmcVm, SetOptionError};

    assert!(!instance.is_null());

    if key.is_null() {
        return ::evmc_vm::ffi::evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME;
    }

    let key = unsafe { CStr::from_ptr(key) };
    let key = match key.to_str() {
        Ok(k) => k,
        Err(e) => return ::evmc_vm::ffi::evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME,
    };

    let value = if !value.is_null() {
        unsafe { CStr::from_ptr(value) }
    } else {
        unsafe { CStr::from_bytes_with_nul_unchecked(&[0]) }
    };

    let value = match value.to_str() {
        Ok(k) => k,
        Err(e) => return ::evmc_vm::ffi::evmc_set_option_result::EVMC_SET_OPTION_INVALID_VALUE,
    };

    let mut container = unsafe {
        // Acquire ownership from EVMC.
        ::evmc_vm::EvmcContainer::<EvmRs>::from_ffi_pointer(instance)
    };

    let result = match container.set_option(key, value) {
        Ok(()) => ::evmc_vm::ffi::evmc_set_option_result::EVMC_SET_OPTION_SUCCESS,
        Err(SetOptionError::InvalidKey) => {
            ::evmc_vm::ffi::evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME
        }
        Err(SetOptionError::InvalidValue) => {
            ::evmc_vm::ffi::evmc_set_option_result::EVMC_SET_OPTION_INVALID_VALUE
        }
    };

    unsafe {
        // Release ownership to EVMC.
        ::evmc_vm::EvmcContainer::into_ffi_pointer(container);
    }

    result
}

#[no_mangle]
pub extern "C" fn evmc_create_evmrs() -> *const ::evmc_vm::ffi::evmc_vm {
    let new_instance = ::evmc_vm::ffi::evmc_vm {
        abi_version: ::evmc_vm::ffi::EVMC_ABI_VERSION as i32,
        destroy: Some(__evmc_destroy),
        execute: Some(__evmc_execute),
        get_capabilities: Some(__evmc_get_capabilities),
        set_option: Some(__evmc_set_option),
        name: unsafe {
            ::std::ffi::CStr::from_bytes_with_nul_unchecked(EVM_RS_NAME.as_bytes()).as_ptr()
        },
        version: unsafe {
            ::std::ffi::CStr::from_bytes_with_nul_unchecked(EVM_RS_VERSION.as_bytes()).as_ptr()
        },
    };

    let container = ::evmc_vm::EvmcContainer::<EvmRs>::new(new_instance);

    unsafe {
        // Release ownership to EVMC.
        ::evmc_vm::EvmcContainer::into_ffi_pointer(container)
    }
}

extern "C" fn __evmc_destroy(instance: *mut ::evmc_vm::ffi::evmc_vm) {
    if instance.is_null() {
        // This is an irrecoverable error that violates the EVMC spec.
        std::process::abort();
    }
    unsafe {
        // Acquire ownership from EVMC. This will deallocate it also at the end of the scope.
        ::evmc_vm::EvmcContainer::<EvmRs>::from_ffi_pointer(instance);
    }
}

extern "C" fn __evmc_execute(
    instance: *mut ::evmc_vm::ffi::evmc_vm,
    host: *const ::evmc_vm::ffi::evmc_host_interface,
    context: *mut ::evmc_vm::ffi::evmc_host_context,
    revision: ::evmc_vm::ffi::evmc_revision,
    msg: *const ::evmc_vm::ffi::evmc_message,
    code: *const u8,
    code_size: usize,
) -> ::evmc_vm::ffi::evmc_result {
    use evmc_vm::EvmcVm;

    // TODO: context is optional in case of the "precompiles" capability
    if instance.is_null() || msg.is_null() || (code.is_null() && code_size != 0) {
        // These are irrecoverable errors that violate the EVMC spec.
        std::process::abort();
    }

    assert!(!instance.is_null());
    assert!(!msg.is_null());

    let execution_message: ::evmc_vm::ExecutionMessage =
        unsafe { msg.as_ref().expect("EVMC message is null").into() };

    let empty_code = [0u8; 0];
    let code_ref: &[u8] = if code.is_null() {
        assert_eq!(code_size, 0);
        &empty_code
    } else {
        unsafe { ::std::slice::from_raw_parts(code, code_size) }
    };

    let container = unsafe {
        // Acquire ownership from EVMC.
        ::evmc_vm::EvmcContainer::<EvmRs>::from_ffi_pointer(instance)
    };

    let result = ::std::panic::catch_unwind(|| {
        if host.is_null() {
            container.execute(revision, code_ref, &execution_message, None)
        } else {
            let mut execution_context = unsafe {
                ::evmc_vm::ExecutionContext::new(host.as_ref().expect("EVMC host is null"), context)
            };
            container.execute(
                revision,
                code_ref,
                &execution_message,
                Some(&mut execution_context),
            )
        }
    });

    let result = if result.is_err() {
        // Consider a panic an internal error.
        ::evmc_vm::ExecutionResult::new(
            ::evmc_vm::ffi::evmc_status_code::EVMC_INTERNAL_ERROR,
            0,
            0,
            None,
        )
    } else {
        result.unwrap()
    };

    unsafe {
        // Release ownership to EVMC.
        ::evmc_vm::EvmcContainer::into_ffi_pointer(container);
    }

    result.into()
}

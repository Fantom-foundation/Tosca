use std::{
    ffi::{c_char, CStr},
    panic, process, slice,
};

use evmc_vm::{
    ffi::{
        evmc_capabilities, evmc_capabilities_flagset, evmc_host_context, evmc_host_interface,
        evmc_message, evmc_result, evmc_revision, evmc_set_option_result, evmc_status_code,
        evmc_vm as evmc_vm_t, EVMC_ABI_VERSION,
    },
    EvmcContainer, EvmcVm, ExecutionContext, ExecutionMessage, ExecutionResult, SetOptionError,
};

use crate::evmc::EvmRs;

static EVM_RS_NAME: &str = "evmrs\0";
static EVM_RS_VERSION: &str = "0.1.0\0";

pub const EVMC_CAPABILITY: evmc_capabilities = evmc_capabilities::EVMC_CAPABILITY_EVM1;

extern "C" fn __evmc_get_capabilities(_instance: *mut evmc_vm_t) -> evmc_capabilities_flagset {
    EVMC_CAPABILITY as evmc_capabilities_flagset
}

extern "C" fn __evmc_set_option(
    instance: *mut evmc_vm_t,
    key: *const c_char,
    value: *const c_char,
) -> evmc_set_option_result {
    assert!(!instance.is_null());

    if key.is_null() {
        return evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME;
    }

    let key = unsafe { CStr::from_ptr(key) };
    let Ok(key) = key.to_str() else {
        return evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME;
    };

    let value = if !value.is_null() {
        unsafe { CStr::from_ptr(value) }
    } else {
        unsafe { CStr::from_bytes_with_nul_unchecked(&[0]) }
    };

    let Ok(value) = value.to_str() else {
        return evmc_set_option_result::EVMC_SET_OPTION_INVALID_VALUE;
    };

    let mut container = unsafe {
        // Acquire ownership from EVMC.
        EvmcContainer::<EvmRs>::from_ffi_pointer(instance)
    };

    let result = match container.set_option(key, value) {
        Ok(()) => evmc_set_option_result::EVMC_SET_OPTION_SUCCESS,
        Err(SetOptionError::InvalidKey) => evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME,
        Err(SetOptionError::InvalidValue) => evmc_set_option_result::EVMC_SET_OPTION_INVALID_VALUE,
    };

    // Release ownership to EVMC.
    unsafe {
        EvmcContainer::into_ffi_pointer(container);
    }

    result
}

#[no_mangle]
pub extern "C" fn evmc_create_evmrs() -> *const evmc_vm_t {
    let new_instance = evmc_vm_t {
        abi_version: EVMC_ABI_VERSION as i32,
        destroy: Some(__evmc_destroy),
        execute: Some(__evmc_execute),
        get_capabilities: Some(__evmc_get_capabilities),
        set_option: Some(__evmc_set_option),
        name: unsafe { CStr::from_bytes_with_nul_unchecked(EVM_RS_NAME.as_bytes()).as_ptr() },
        version: unsafe { CStr::from_bytes_with_nul_unchecked(EVM_RS_VERSION.as_bytes()).as_ptr() },
    };

    let container = EvmcContainer::<EvmRs>::new(new_instance);

    // Release ownership to EVMC.
    unsafe { EvmcContainer::into_ffi_pointer(container) }
}

extern "C" fn __evmc_destroy(instance: *mut evmc_vm_t) {
    if instance.is_null() {
        // This is an irrecoverable error that violates the EVMC spec.
        process::abort();
    }
    unsafe {
        // Acquire ownership from EVMC. This will deallocate it also at the end of the scope.
        EvmcContainer::<EvmRs>::from_ffi_pointer(instance);
    }
}

extern "C" fn __evmc_execute(
    instance: *mut evmc_vm_t,
    host: *const evmc_host_interface,
    context: *mut evmc_host_context,
    revision: evmc_revision,
    message: *const evmc_message,
    code: *const u8,
    code_size: usize,
) -> evmc_result {
    if instance.is_null()
        || (host.is_null() && EVMC_CAPABILITY != evmc_capabilities::EVMC_CAPABILITY_PRECOMPILES)
        || message.is_null()
        || (code.is_null() && code_size != 0)
    {
        // These are irrecoverable errors that violate the EVMC spec.
        std::process::abort();
    }

    let execution_message: ExecutionMessage = unsafe {
        // SAFETY:
        // message is not null
        (&*message).into()
    };

    let code_ref = if code.is_null() {
        assert_eq!(code_size, 0);
        &[0u8; 0]
    } else {
        // SAFETY:
        // code is not null and code size > 0
        unsafe { slice::from_raw_parts(code, code_size) }
    };

    let container = unsafe {
        // Acquire ownership from EVMC.
        EvmcContainer::<EvmRs>::from_ffi_pointer(instance)
    };

    let result = panic::catch_unwind(|| {
        let mut execution_context = if host.is_null() {
            None
        } else {
            let execution_context = unsafe {
                // SAFETY:
                // host is not null
                ExecutionContext::new(&*host, context)
            };
            Some(execution_context)
        };

        container.execute(
            revision,
            code_ref,
            &execution_message,
            execution_context.as_mut(),
        )
    });

    // Release ownership to EVMC.
    unsafe {
        EvmcContainer::into_ffi_pointer(container);
    }

    result
        .unwrap_or(ExecutionResult::new(
            evmc_status_code::EVMC_INTERNAL_ERROR,
            0,
            0,
            None,
        ))
        .into()
}

#[cfg(test)]
mod tests {
    use evmc_vm::ffi::{evmc_capabilities_flagset, evmc_set_option_result, evmc_vm as evmc_vm_t};

    use crate::ffi::{
        evmc_vm::{__evmc_destroy, __evmc_get_capabilities, __evmc_set_option, evmc_create_evmrs},
        EVMC_CAPABILITY,
    };

    #[test]
    fn create_destroy() {
        let vm = evmc_create_evmrs() as *mut evmc_vm_t;
        assert_eq!(
            __evmc_get_capabilities(vm),
            EVMC_CAPABILITY as evmc_capabilities_flagset
        );
        assert_eq!(
            __evmc_set_option(vm, std::ptr::null(), std::ptr::null()),
            evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME,
        );
        assert_eq!(
            __evmc_set_option(vm, c"\xF0\x90\x80".as_ptr(), std::ptr::null()),
            evmc_set_option_result::EVMC_SET_OPTION_INVALID_NAME,
        );
        assert_eq!(
            __evmc_set_option(vm, c"key".as_ptr(), std::ptr::null()),
            evmc_set_option_result::EVMC_SET_OPTION_SUCCESS,
        );
        assert_eq!(
            __evmc_set_option(vm, c"key".as_ptr(), c"\xF0\x90\x80".as_ptr()),
            evmc_set_option_result::EVMC_SET_OPTION_INVALID_VALUE,
        );
        assert_eq!(
            __evmc_set_option(vm, c"key".as_ptr(), c"value".as_ptr()),
            evmc_set_option_result::EVMC_SET_OPTION_SUCCESS,
        );
        __evmc_destroy(vm);
    }
}

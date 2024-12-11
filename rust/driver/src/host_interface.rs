use common::evmc_vm::ffi::evmc_host_interface;

#[cfg(feature = "mock")]
mod mock_callbacks {
    use std::{ffi, slice};

    use common::{
        evmc_vm::{
            ffi::{evmc_message, evmc_result, evmc_tx_context},
            AccessStatus, Address, StorageStatus, Uint256,
        },
        ExecutionContextTrait, MockExecutionContextTrait,
    };

    pub extern "C" fn account_exists(context: *mut ffi::c_void, addr: *const Address) -> bool {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        mock.account_exists(addr)
    }

    pub extern "C" fn get_storage(
        context: *mut ffi::c_void,
        addr: *const Address,
        key: *const Uint256,
    ) -> Uint256 {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let key = unsafe { &*key };
        mock.get_storage(addr, key)
    }

    pub extern "C" fn set_storage(
        context: *mut ffi::c_void,
        addr: *const Address,
        key: *const Uint256,
        value: *const Uint256,
    ) -> StorageStatus {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let key = unsafe { &*key };
        let value = unsafe { &*value };
        mock.set_storage(addr, key, value)
    }

    pub extern "C" fn get_balance(context: *mut ffi::c_void, addr: *const Address) -> Uint256 {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        mock.get_balance(addr)
    }

    pub extern "C" fn get_code_size(context: *mut ffi::c_void, addr: *const Address) -> usize {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        mock.get_code_size(addr)
    }

    pub extern "C" fn get_code_hash(context: *mut ffi::c_void, addr: *const Address) -> Uint256 {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        mock.get_code_hash(addr)
    }

    pub extern "C" fn copy_code(
        context: *mut ffi::c_void,
        addr: *const Address,
        code_offset: usize,
        buffer: *mut u8,
        buffer_len: usize,
    ) -> usize {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let buffer = unsafe { slice::from_raw_parts_mut(buffer, buffer_len) };
        mock.copy_code(addr, code_offset, buffer)
    }

    pub extern "C" fn selfdestruct(
        context: *mut ffi::c_void,
        addr: *const Address,
        beneficiary: *const Address,
    ) -> bool {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let beneficiary = unsafe { &*beneficiary };
        mock.selfdestruct(addr, beneficiary)
    }

    pub unsafe extern "C" fn call(
        context: *mut ffi::c_void,
        message: *const evmc_message,
    ) -> evmc_result {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let message = (unsafe { &*message }).into();
        mock.call(&message).into()
    }

    pub unsafe extern "C" fn get_tx_context(context: *mut ffi::c_void) -> evmc_tx_context {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        *mock.get_tx_context()
    }

    pub extern "C" fn get_block_hash(context: *mut ffi::c_void, num: i64) -> Uint256 {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        mock.get_block_hash(num)
    }

    pub unsafe extern "C" fn emit_log(
        context: *mut ffi::c_void,
        addr: *const Address,
        data: *const u8,
        data_len: usize,
        topic: *const Uint256,
        topic_len: usize,
    ) {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let data = unsafe { slice::from_raw_parts(data, data_len) };
        let topic = unsafe { slice::from_raw_parts(topic, topic_len) };
        mock.emit_log(addr, data, topic);
    }

    pub extern "C" fn access_account(
        context: *mut ffi::c_void,
        addr: *const Address,
    ) -> AccessStatus {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        mock.access_account(addr)
    }

    pub extern "C" fn access_storage(
        context: *mut ffi::c_void,
        addr: *const Address,
        key: *const Uint256,
    ) -> AccessStatus {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let key = unsafe { &*key };
        mock.access_storage(addr, key)
    }

    pub extern "C" fn get_transient_storage(
        context: *mut ffi::c_void,
        addr: *const Address,
        key: *const Uint256,
    ) -> Uint256 {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let key = unsafe { &*key };
        mock.get_transient_storage(addr, key)
    }

    pub unsafe extern "C" fn set_transient_storage(
        context: *mut ffi::c_void,
        addr: *const Address,
        key: *const Uint256,
        value: *const Uint256,
    ) {
        let mock = unsafe { &mut *(context as *mut MockExecutionContextTrait) };
        let addr = unsafe { &*addr };
        let key = unsafe { &*key };
        let value = unsafe { &*value };
        mock.set_transient_storage(addr, key, value);
    }
}

#[cfg(feature = "mock")]
pub fn mocked_host_interface() -> evmc_host_interface {
    use mock_callbacks::*;
    evmc_host_interface {
        account_exists: Some(account_exists),
        get_storage: Some(get_storage),
        set_storage: Some(set_storage),
        get_balance: Some(get_balance),
        get_code_size: Some(get_code_size),
        get_code_hash: Some(get_code_hash),
        copy_code: Some(copy_code),
        selfdestruct: Some(selfdestruct),
        call: Some(call),
        get_tx_context: Some(get_tx_context),
        get_block_hash: Some(get_block_hash),
        emit_log: Some(emit_log),
        access_account: Some(access_account),
        access_storage: Some(access_storage),
        get_transient_storage: Some(get_transient_storage),
        set_transient_storage: Some(set_transient_storage),
    }
}

pub fn null_ptr_host_interface() -> evmc_host_interface {
    evmc_host_interface {
        account_exists: None,
        get_storage: None,
        set_storage: None,
        get_balance: None,
        get_code_size: None,
        get_code_hash: None,
        copy_code: None,
        selfdestruct: None,
        call: None,
        get_tx_context: None,
        get_block_hash: None,
        emit_log: None,
        access_account: None,
        access_storage: None,
        get_transient_storage: None,
        set_transient_storage: None,
    }
}

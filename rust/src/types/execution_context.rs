use evmc_vm::{
    AccessStatus, Address, ExecutionContext, ExecutionMessage, ExecutionResult, ExecutionTxContext,
    StorageStatus, Uint256,
};

#[cfg_attr(feature = "mock", mockall::automock)]
pub trait ExecutionContextTrait {
    /// Retrieve the transaction context.
    fn get_tx_context(&mut self) -> &ExecutionTxContext;

    /// Check if an account exists.
    fn account_exists(&self, address: &Address) -> bool;

    /// Read from a storage key.
    fn get_storage(&self, address: &Address, key: &Uint256) -> Uint256;

    // Set value of a storage key.
    fn set_storage(&mut self, address: &Address, key: &Uint256, value: &Uint256) -> StorageStatus;

    /// Get balance of an account.
    fn get_balance(&self, address: &Address) -> Uint256;

    /// Get code size of an account.
    fn get_code_size(&self, address: &Address) -> usize;

    /// Get code hash of an account.
    fn get_code_hash(&self, address: &Address) -> Uint256;

    /// Copy code of an account.
    fn copy_code(&self, address: &Address, code_offset: usize, buffer: &mut [u8]) -> usize;

    /// Self-destruct the current account.
    fn selfdestruct(&mut self, address: &Address, beneficiary: &Address) -> bool;

    /// Call to another account.
    #[cfg(not(feature = "custom-evmc"))]
    fn call(&mut self, message: &ExecutionMessage) -> ExecutionResult;
    #[cfg(feature = "custom-evmc")]
    #[allow(clippy::needless_lifetimes)] // this is a bug in clippy
    fn call<'a>(&mut self, message: &ExecutionMessage<'a>) -> ExecutionResult;

    /// Get block hash of an account.
    fn get_block_hash(&self, num: i64) -> Uint256;

    /// Emit a log.
    fn emit_log(&mut self, address: &Address, data: &[u8], topics: &[Uint256]);

    /// Access an account.
    fn access_account(&mut self, address: &Address) -> AccessStatus;

    /// Access a storage key.
    fn access_storage(&mut self, address: &Address, key: &Uint256) -> AccessStatus;

    /// Read from a transient storage key.
    fn get_transient_storage(&self, address: &Address, key: &Uint256) -> Uint256;

    /// Set value of a transient storage key.
    fn set_transient_storage(&mut self, address: &Address, key: &Uint256, value: &Uint256);
}

impl ExecutionContextTrait for ExecutionContext<'_> {
    fn get_tx_context(&mut self) -> &ExecutionTxContext {
        ExecutionContext::get_tx_context(self)
    }

    fn account_exists(&self, address: &Address) -> bool {
        self.account_exists(address)
    }

    fn get_storage(&self, address: &Address, key: &Uint256) -> Uint256 {
        self.get_storage(address, key)
    }

    fn set_storage(&mut self, address: &Address, key: &Uint256, value: &Uint256) -> StorageStatus {
        self.set_storage(address, key, value)
    }

    fn get_balance(&self, address: &Address) -> Uint256 {
        self.get_balance(address)
    }

    fn get_code_size(&self, address: &Address) -> usize {
        self.get_code_size(address)
    }

    fn get_code_hash(&self, address: &Address) -> Uint256 {
        self.get_code_hash(address)
    }

    fn copy_code(&self, address: &Address, code_offset: usize, buffer: &mut [u8]) -> usize {
        self.copy_code(address, code_offset, buffer)
    }

    fn selfdestruct(&mut self, address: &Address, beneficiary: &Address) -> bool {
        self.selfdestruct(address, beneficiary)
    }

    fn call(&mut self, message: &ExecutionMessage) -> ExecutionResult {
        self.call(message)
    }

    fn get_block_hash(&self, num: i64) -> Uint256 {
        self.get_block_hash(num)
    }

    fn emit_log(&mut self, address: &Address, data: &[u8], topics: &[Uint256]) {
        self.emit_log(address, data, topics);
    }

    fn access_account(&mut self, address: &Address) -> AccessStatus {
        self.access_account(address)
    }

    fn access_storage(&mut self, address: &Address, key: &Uint256) -> AccessStatus {
        self.access_storage(address, key)
    }

    fn get_transient_storage(&self, address: &Address, key: &Uint256) -> Uint256 {
        self.get_transient_storage(address, key)
    }

    fn set_transient_storage(&mut self, address: &Address, key: &Uint256, value: &Uint256) {
        self.set_transient_storage(address, key, value);
    }
}

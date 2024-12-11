//! This module holds the [ExecutionTxContext].
//! This type is the same as [evmc_vm::ExecutionTxContext] except that the pointer and length fields
//! are converted to slices.
//!
//! Ideally this would be done already in [evmc_vm].
use std::slice;

use common::evmc_vm::{
    self,
    ffi::{evmc_tx_context, evmc_tx_initcode},
    Address, Uint256,
};

/// EVMC transaction context.
#[derive(Debug, Copy, Clone, Hash, PartialEq)]
pub struct ExecutionTxContext<'a> {
    /// The transaction gas price.
    pub tx_gas_price: Uint256,
    /// The transaction origin account.
    pub tx_origin: Address,
    /// The miner of the block.
    pub block_coinbase: Address,
    /// The block number.
    pub block_number: i64,
    /// The block timestamp.
    pub block_timestamp: i64,
    /// The block gas limit.
    pub block_gas_limit: i64,
    /// The block previous RANDAO (EIP-4399).
    pub block_prev_randao: Uint256,
    /// The blockchain's ChainID.
    pub chain_id: Uint256,
    /// The block base fee per gas (EIP-1559, EIP-3198).
    pub block_base_fee: Uint256,
    /// The blob base fee (EIP-7516).
    pub blob_base_fee: Uint256,
    /// The array of blob hashes (EIP-4844).
    pub blob_hashes: &'a [Uint256],
    /// The array of transaction initcodes (TXCREATE).
    pub initcodes: &'a [evmc_tx_initcode],
}

impl<'a> From<&'a evmc_vm::ExecutionTxContext> for ExecutionTxContext<'a> {
    fn from(context: &'a evmc_tx_context) -> Self {
        let blob_hashes = if context.blob_hashes.is_null() || context.blob_hashes_count == 0 {
            &[]
        } else {
            // SAFETY:
            // `context.blob_hashes` is not null an `context.blob_hashes_count > 0`
            unsafe { slice::from_raw_parts(context.blob_hashes, context.blob_hashes_count) }
        };
        let initcodes = if context.initcodes.is_null() || context.initcodes_count == 0 {
            &[]
        } else {
            // SAFETY:
            // `context.initcodes` is not null an `context.initcodes_count > 0`
            unsafe { slice::from_raw_parts(context.initcodes, context.initcodes_count) }
        };
        ExecutionTxContext {
            tx_gas_price: context.tx_gas_price,
            tx_origin: context.tx_origin,
            block_coinbase: context.block_coinbase,
            block_number: context.block_number,
            block_timestamp: context.block_timestamp,
            block_gas_limit: context.block_gas_limit,
            block_prev_randao: context.block_prev_randao,
            chain_id: context.chain_id,
            block_base_fee: context.block_base_fee,
            blob_base_fee: context.blob_base_fee,
            blob_hashes,
            initcodes,
        }
    }
}

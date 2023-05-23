#pragma once

#include <array>
#include <cstdint>

#include <ethash/keccak.hpp>
#include <evmc/evmc.hpp>
#include <intx/intx.hpp>

#include <tosca/macro_utils.h>

#include "evmzero.h"
#include "evmzero_context_utils.h"
#include "evmzero_debug_utils.h"

namespace tosca::evmzero::op {

TOSCA_FORCEINLINE inline void stop(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  ctx.state = RunState::kDone;
}

TOSCA_FORCEINLINE inline void add(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a + b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void mul(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a * b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void sub(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a - b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void div(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  if (b == 0)
    StackPush(ctx, 0);
  else
    StackPush(ctx, a / b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void sdiv(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  if (b == 0)
    StackPush(ctx, 0);
  else
    StackPush(ctx, intx::sdivrem(a, b).quot);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void mod(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  if (b == 0)
    StackPush(ctx, 0);
  else
    StackPush(ctx, a % b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void smod(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  if (b == 0)
    StackPush(ctx, 0);
  else
    StackPush(ctx, intx::sdivrem(a, b).rem);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void addmod(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 3)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 8)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  uint256_t N = StackPop(ctx);
  if (N == 0)
    StackPush(ctx, 0);
  else
    StackPush(ctx, intx::addmod(a, b, N));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void mulmod(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 3)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 8)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  uint256_t N = StackPop(ctx);
  if (N == 0)
    StackPush(ctx, 0);
  else
    StackPush(ctx, intx::mulmod(a, b, N));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void exp(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 10)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t exponent = StackPop(ctx);
  if (!ApplyGasCost(ctx, 50 * intx::count_significant_bytes(exponent))) [[unlikely]]
    return;
  StackPush(ctx, intx::exp(a, exponent));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void signextend(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5)) [[unlikely]]
    return;

  uint8_t leading_byte_index = static_cast<uint8_t>(StackPop(ctx));
  if (leading_byte_index > 31) {
    leading_byte_index = 31;
  }

  uint256_t value = StackPop(ctx);

  bool is_negative = ToByteArrayLe(value)[leading_byte_index] & 0b1000'0000;
  if (is_negative) {
    auto mask = kUint256Max << (8 * (leading_byte_index + 1));
    StackPush(ctx, mask | value);
  } else {
    auto mask = kUint256Max >> (8 * (31 - leading_byte_index));
    StackPush(ctx, mask & value);
  }

  ctx.pc++;
}

TOSCA_FORCEINLINE inline void lt(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a < b ? 1 : 0);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void gt(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a > b ? 1 : 0);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void slt(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, intx::slt(a, b) ? 1 : 0);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void sgt(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, intx::slt(b, a) ? 1 : 0);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void eq(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a == b ? 1 : 0);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void iszero(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t val = StackPop(ctx);
  StackPush(ctx, val == 0);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void bit_and(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a & b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void bit_or(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a | b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void bit_xor(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  uint256_t b = StackPop(ctx);
  StackPush(ctx, a ^ b);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void bit_not(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t a = StackPop(ctx);
  StackPush(ctx, ~a);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void byte(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t offset = StackPop(ctx);
  uint256_t x = StackPop(ctx);
  if (offset < 32) {
    // Offset starts at most significant byte.
    StackPush(ctx, ToByteArrayLe(x)[31 - static_cast<uint8_t>(offset)]);
  } else {
    StackPush(ctx, 0);
  }
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void shl(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t shift = StackPop(ctx);
  uint256_t value = StackPop(ctx);
  StackPush(ctx, value << shift);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void shr(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t shift = StackPop(ctx);
  uint256_t value = StackPop(ctx);
  StackPush(ctx, value >> shift);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void sar(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  uint256_t shift = StackPop(ctx);
  uint256_t value = StackPop(ctx);
  const bool is_negative = ToByteArrayLe(value)[31] & 0b1000'0000;

  if (shift > 31) {
    shift = 31;
  }

  value >>= shift;

  if (is_negative) {
    value |= (kUint256Max << (31 - shift));
  }

  StackPush(ctx, value);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void sha3(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 30)) [[unlikely]]
    return;
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t size = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t new_mem_cost = DynamicMemoryCost(offset + size, ctx.current_mem_cost);
  auto minimum_word_size = (size + 31) / 32;
  if (!ApplyGasCost(ctx, 6 * minimum_word_size + new_mem_cost)) [[unlikely]]
    return;

  GrowMemory(ctx, offset + size);

  auto hash = ethash::keccak256(ctx.memory.data() + offset, size);
  StackPush(ctx, ToUint256(hash));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void address(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.message->recipient));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void balance(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;

  auto address = ToEvmcAddress(StackPop(ctx));

  uint64_t dynamic_gas_cost = 2600;
  if (ctx.host.access_account(address) == EVMC_ACCESS_WARM) {
    dynamic_gas_cost = 200;
  }
  if (!ApplyGasCost(ctx, dynamic_gas_cost)) [[unlikely]]
    return;

  StackPush(ctx, ToUint256(ctx.host.get_balance(address)));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void origin(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.host.get_tx_context().tx_origin));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void caller(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.message->sender));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void callvalue(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.message->value));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void calldataload(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;

  uint64_t offset = static_cast<uint64_t>(StackPop(ctx));

  evmc::bytes32 value{};
  if (offset < ctx.message->input_size) {
    auto end = std::min<uint64_t>(ctx.message->input_size - offset, 32);
    for (unsigned i = 0; i < end; ++i) {
      value.bytes[i] = ctx.message->input_data[i + offset];
    }
  }

  StackPush(ctx, ToUint256(value));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void calldatasize(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.message->input_size);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void calldatacopy(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 3)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  const uint64_t destoffset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t size = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t minimum_word_size = (size + 31) / 32;
  const uint64_t new_mem_cost = DynamicMemoryCost(destoffset + size, ctx.current_mem_cost);
  if (!ApplyGasCost(ctx, 3 * minimum_word_size + new_mem_cost)) [[unlikely]]
    return;

  GrowMemory(ctx, destoffset + size);

  std::copy_n(ctx.message->input_data + offset, std::min(size, ctx.message->input_size - offset),
              ctx.memory.data() + destoffset);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void codesize(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.code.size());
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void codecopy(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 3)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  const uint64_t destoffset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t size = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t minimum_word_size = (size + 31) / 32;
  const uint64_t new_mem_cost = DynamicMemoryCost(destoffset + size, ctx.current_mem_cost);
  if (!ApplyGasCost(ctx, 3 * minimum_word_size + new_mem_cost)) [[unlikely]]
    return;

  GrowMemory(ctx, destoffset + size);

  std::copy_n(ctx.code.begin() + static_cast<ptrdiff_t>(offset), size,
              ctx.memory.begin() + static_cast<ptrdiff_t>(destoffset));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void gasprice(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.host.get_tx_context().tx_gas_price));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void extcodesize(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;

  auto address = ToEvmcAddress(StackPop(ctx));

  uint64_t dynamic_gas_cost = 2600;
  if (ctx.host.access_account(address) == EVMC_ACCESS_WARM) {
    dynamic_gas_cost = 100;
  }
  if (!ApplyGasCost(ctx, dynamic_gas_cost)) [[unlikely]]
    return;

  StackPush(ctx, ctx.host.get_code_size(address));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void extcodecopy(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 4)) [[unlikely]]
    return;

  auto address = ToEvmcAddress(StackPop(ctx));
  const uint64_t destoffset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t size = static_cast<uint64_t>(StackPop(ctx));

  const uint64_t minimum_word_size = (size + 31) / 32;
  const uint64_t new_mem_cost = DynamicMemoryCost(destoffset + size, ctx.current_mem_cost);
  uint64_t address_access_cost = 2600;
  if (ctx.host.access_account(address) == EVMC_ACCESS_WARM) {
    address_access_cost = 100;
  }
  const uint64_t dynamic_gas_cost = 3 * minimum_word_size + new_mem_cost + address_access_cost;
  if (!ApplyGasCost(ctx, dynamic_gas_cost)) [[unlikely]]
    return;

  GrowMemory(ctx, destoffset + size);

  ctx.host.copy_code(address, offset, ctx.memory.data() + destoffset, size);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void returndatasize(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.return_data.size());
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void returndatacopy(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 3)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;

  const uint64_t destoffset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t size = static_cast<uint64_t>(StackPop(ctx));

  const uint64_t minimum_word_size = (size + 31) / 32;
  const uint64_t new_mem_cost = DynamicMemoryCost(destoffset + size, ctx.current_mem_cost);
  if (!ApplyGasCost(ctx, 3 * minimum_word_size + new_mem_cost)) [[unlikely]]
    return;

  GrowMemory(ctx, destoffset + size);

  uint64_t bytes_to_copy = 0;
  if (offset < ctx.return_data.size()) {
    bytes_to_copy = std::min(ctx.return_data.size() - offset, size);
  }
  std::fill_n(ctx.memory.data() + destoffset, size, 0);
  std::copy_n(ctx.return_data.data() + offset, bytes_to_copy, ctx.memory.data() + destoffset);

  ctx.pc++;
}

TOSCA_FORCEINLINE inline void extcodehash(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;

  auto address = ToEvmcAddress(StackPop(ctx));

  uint64_t dynamic_gas_cost = 2600;
  if (ctx.host.access_account(address) == EVMC_ACCESS_WARM) {
    dynamic_gas_cost = 100;
  }
  if (!ApplyGasCost(ctx, dynamic_gas_cost)) [[unlikely]]
    return;

  StackPush(ctx, ToUint256(ctx.host.get_code_hash(address)));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void blockhash(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 20)) [[unlikely]]
    return;
  int64_t number = static_cast<int64_t>(StackPop(ctx));
  StackPush(ctx, ToUint256(ctx.host.get_block_hash(number)));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void coinbase(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.host.get_tx_context().block_coinbase));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void timestamp(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.host.get_tx_context().block_timestamp);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void number(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.host.get_tx_context().block_number);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void prevrandao(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.host.get_tx_context().block_prev_randao));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void gaslimit(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.host.get_tx_context().block_gas_limit);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void chainid(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.host.get_tx_context().chain_id));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void selfbalance(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.host.get_balance(ctx.message->recipient)));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void basefee(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ToUint256(ctx.host.get_tx_context().block_base_fee));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void pop(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  (void)StackPop(ctx);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void mload(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  if (!ApplyGasCost(ctx, DynamicMemoryCost(offset + 32, ctx.current_mem_cost))) [[unlikely]]
    return;

  GrowMemory(ctx, offset + 32);

  if (offset < ctx.memory.size() + 32) {
    StackPush(ctx, intx::be::unsafe::load<uint256_t>(ctx.memory.data() + offset));
  } else {
    StackPush(ctx, 0);
  }
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void mstore(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  const uint256_t value = StackPop(ctx);
  if (!ApplyGasCost(ctx, DynamicMemoryCost(offset + 32, ctx.current_mem_cost))) [[unlikely]]
    return;

  GrowMemory(ctx, offset + 32);

  intx::be::unsafe::store(ctx.memory.data() + offset, value);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void mstore8(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  const uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  const uint256_t value = StackPop(ctx);
  if (!ApplyGasCost(ctx, DynamicMemoryCost(offset + 1, ctx.current_mem_cost))) [[unlikely]]
    return;

  GrowMemory(ctx, offset + 1);

  ctx.memory[offset] = static_cast<uint8_t>(value);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void sload(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;

  uint256_t key = StackPop(ctx);

  uint64_t dynamic_gas_cost = 2100;
  if (ctx.host.access_storage(ctx.message->recipient, ToEvmcBytes(key)) == EVMC_ACCESS_WARM) {
    dynamic_gas_cost = 100;
  }
  if (!ApplyGasCost(ctx, dynamic_gas_cost)) [[unlikely]]
    return;

  auto value = ctx.host.get_storage(ctx.message->recipient, ToEvmcBytes(key));
  StackPush(ctx, ToUint256(value));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void sstore(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  uint256_t key = StackPop(ctx);
  uint256_t value = StackPop(ctx);

  // TODO: Take current_value and original_value into account!

  uint64_t dynamic_gas_cost = 100;
  if (ctx.host.access_storage(ctx.message->recipient, ToEvmcBytes(key)) == EVMC_ACCESS_COLD) {
    dynamic_gas_cost += 2100;
  }
  if (!ApplyGasCost(ctx, dynamic_gas_cost)) [[unlikely]]
    return;

  ctx.host.set_storage(ctx.message->recipient, ToEvmcBytes(key), ToEvmcBytes(value));
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void jump(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 8)) [[unlikely]]
    return;
  uint64_t counter = static_cast<uint64_t>(StackPop(ctx));
  if (!CheckJumpDest(ctx, counter)) [[unlikely]]
    return;
  ctx.pc = counter;
}

TOSCA_FORCEINLINE inline void jumpi(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 10)) [[unlikely]]
    return;
  uint64_t counter = static_cast<uint64_t>(StackPop(ctx));
  uint64_t b = static_cast<uint64_t>(StackPop(ctx));
  if (b != 0) {
    if (!CheckJumpDest(ctx, counter)) [[unlikely]]
      return;
    ctx.pc = counter;
  } else {
    ctx.pc++;
  }
}

TOSCA_FORCEINLINE inline void pc(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.pc);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void msize(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.memory.size());
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void gas(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 2)) [[unlikely]]
    return;
  StackPush(ctx, ctx.gas);
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void jumpdest(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  ctx.pc++;
}

template <uint64_t N>
TOSCA_FORCEINLINE void push(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION_TEMPLATE(N);

  // Check if there are enough bytes left in the code.
  if (ctx.code.size() < ctx.pc + 1 + N) [[unlikely]] {
    ctx.state = RunState::kErrorOpcode;
    return;
  }

  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;

  // Idea was to maybe optimize this by just bitfiddling it out; or specialize
  // push1 and 2 tried both quickly, 0 positive impact on perf measured.
  // Templates are great!
  uint256_t val = 0;
  for (uint64_t i = 1; i <= N; ++i) {
    val |= static_cast<uint256_t>(ctx.code[ctx.pc + i]) << (N - i) * 8;
  }
  StackPush(ctx, val);
  ctx.pc += 1 + N;
}

template <uint64_t N>
TOSCA_FORCEINLINE void dup(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION_TEMPLATE(N);
  if (!CheckStackAvailable(ctx, N)) [[unlikely]]
    return;
  if (!CheckStackOverflow(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  StackPush(ctx, ctx.stack[ctx.stack_pos - N + 1]);
  ctx.pc++;
}

template <uint64_t N>
TOSCA_FORCEINLINE void swap(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION_TEMPLATE(N);
  if (!CheckStackAvailable(ctx, N + 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 3)) [[unlikely]]
    return;
  std::swap(ctx.stack[ctx.stack_pos - N], ctx.stack[ctx.stack_pos]);
  ctx.pc++;
}

template <uint64_t N>
TOSCA_FORCEINLINE void log(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION_TEMPLATE(N);
  if (!CheckStackAvailable(ctx, 2 + N)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 375)) [[unlikely]]
    return;

  uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  uint64_t size = static_cast<uint64_t>(StackPop(ctx));
  std::array<evmc::bytes32, N> topics;
  for (unsigned i = 0; i < N; ++i) {
    topics[i] = ToEvmcBytes(StackPop(ctx));
  }

  if (!ApplyGasCost(ctx, 375 * N + 8 * size + DynamicMemoryCost(offset + size, ctx.current_mem_cost))) [[unlikely]]
    return;

  GrowMemory(ctx, offset + size);

  ctx.host.emit_log(ctx.message->recipient, ctx.memory.data() + offset, size, topics.data(), topics.size());
  ctx.pc++;
}

TOSCA_FORCEINLINE inline void return_op(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 2)) [[unlikely]]
    return;
  uint64_t offset = static_cast<uint64_t>(StackPop(ctx));
  uint64_t size = static_cast<uint64_t>(StackPop(ctx));
  if (!ApplyGasCost(ctx, DynamicMemoryCost(offset + size, ctx.current_mem_cost))) [[unlikely]]
    return;

  GrowMemory(ctx, offset + size);

  ctx.return_data.resize(size);
  std::copy_n(ctx.memory.data() + offset, size, ctx.return_data.data());

  ctx.state = RunState::kDone;
}

TOSCA_FORCEINLINE inline void invalid(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;

  // All the remaining gas in this context is consumed.
  ctx.gas = 0;

  ctx.state = RunState::kInvalid;
}

TOSCA_FORCEINLINE inline void selfdestruct(Context& ctx) noexcept {
  DBG_TRACE_INSTRUCTION;
  if (!CheckStackAvailable(ctx, 1)) [[unlikely]]
    return;
  if (!ApplyGasCost(ctx, 5000)) [[unlikely]]
    return;

  auto address = ToEvmcAddress(StackPop(ctx));

  // TODO: Dynamic gas cost

  ctx.host.selfdestruct(ctx.message->recipient, address);
  ctx.state = RunState::kDone;
}

////////////////////////////////////////////////////////////
// Instructions below are taken from evmone!

template <op::OpCodes Op>
TOSCA_FORCEINLINE inline void create_impl(Context& ctx) noexcept {
  static_assert(Op == op::CREATE || Op == op::CREATE2);

  DBG_TRACE_INSTRUCTION;

  // if (state.in_static_mode()) {
  //   return {EVMC_STATIC_MODE_VIOLATION, gas_left};
  // }

  const auto endowment = StackPop(ctx);
  const uint64_t init_code_offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t init_code_size = static_cast<uint64_t>(StackPop(ctx));
  const auto salt = (Op == op::CREATE2) ? StackPop(ctx) : uint256_t{0};

  ctx.return_data.clear();

  // Resize memory for input / output data.
  uint64_t new_mem_cost = 0;
  {
    uint64_t size_needed = init_code_offset + init_code_size;

    new_mem_cost = DynamicMemoryCost(size_needed, ctx.current_mem_cost);
    ctx.current_mem_cost = new_mem_cost;  // TODO move

    if (ctx.memory.size() < size_needed) {
      ctx.memory.resize(size_needed);
    }
  }

  // if (state.rev >= EVMC_SHANGHAI && init_code_size > 0xC000) {
  //   return {EVMC_OUT_OF_GAS, gas_left};
  // }

  // const auto init_code_word_cost = 6 * (Op == op::CREATE2) + 2 * (state.rev >= EVMC_SHANGHAI);
  // const auto init_code_cost = num_words(init_code_size) * init_code_word_cost;
  // if ((gas_left -= init_code_cost) < 0) {
  //   return {EVMC_OUT_OF_GAS, gas_left};
  // }

  if (ctx.message->depth >= 1024) {
    ctx.state = RunState::kErrorCreate;
    return;
  }

  if (endowment != 0 && ToUint256(ctx.host.get_balance(ctx.message->recipient)) < endowment) {
    ctx.state = RunState::kErrorCreate;
    return;
  }

  evmc_message msg{
      .kind = (Op == op::CREATE) ? EVMC_CREATE : EVMC_CREATE2,
      .depth = ctx.message->depth + 1,
      .sender = ctx.message->recipient,
      .input_data = ctx.memory.data() + init_code_offset,
      .input_size = init_code_size,
      .value = ToEvmcBytes(endowment),
      .create2_salt = ToEvmcBytes(salt),
  };

  // msg.gas = gas_left;
  // if (state.rev >= EVMC_TANGERINE_WHISTLE) {
  //   msg.gas = msg.gas - msg.gas / 64;
  // }

  const evmc::Result result = ctx.host.call(msg);
  // gas_left -= msg.gas - result.gas_left;
  // state.gas_refund += result.gas_refund;

  ctx.return_data.resize(result.output_size);
  std::copy_n(result.output_data, result.output_size, ctx.return_data.data());

  if (result.status_code == EVMC_SUCCESS) {
    StackPush(ctx, ToUint256(result.create_address));
  }

  ctx.pc++;
}

template <op::OpCodes Op>
TOSCA_FORCEINLINE inline void call_impl(Context& ctx) noexcept {
  static_assert(Op == op::CALL || Op == op::CALLCODE || Op == op::DELEGATECALL || Op == op::STATICCALL);

  DBG_TRACE_INSTRUCTION;

  if (!CheckStackAvailable(ctx, (Op == op::STATICCALL || Op == op::DELEGATECALL) ? 6 : 7)) [[unlikely]]
    return;

  const auto gas = StackPop(ctx);
  const auto dst = ToEvmcAddress(StackPop(ctx));
  const auto value = (Op == op::STATICCALL || Op == op::DELEGATECALL) ? 0 : StackPop(ctx);
  const auto has_value = value != 0;
  const uint64_t input_offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t input_size = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t output_offset = static_cast<uint64_t>(StackPop(ctx));
  const uint64_t output_size = static_cast<uint64_t>(StackPop(ctx));

  ctx.return_data.clear();

  // if (state.rev >= EVMC_BERLIN && ctx.host.access_account(dst) == EVMC_ACCESS_COLD) {
  //   if ((gas_left -= instr::additional_cold_account_access_cost) < 0) {
  //     return {EVMC_OUT_OF_GAS, gas_left};
  //   }
  // }

  // Resize memory for input / output data.
  uint64_t new_mem_cost = 0;
  {
    uint64_t size_needed = std::max(input_offset + input_size, output_offset + output_size);

    new_mem_cost = DynamicMemoryCost(size_needed, ctx.current_mem_cost);
    ctx.current_mem_cost = new_mem_cost;  // TODO move

    if (ctx.memory.size() < size_needed) {
      ctx.memory.resize(size_needed);
    }
  }

  evmc_message msg{
      .kind = (Op == op::DELEGATECALL) ? EVMC_DELEGATECALL
              : (Op == op::CALLCODE)   ? EVMC_CALLCODE
                                       : EVMC_CALL,
      .flags = (Op == op::STATICCALL) ? uint32_t{EVMC_STATIC} : ctx.message->flags,
      .depth = ctx.message->depth + 1,
      .recipient = (Op == op::CALL || Op == op::STATICCALL) ? dst : ctx.message->recipient,
      .sender = (Op == op::DELEGATECALL) ? ctx.message->sender : ctx.message->recipient,
      .input_data = ctx.memory.data() + input_offset,
      .input_size = input_size,
      .value = (Op == op::DELEGATECALL) ? ctx.message->value : ToEvmcBytes(value),
      .code_address = dst,
  };

  // auto cost = has_value ? 9000 : 0;

  // if constexpr (Op == op::CALL) {
  //   if (has_value && state.in_static_mode()) {
  //     return {EVMC_STATIC_MODE_VIOLATION, gas_left};
  //   }

  //   if ((has_value || state.rev < EVMC_SPURIOUS_DRAGON) && !state.host.account_exists(dst)) {
  //     cost += 25000;
  //   }
  // }

  // if ((gas_left -= cost) < 0) {
  //   return {EVMC_OUT_OF_GAS, gas_left};
  // }

  msg.gas = std::numeric_limits<int64_t>::max();
  if (gas < msg.gas) {
    msg.gas = static_cast<int64_t>(gas);
  }

  // if (state.rev >= EVMC_TANGERINE_WHISTLE) {  // TODO: Always true for STATICCALL.
  //   msg.gas = std::min(msg.gas, gas_left - gas_left / 64);
  // } else if (msg.gas > gas_left) {
  //   return {EVMC_OUT_OF_GAS, gas_left};
  // }

  // if (has_value) {
  //   msg.gas += 2300;  // Add stipend.
  //   gas_left += 2300;
  // }

  if (ctx.message->depth >= 1024) {
    ctx.state = RunState::kErrorCall;
    return;
  }

  if (has_value && ToUint256(ctx.host.get_balance(ctx.message->recipient)) < value) {
    ctx.state = RunState::kErrorCall;
    return;
  }

  const evmc::Result result = ctx.host.call(msg);
  ctx.return_data.assign(result.output_data, result.output_data + result.output_size);

  StackPush(ctx, result.status_code == EVMC_SUCCESS);

  std::copy_n(result.output_data, std::min<size_t>(output_size, result.output_size),  //
              ctx.memory.data() + output_offset);

  // const auto gas_used = msg.gas - result.gas_left;
  // gas_left -= gas_used;
  // state.gas_refund += result.gas_refund;

  ctx.pc++;
}

////////////////////////////////////////////////////////////

}  // namespace tosca::evmzero::op

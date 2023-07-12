#include "vm/evmzero/interpreter.h"

#include <bit>
#include <cstdio>
#include <iostream>

#include "common/assert.h"
#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero {

const char* ToString(RunState state) {
  switch (state) {
    case RunState::kRunning:
      return "Running";
    case RunState::kDone:
      return "Done";
    case RunState::kReturn:
      return "Return";
    case RunState::kRevert:
      return "Revert";
    case RunState::kInvalid:
      return "Invalid";
    case RunState::kErrorOpcode:
      return "ErrorOpcode";
    case RunState::kErrorGas:
      return "ErrorGas";
    case RunState::kErrorStackUnderflow:
      return "ErrorStackUnderflow";
    case RunState::kErrorStackOverflow:
      return "ErrorStackOverflow";
    case RunState::kErrorJump:
      return "ErrorJump";
    case RunState::kErrorReturnDataCopyOutOfBounds:
      return "ErrorReturnDataCopyOutOfBounds";
    case RunState::kErrorCall:
      return "ErrorCall";
    case RunState::kErrorCreate:
      return "ErrorCreate";
    case RunState::kErrorStaticCall:
      return "ErrorStaticCall";
  }
  return "UNKNOWN_STATE";
}

bool IsSuccess(RunState state) {
  return state == RunState::kDone       //
         || state == RunState::kReturn  //
         || state == RunState::kRevert;
}

std::ostream& operator<<(std::ostream& out, RunState state) { return out << ToString(state); }

template <bool LoggingEnabled>
InterpreterResult Interpret(const InterpreterArgs& args) {
  evmc::HostContext host(*args.host_interface, args.host_context);

  internal::Context ctx{
      .is_static_call = static_cast<bool>(args.message->flags & EVMC_STATIC),
      .gas = args.message->gas,
      .message = args.message,
      .host = &host,
      .revision = args.revision,
  };
  ctx.code.reserve(args.code.size() + 32);  // with additional STOP bytes
  ctx.code.assign(args.code.begin(), args.code.end());

  internal::RunInterpreter<LoggingEnabled>(ctx);

  return {
      .state = ctx.state,
      .remaining_gas = ctx.gas,
      .refunded_gas = ctx.gas_refunds,
      .return_data = ctx.return_data,
  };
}

template InterpreterResult Interpret<false>(const InterpreterArgs&);
template InterpreterResult Interpret<true>(const InterpreterArgs&);

///////////////////////////////////////////////////////////

namespace op {

using internal::Context;
using internal::kMaxGas;

static void stop(Context& ctx) noexcept { ctx.state = RunState::kDone; }

static void add(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a + b);
  ctx.pc++;
}

static void mul(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a * b);
  ctx.pc++;
}

static void sub(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a - b);
  ctx.pc++;
}

static void div(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  if (b == 0)
    ctx.stack.Push(0);
  else
    ctx.stack.Push(a / b);
  ctx.pc++;
}

static void sdiv(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  if (b == 0)
    ctx.stack.Push(0);
  else
    ctx.stack.Push(intx::sdivrem(a, b).quot);
  ctx.pc++;
}

static void mod(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  if (b == 0)
    ctx.stack.Push(0);
  else
    ctx.stack.Push(a % b);
  ctx.pc++;
}

static void smod(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  if (b == 0)
    ctx.stack.Push(0);
  else
    ctx.stack.Push(intx::sdivrem(a, b).rem);
  ctx.pc++;
}

static void addmod(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(3)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(8)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  uint256_t N = ctx.stack.Pop();
  if (N == 0)
    ctx.stack.Push(0);
  else
    ctx.stack.Push(intx::addmod(a, b, N));
  ctx.pc++;
}

static void mulmod(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(3)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(8)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  uint256_t N = ctx.stack.Pop();
  if (N == 0)
    ctx.stack.Push(0);
  else
    ctx.stack.Push(intx::mulmod(a, b, N));
  ctx.pc++;
}

static void exp(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(10)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t exponent = ctx.stack.Pop();
  if (!ctx.ApplyGasCost(50 * intx::count_significant_bytes(exponent))) [[unlikely]]
    return;
  ctx.stack.Push(intx::exp(a, exponent));
  ctx.pc++;
}

static void signextend(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5)) [[unlikely]]
    return;

  uint8_t leading_byte_index = static_cast<uint8_t>(ctx.stack.Pop());
  if (leading_byte_index > 31) {
    leading_byte_index = 31;
  }

  uint256_t value = ctx.stack.Pop();

  bool is_negative = ToByteArrayLe(value)[leading_byte_index] & 0b1000'0000;
  if (is_negative) {
    auto mask = kUint256Max << (8 * (leading_byte_index + 1));
    ctx.stack.Push(mask | value);
  } else {
    auto mask = kUint256Max >> (8 * (31 - leading_byte_index));
    ctx.stack.Push(mask & value);
  }

  ctx.pc++;
}

static void lt(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a < b ? 1 : 0);
  ctx.pc++;
}

static void gt(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a > b ? 1 : 0);
  ctx.pc++;
}

static void slt(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(intx::slt(a, b) ? 1 : 0);
  ctx.pc++;
}

static void sgt(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(intx::slt(b, a) ? 1 : 0);
  ctx.pc++;
}

static void eq(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a == b ? 1 : 0);
  ctx.pc++;
}

static void iszero(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t val = ctx.stack.Pop();
  ctx.stack.Push(val == 0);
  ctx.pc++;
}

static void bit_and(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a & b);
  ctx.pc++;
}

static void bit_or(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a | b);
  ctx.pc++;
}

static void bit_xor(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  uint256_t b = ctx.stack.Pop();
  ctx.stack.Push(a ^ b);
  ctx.pc++;
}

static void bit_not(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t a = ctx.stack.Pop();
  ctx.stack.Push(~a);
  ctx.pc++;
}

static void byte(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t offset = ctx.stack.Pop();
  uint256_t x = ctx.stack.Pop();
  if (offset < 32) {
    // Offset starts at most significant byte.
    ctx.stack.Push(ToByteArrayLe(x)[31 - static_cast<uint8_t>(offset)]);
  } else {
    ctx.stack.Push(0);
  }
  ctx.pc++;
}

static void shl(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t shift = ctx.stack.Pop();
  uint256_t value = ctx.stack.Pop();
  ctx.stack.Push(value << shift);
  ctx.pc++;
}

static void shr(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t shift = ctx.stack.Pop();
  uint256_t value = ctx.stack.Pop();
  ctx.stack.Push(value >> shift);
  ctx.pc++;
}

static void sar(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t shift = ctx.stack.Pop();
  uint256_t value = ctx.stack.Pop();
  const bool is_negative = ToByteArrayLe(value)[31] & 0b1000'0000;

  if (shift <= 255) {
    value >>= shift;
    if (is_negative) {
      value |= (kUint256Max << (255 - shift));
    }
    ctx.stack.Push(value);
  } else {
    ctx.stack.Push(0);
  }
  ctx.pc++;
}

static void sha3(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(30)) [[unlikely]]
    return;

  const uint256_t offset_u256 = ctx.stack.Pop();
  const uint256_t size_u256 = ctx.stack.Pop();

  const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  if (!ctx.ApplyGasCost(6 * minimum_word_size)) [[unlikely]]
    return;

  std::vector<uint8_t> buffer(size);
  ctx.memory.WriteTo(buffer, offset);

  auto hash = ethash::keccak256(buffer.data(), buffer.size());
  ctx.stack.Push(ToUint256(hash));
  ctx.pc++;
}

static void address(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.message->recipient));
  ctx.pc++;
}

static void balance(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;

  evmc::address address = ToEvmcAddress(ctx.stack.Pop());

  int64_t dynamic_gas_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2600;
    }
  }
  if (!ctx.ApplyGasCost(dynamic_gas_cost)) [[unlikely]]
    return;

  ctx.stack.Push(ToUint256(ctx.host->get_balance(address)));
  ctx.pc++;
}

static void origin(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.host->get_tx_context().tx_origin));
  ctx.pc++;
}

static void caller(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.message->sender));
  ctx.pc++;
}

static void callvalue(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.message->value));
  ctx.pc++;
}

static void calldataload(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  const uint256_t offset_u256 = ctx.stack.Pop();

  std::span<const uint8_t> input_view;
  if (offset_u256 < ctx.message->input_size) {
    input_view = std::span(ctx.message->input_data, ctx.message->input_size)  //
                     .subspan(static_cast<uint64_t>(offset_u256));
  }

  evmc::bytes32 value{};
  std::copy_n(input_view.begin(), std::min<size_t>(input_view.size(), 32), value.bytes);

  ctx.stack.Push(ToUint256(value));
  ctx.pc++;
}

static void calldatasize(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.message->input_size);
  ctx.pc++;
}

static void calldatacopy(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(3)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  const uint256_t memory_offset_u256 = ctx.stack.Pop();
  const uint256_t data_offset_u256 = ctx.stack.Pop();
  const uint256_t size_u256 = ctx.stack.Pop();

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  if (!ctx.ApplyGasCost(3 * minimum_word_size)) [[unlikely]]
    return;

  std::span<const uint8_t> data_view;
  if (data_offset_u256 < ctx.message->input_size) {
    data_view = std::span(ctx.message->input_data, ctx.message->input_size)  //
                    .subspan(static_cast<uint64_t>(data_offset_u256));
  }

  ctx.memory.ReadFromWithSize(data_view, memory_offset, size);
  ctx.pc++;
}

static void codesize(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.code.size() - 32);  // accounted for added STOP bytes
  ctx.pc++;
}

static void codecopy(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(3)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  const uint256_t memory_offset_u256 = ctx.stack.Pop();
  const uint256_t code_offset_u256 = ctx.stack.Pop();
  const uint256_t size_u256 = ctx.stack.Pop();

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  if (!ctx.ApplyGasCost(3 * minimum_word_size)) [[unlikely]]
    return;

  std::span<const uint8_t> code_view;
  if (code_offset_u256 < ctx.code.size()) {
    code_view = std::span(ctx.code).subspan(static_cast<uint64_t>(code_offset_u256));
  }

  ctx.memory.ReadFromWithSize(code_view, memory_offset, size);
  ctx.pc++;
}

static void gasprice(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.host->get_tx_context().tx_gas_price));
  ctx.pc++;
}

static void extcodesize(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;

  auto address = ToEvmcAddress(ctx.stack.Pop());

  int64_t dynamic_gas_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2600;
    }
  }
  if (!ctx.ApplyGasCost(dynamic_gas_cost)) [[unlikely]]
    return;

  ctx.stack.Push(ctx.host->get_code_size(address));
  ctx.pc++;
}

static void extcodecopy(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(4)) [[unlikely]]
    return;

  const auto address = ToEvmcAddress(ctx.stack.Pop());
  const uint256_t memory_offset_u256 = ctx.stack.Pop();
  const uint256_t code_offset_u256 = ctx.stack.Pop();
  const uint256_t size_u256 = ctx.stack.Pop();

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  int64_t address_access_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      address_access_cost = 100;
    } else {
      address_access_cost = 2600;
    }
  }
  if (!ctx.ApplyGasCost(3 * minimum_word_size + address_access_cost)) [[unlikely]]
    return;

  std::vector<uint8_t> buffer(size);
  if (code_offset_u256 <= std::numeric_limits<uint64_t>::max()) {
    ctx.host->copy_code(address, static_cast<uint64_t>(code_offset_u256), buffer.data(), buffer.size());
  }

  ctx.memory.ReadFrom(buffer, memory_offset);
  ctx.pc++;
}

static void returndatasize(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.return_data.size());
  ctx.pc++;
}

static void returndatacopy(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(3)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  const uint256_t memory_offset_u256 = ctx.stack.Pop();
  const uint256_t return_data_offset_u256 = ctx.stack.Pop();
  const uint256_t size_u256 = ctx.stack.Pop();

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  if (!ctx.ApplyGasCost(3 * minimum_word_size)) [[unlikely]]
    return;

  {
    const auto [end_u256, carry] = intx::addc(return_data_offset_u256, size_u256);
    if (carry || end_u256 > ctx.return_data.size()) {
      ctx.state = RunState::kErrorReturnDataCopyOutOfBounds;
      return;
    }
  }

  std::span<const uint8_t> return_data_view = std::span(ctx.return_data)  //
                                                  .subspan(static_cast<uint64_t>(return_data_offset_u256));
  ctx.memory.ReadFromWithSize(return_data_view, memory_offset, size);
  ctx.pc++;
}

static void extcodehash(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;

  auto address = ToEvmcAddress(ctx.stack.Pop());

  int64_t dynamic_gas_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2600;
    }
  }
  if (!ctx.ApplyGasCost(dynamic_gas_cost)) [[unlikely]]
    return;

  ctx.stack.Push(ToUint256(ctx.host->get_code_hash(address)));
  ctx.pc++;
}

static void blockhash(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(20)) [[unlikely]]
    return;
  int64_t number = static_cast<int64_t>(ctx.stack.Pop());
  ctx.stack.Push(ToUint256(ctx.host->get_block_hash(number)));
  ctx.pc++;
}

static void coinbase(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.host->get_tx_context().block_coinbase));
  ctx.pc++;
}

static void timestamp(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.host->get_tx_context().block_timestamp);
  ctx.pc++;
}

static void blocknumber(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.host->get_tx_context().block_number);
  ctx.pc++;
}

static void prevrandao(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.host->get_tx_context().block_prev_randao));
  ctx.pc++;
}

static void gaslimit(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.host->get_tx_context().block_gas_limit);
  ctx.pc++;
}

static void chainid(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.host->get_tx_context().chain_id));
  ctx.pc++;
}

static void selfbalance(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.host->get_balance(ctx.message->recipient)));
  ctx.pc++;
}

static void basefee(Context& ctx) noexcept {
  if (!ctx.CheckOpcodeAvailable(EVMC_LONDON)) [[unlikely]]
    return;
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ToUint256(ctx.host->get_tx_context().block_base_fee));
  ctx.pc++;
}

static void pop(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Pop();
  ctx.pc++;
}

static void mload(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  const uint256_t offset_u256 = ctx.stack.Pop();

  const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 32);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  uint256_t value;
  ctx.memory.WriteTo({ToBytes(value), 32}, offset);

  if constexpr (std::endian::native == std::endian::little) {
    value = intx::bswap(value);
  }

  ctx.stack.Push(value);
  ctx.pc++;
}

static void mstore(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  const uint256_t offset_u256 = ctx.stack.Pop();
  uint256_t value = ctx.stack.Pop();

  const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 32);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  if constexpr (std::endian::native == std::endian::little) {
    value = intx::bswap(value);
  }

  ctx.memory.ReadFrom({ToBytes(value), 32}, offset);
  ctx.pc++;
}

static void mstore8(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  const uint256_t offset_u256 = ctx.stack.Pop();
  const uint8_t value = static_cast<uint8_t>(ctx.stack.Pop());

  const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 1);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  ctx.memory.ReadFrom({&value, 1}, offset);
  ctx.pc++;
}

static void sload(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;

  const uint256_t key = ctx.stack.Pop();

  int64_t dynamic_gas_cost = 800;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_storage(ctx.message->recipient, ToEvmcBytes(key)) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2100;
    }
  }
  if (!ctx.ApplyGasCost(dynamic_gas_cost)) [[unlikely]]
    return;

  auto value = ctx.host->get_storage(ctx.message->recipient, ToEvmcBytes(key));
  ctx.stack.Push(ToUint256(value));
  ctx.pc++;
}

static void sstore(Context& ctx) noexcept {
  // EIP-2200
  if (ctx.gas <= 2300) [[unlikely]] {
    ctx.state = RunState::kErrorGas;
    return;
  }

  if (!ctx.CheckStaticCallConformance()) [[unlikely]]
    return;
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  const uint256_t key = ctx.stack.Pop();
  const uint256_t value = ctx.stack.Pop();

  bool key_is_warm = false;
  if (ctx.revision >= EVMC_BERLIN) {
    key_is_warm = ctx.host->access_storage(ctx.message->recipient, ToEvmcBytes(key)) == EVMC_ACCESS_WARM;
  }

  int64_t dynamic_gas_cost = 800;
  if (ctx.revision >= EVMC_BERLIN) {
    dynamic_gas_cost = 100;
  }

  const auto storage_status = ctx.host->set_storage(ctx.message->recipient, ToEvmcBytes(key), ToEvmcBytes(value));

  // Dynamic gas cost depends on the current value in storage. set_storage
  // provides the relevant information we need.
  if (storage_status == EVMC_STORAGE_ADDED) {
    dynamic_gas_cost = 20000;
  }
  if (storage_status == EVMC_STORAGE_MODIFIED || storage_status == EVMC_STORAGE_DELETED) {
    if (ctx.revision >= EVMC_BERLIN) {
      dynamic_gas_cost = 2900;
    } else {
      dynamic_gas_cost = 5000;
    }
  }

  if (ctx.revision >= EVMC_BERLIN && !key_is_warm) {
    dynamic_gas_cost += 2100;
  }

  if (!ctx.ApplyGasCost(dynamic_gas_cost)) [[unlikely]]
    return;

  // gas refund
  {
    auto warm_cold_restored = [&]() -> int64_t {
      if (ctx.revision >= EVMC_BERLIN) {
        if (key_is_warm) {
          return 5000 - 2100 - 100;
        } else {
          return 4900;
        }
      } else {
        return 4200;
      }
    };

    switch (storage_status) {
      case EVMC_STORAGE_DELETED:
        ctx.gas_refunds += ctx.revision >= EVMC_LONDON ? 4800 : 15000;
        break;
      case EVMC_STORAGE_DELETED_ADDED:
        ctx.gas_refunds -= ctx.revision >= EVMC_LONDON ? 4800 : 15000;
        break;
      case EVMC_STORAGE_MODIFIED_DELETED:
        ctx.gas_refunds += ctx.revision >= EVMC_LONDON ? 4800 : 15000;
        break;
      case EVMC_STORAGE_DELETED_RESTORED:
        ctx.gas_refunds -= ctx.revision >= EVMC_LONDON ? 4800 : 15000;
        ctx.gas_refunds += warm_cold_restored();
        break;
      case EVMC_STORAGE_ADDED_DELETED:
        ctx.gas_refunds += ctx.revision >= EVMC_BERLIN ? 19900 : 19200;
        break;
      case EVMC_STORAGE_MODIFIED_RESTORED:
        ctx.gas_refunds += warm_cold_restored();
        break;
      case EVMC_STORAGE_ASSIGNED:
      case EVMC_STORAGE_ADDED:
      case EVMC_STORAGE_MODIFIED:
        break;
    }
  }

  ctx.pc++;
}

static void jump(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(8)) [[unlikely]]
    return;
  const uint256_t counter_u256 = ctx.stack.Pop();
  if (!ctx.CheckJumpDest(counter_u256)) [[unlikely]]
    return;
  ctx.pc = static_cast<uint64_t>(counter_u256);
}

static void jumpi(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(10)) [[unlikely]]
    return;
  const uint256_t counter_u256 = ctx.stack.Pop();
  const uint256_t b = ctx.stack.Pop();
  if (b != 0) {
    if (!ctx.CheckJumpDest(counter_u256)) [[unlikely]]
      return;
    ctx.pc = static_cast<uint64_t>(counter_u256);
  } else {
    ctx.pc++;
  }
}

static void pc(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.pc);
  ctx.pc++;
}

static void msize(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.memory.GetSize());
  ctx.pc++;
}

static void gas(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(2)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.gas);
  ctx.pc++;
}

static void jumpdest(Context& ctx) noexcept {
  if (!ctx.ApplyGasCost(1)) [[unlikely]]
    return;
  ctx.pc++;
}

template <uint64_t N>
static void push(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  uint256_t value = 0;
  for (uint64_t i = 1; i <= N; ++i) {
    value |= static_cast<uint256_t>(ctx.code[ctx.pc + i]) << (N - i) * 8;
  }
  ctx.stack.Push(value);
  ctx.pc += 1 + N;
}

template <uint64_t N>
static void dup(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(N)) [[unlikely]]
    return;
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  ctx.stack.Push(ctx.stack[N - 1]);
  ctx.pc++;
}

template <uint64_t N>
static void swap(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(N + 1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  std::swap(ctx.stack[0], ctx.stack[N]);
  ctx.pc++;
}

template <uint64_t N>
static void log(Context& ctx) noexcept {
  if (!ctx.CheckStaticCallConformance()) [[unlikely]]
    return;
  if (!ctx.CheckStackAvailable(2 + N)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(375)) [[unlikely]]
    return;

  const uint256_t offset_u256 = ctx.stack.Pop();
  const uint256_t size_u256 = ctx.stack.Pop();

  const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  std::array<evmc::bytes32, N> topics;
  for (unsigned i = 0; i < N; ++i) {
    topics[i] = ToEvmcBytes(ctx.stack.Pop());
  }

  if (!ctx.ApplyGasCost(static_cast<int64_t>(375 * N + 8 * size))) [[unlikely]]
    return;

  std::vector<uint8_t> buffer(size);
  ctx.memory.WriteTo(buffer, offset);

  ctx.host->emit_log(ctx.message->recipient, buffer.data(), buffer.size(), topics.data(), topics.size());
  ctx.pc++;
}

template <RunState result_state>
static void return_op(Context& ctx) noexcept {
  static_assert(result_state == RunState::kReturn || result_state == RunState::kRevert);

  if (!ctx.CheckStackAvailable(2)) [[unlikely]]
    return;

  const uint256_t offset_u256 = ctx.stack.Pop();
  const uint256_t size_u256 = ctx.stack.Pop();

  const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  ctx.return_data.resize(size);
  ctx.memory.WriteTo(ctx.return_data, offset);
  ctx.state = result_state;
}

static void invalid(Context& ctx) noexcept { ctx.state = RunState::kInvalid; }

static void selfdestruct(Context& ctx) noexcept {
  if (!ctx.CheckStaticCallConformance()) [[unlikely]]
    return;
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(5000)) [[unlikely]]
    return;

  auto account = ToEvmcAddress(ctx.stack.Pop());

  {
    int64_t dynamic_gas_cost = 0;
    if (ctx.host->get_balance(ctx.message->recipient) && !ctx.host->account_exists(account)) {
      dynamic_gas_cost += 25000;
    }
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_account(account) == EVMC_ACCESS_COLD) {
        dynamic_gas_cost += 2600;
      }
    }
    if (!ctx.ApplyGasCost(dynamic_gas_cost)) [[unlikely]]
      return;
  }

  if (ctx.host->selfdestruct(ctx.message->recipient, account)) {
    if (ctx.revision < EVMC_LONDON) {
      ctx.gas_refunds += 24000;
    }
  }

  ctx.state = RunState::kDone;
}

template <op::OpCodes Op>
static void create_impl(Context& ctx) noexcept {
  static_assert(Op == op::CREATE || Op == op::CREATE2);

  if (ctx.message->depth >= 1024) [[unlikely]] {
    ctx.state = RunState::kErrorCreate;
    return;
  }

  if (!ctx.CheckStaticCallConformance()) [[unlikely]]
    return;
  if (!ctx.CheckStackAvailable((Op == op::CREATE2) ? 4 : 3)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(32000)) [[unlikely]]
    return;

  const auto endowment = ctx.stack.Pop();
  const uint256_t init_code_offset_u256 = ctx.stack.Pop();
  const uint256_t init_code_size_u256 = ctx.stack.Pop();
  const auto salt = (Op == op::CREATE2) ? ctx.stack.Pop() : uint256_t{0};

  const auto [mem_cost, init_code_offset, init_code_size] =
      ctx.MemoryExpansionCost(init_code_offset_u256, init_code_size_u256);
  if (!ctx.ApplyGasCost(mem_cost)) [[unlikely]]
    return;

  if constexpr (Op == op::CREATE2) {
    const int64_t minimum_word_size = static_cast<int64_t>((init_code_size + 31) / 32);
    if (!ctx.ApplyGasCost(6 * minimum_word_size)) [[unlikely]]
      return;
  }

  ctx.return_data.clear();

  if (endowment != 0 && ToUint256(ctx.host->get_balance(ctx.message->recipient)) < endowment) {
    ctx.state = RunState::kErrorCreate;
    return;
  }

  std::vector<uint8_t> init_code(init_code_size);
  ctx.memory.WriteTo(init_code, init_code_offset);

  evmc_message msg{
      .kind = (Op == op::CREATE) ? EVMC_CREATE : EVMC_CREATE2,
      .depth = ctx.message->depth + 1,
      .gas = ctx.gas - ctx.gas / 64,
      .sender = ctx.message->recipient,
      .input_data = init_code.data(),
      .input_size = init_code.size(),
      .value = ToEvmcBytes(endowment),
      .create2_salt = ToEvmcBytes(salt),
  };

  const evmc::Result result = ctx.host->call(msg);
  if (result.status_code == EVMC_REVERT) {
    ctx.return_data.assign(result.output_data, result.output_data + result.output_size);
  }

  if (!ctx.ApplyGasCost(msg.gas - result.gas_left)) [[unlikely]]
    return;

  ctx.gas_refunds += result.gas_refund;

  if (result.status_code == EVMC_SUCCESS) {
    ctx.stack.Push(ToUint256(result.create_address));
  } else {
    ctx.stack.Push(0);
  }

  ctx.pc++;
}

template <op::OpCodes Op>
static void call_impl(Context& ctx) noexcept {
  static_assert(Op == op::CALL || Op == op::CALLCODE || Op == op::DELEGATECALL || Op == op::STATICCALL);

  if (ctx.message->depth >= 1024) [[unlikely]] {
    ctx.state = RunState::kErrorCall;
    return;
  }

  if (!ctx.CheckStackAvailable((Op == op::STATICCALL || Op == op::DELEGATECALL) ? 6 : 7)) [[unlikely]]
    return;

  const uint256_t gas_u256 = ctx.stack.Pop();
  const auto account = ToEvmcAddress(ctx.stack.Pop());
  const auto value = (Op == op::STATICCALL || Op == op::DELEGATECALL) ? 0 : ctx.stack.Pop();
  const bool has_value = value != 0;
  const uint256_t input_offset_u256 = ctx.stack.Pop();
  const uint256_t input_size_u256 = ctx.stack.Pop();
  const uint256_t output_offset_u256 = ctx.stack.Pop();
  const uint256_t output_size_u256 = ctx.stack.Pop();

  const auto [input_mem_cost, input_offset, input_size] = ctx.MemoryExpansionCost(input_offset_u256, input_size_u256);
  const auto [output_mem_cost, output_offset, output_size] =
      ctx.MemoryExpansionCost(output_offset_u256, output_size_u256);

  if (!ctx.ApplyGasCost(std::max(input_mem_cost, output_mem_cost))) [[unlikely]]
    return;

  if constexpr (Op == op::CALL) {
    if (has_value && !ctx.CheckStaticCallConformance()) [[unlikely]]
      return;
  }

  // Dynamic gas costs (excluding memory expansion and code execution costs)
  {
    int64_t address_access_cost = 700;
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_account(account) == EVMC_ACCESS_WARM) {
        address_access_cost = 100;
      } else {
        address_access_cost = 2600;
      }
    }

    int64_t positive_value_cost = has_value ? 9000 : 0;
    int64_t value_to_empty_account_cost = 0;
    if constexpr (Op != op::CALLCODE) {
      if (has_value && !ctx.host->account_exists(account)) {
        value_to_empty_account_cost = 25000;
      }
    }

    if (!ctx.ApplyGasCost(address_access_cost + positive_value_cost + value_to_empty_account_cost)) [[unlikely]]
      return;
  }

  ctx.return_data.clear();

  std::vector<uint8_t> input_data(input_size);
  ctx.memory.WriteTo(input_data, input_offset);

  int64_t gas = kMaxGas;
  if (gas_u256 < kMaxGas) {
    gas = static_cast<int64_t>(gas_u256);
  }

  evmc_message msg{
      .kind = (Op == op::DELEGATECALL) ? EVMC_DELEGATECALL
              : (Op == op::CALLCODE)   ? EVMC_CALLCODE
                                       : EVMC_CALL,
      .flags = (Op == op::STATICCALL) ? uint32_t{EVMC_STATIC} : ctx.message->flags,
      .depth = ctx.message->depth + 1,
      .gas = std::min(gas, ctx.gas - ctx.gas / 64),
      .recipient = (Op == op::CALL || Op == op::STATICCALL) ? account : ctx.message->recipient,
      .sender = (Op == op::DELEGATECALL) ? ctx.message->sender : ctx.message->recipient,
      .input_data = input_data.data(),
      .input_size = input_data.size(),
      .value = (Op == op::DELEGATECALL) ? ctx.message->value : ToEvmcBytes(value),
      .code_address = account,
  };

  // call stipend
  if (has_value) {
    msg.gas += 2300;
    ctx.gas += 2300;
  }

  if (has_value && ToUint256(ctx.host->get_balance(ctx.message->recipient)) < value) {
    ctx.stack.Push(0);
    ctx.pc++;
    return;
  }

  const evmc::Result result = ctx.host->call(msg);
  ctx.return_data.assign(result.output_data, result.output_data + result.output_size);

  ctx.memory.Grow(output_offset, output_size);
  if (ctx.return_data.size() > 0) {
    ctx.memory.ReadFromWithSize(ctx.return_data, output_offset, output_size);
  }

  if (!ctx.ApplyGasCost(msg.gas - result.gas_left)) [[unlikely]]
    return;

  ctx.gas_refunds += result.gas_refund;

  ctx.stack.Push(result.status_code == EVMC_SUCCESS);
  ctx.pc++;
}

}  // namespace op

///////////////////////////////////////////////////////////

namespace internal {

bool Context::CheckOpcodeAvailable(evmc_revision introduced_in) noexcept {
  if (revision < introduced_in) [[unlikely]] {
    state = RunState::kErrorOpcode;
    return false;
  } else {
    return true;
  }
}

bool Context::CheckStaticCallConformance() noexcept {
  if (is_static_call) [[unlikely]] {
    state = RunState::kErrorStaticCall;
    return false;
  } else {
    return true;
  }
}

bool Context::CheckStackAvailable(uint64_t elements_needed) noexcept {
  if (stack.GetSize() < elements_needed) [[unlikely]] {
    state = RunState::kErrorStackUnderflow;
    return false;
  } else {
    return true;
  }
}

bool Context::CheckStackOverflow(uint64_t slots_needed) noexcept {
  if (stack.GetMaxSize() - stack.GetSize() < slots_needed) [[unlikely]] {
    state = RunState::kErrorStackOverflow;
    return false;
  } else {
    return true;
  }
}

bool Context::CheckJumpDest(uint256_t index_u256) noexcept {
  if (index_u256 >= code.size()) [[unlikely]] {
    state = RunState::kErrorJump;
    return false;
  }

  const uint64_t index = static_cast<uint64_t>(index_u256);

  FillValidJumpTargetsUpTo(index);
  if (!valid_jump_targets[index]) [[unlikely]] {
    state = RunState::kErrorJump;
    return false;
  }

  return true;
}

void Context::FillValidJumpTargetsUpTo(uint64_t index) noexcept {
  if (index < valid_jump_targets.size()) [[likely]] {
    return;
  }

  if (index >= code.size()) [[unlikely]] {
    TOSCA_ASSERT(false);
    return;
  }

  const uint64_t old_size = valid_jump_targets.size();
  valid_jump_targets.resize(index + 1);

  uint64_t cur = old_size;
  while (cur <= index) {
    const auto instruction = code[cur];

    if (op::PUSH1 <= instruction && instruction <= op::PUSH32) {
      // Skip PUSH and arguments
      cur += instruction - op::PUSH1 + 2;
    } else {
      valid_jump_targets[cur] = instruction == op::JUMPDEST;
      cur++;
    }
  }
}

Context::MemoryExpansionCostResult Context::MemoryExpansionCost(uint256_t offset_u256, uint256_t size_u256) noexcept {
  const uint64_t uint64_max = std::numeric_limits<uint64_t>::max();
  if (offset_u256 > uint64_max || size_u256 > uint64_max) [[unlikely]] {
    return {kMaxGas};
  }

  const uint64_t offset = static_cast<uint64_t>(offset_u256);
  const uint64_t size = static_cast<uint64_t>(size_u256);

  if (size == 0) [[unlikely]] {
    return {0, offset, size};
  }

  uint64_t new_size = 0;
  if (TOSCA_CHECK_OVERFLOW_ADD(offset, size, &new_size)) [[unlikely]] {
    return {kMaxGas, offset, size};
  }

  if (new_size <= memory.GetSize()) {
    return {0, offset, size};
  }

  auto calc_memory_cost = [](uint64_t size) -> int64_t {
    const int64_t memory_size_word = static_cast<int64_t>((size + 31) / 32);
    return (memory_size_word * memory_size_word) / 512 + (3 * memory_size_word);
  };

  return {
      .gas_cost = calc_memory_cost(new_size) - calc_memory_cost(memory.GetSize()),
      .offset = offset,
      .size = size,
  };
}

bool Context::ApplyGasCost(int64_t gas_cost) noexcept {
  TOSCA_ASSERT(gas_cost >= 0);

  if (gas < gas_cost) [[unlikely]] {
    state = RunState::kErrorGas;
    return false;
  }

  gas -= gas_cost;

  return true;
}

template <bool LoggingEnabled>
void RunInterpreter(Context& ctx) {
  // Padding the code with additional STOP bytes so we don't have to
  // continuously check for end-of-code. We use multiple STOP bytes in case one
  // of the last instructions is a PUSH with too few arguments.
  ctx.code.resize(ctx.code.size() + 32, op::STOP);

  while (ctx.state == RunState::kRunning) {
    if constexpr (LoggingEnabled) {
      // log format: <op>, <gas>, <top-of-stack>\n
      std::cout << ToString(static_cast<op::OpCodes>(ctx.code[ctx.pc])) << ", "  //
                << ctx.gas << ", ";
      if (ctx.stack.GetSize() == 0) {
        std::cout << "-empty-";
      } else {
        std::cout << ctx.stack[0];
      }
      std::cout << "\n" << std::flush;
    }

    switch (ctx.code[ctx.pc]) {
      // clang-format off
      case op::STOP: op::stop(ctx); break;

      case op::ADD: op::add(ctx); break;
      case op::MUL: op::mul(ctx); break;
      case op::SUB: op::sub(ctx); break;
      case op::DIV: op::div(ctx); break;
      case op::SDIV: op::sdiv(ctx); break;
      case op::MOD: op::mod(ctx); break;
      case op::SMOD: op::smod(ctx); break;
      case op::ADDMOD: op::addmod(ctx); break;
      case op::MULMOD: op::mulmod(ctx); break;
      case op::EXP: op::exp(ctx); break;
      case op::SIGNEXTEND: op::signextend(ctx); break;
      case op::LT: op::lt(ctx); break;
      case op::GT: op::gt(ctx); break;
      case op::SLT: op::slt(ctx); break;
      case op::SGT: op::sgt(ctx); break;
      case op::EQ: op::eq(ctx); break;
      case op::ISZERO: op::iszero(ctx); break;
      case op::AND: op::bit_and(ctx); break;
      case op::OR: op::bit_or(ctx); break;
      case op::XOR: op::bit_xor(ctx); break;
      case op::NOT: op::bit_not(ctx); break;
      case op::BYTE: op::byte(ctx); break;
      case op::SHL: op::shl(ctx); break;
      case op::SHR: op::shr(ctx); break;
      case op::SAR: op::sar(ctx); break;
      case op::SHA3: op::sha3(ctx); break;
      case op::ADDRESS: op::address(ctx); break;
      case op::BALANCE: op::balance(ctx); break;
      case op::ORIGIN: op::origin(ctx); break;
      case op::CALLER: op::caller(ctx); break;
      case op::CALLVALUE: op::callvalue(ctx); break;
      case op::CALLDATALOAD: op::calldataload(ctx); break;
      case op::CALLDATASIZE: op::calldatasize(ctx); break;
      case op::CALLDATACOPY: op::calldatacopy(ctx); break;
      case op::CODESIZE: op::codesize(ctx); break;
      case op::CODECOPY: op::codecopy(ctx); break;
      case op::GASPRICE: op::gasprice(ctx); break;
      case op::EXTCODESIZE: op::extcodesize(ctx); break;
      case op::EXTCODECOPY: op::extcodecopy(ctx); break;
      case op::RETURNDATASIZE: op::returndatasize(ctx); break;
      case op::RETURNDATACOPY: op::returndatacopy(ctx); break;
      case op::EXTCODEHASH: op::extcodehash(ctx); break;
      case op::BLOCKHASH: op::blockhash(ctx); break;
      case op::COINBASE: op::coinbase(ctx); break;
      case op::TIMESTAMP: op::timestamp(ctx); break;
      case op::NUMBER: op::blocknumber(ctx); break;
      case op::DIFFICULTY: op::prevrandao(ctx); break; // intentional
      case op::GASLIMIT: op::gaslimit(ctx); break;
      case op::CHAINID: op::chainid(ctx); break;
      case op::SELFBALANCE: op::selfbalance(ctx); break;
      case op::BASEFEE: op::basefee(ctx); break;

      case op::POP: op::pop(ctx); break;
      case op::MLOAD: op::mload(ctx); break;
      case op::MSTORE: op::mstore(ctx); break;
      case op::MSTORE8: op::mstore8(ctx); break;
      case op::SLOAD: op::sload(ctx); break;
      case op::SSTORE: op::sstore(ctx); break;

      case op::JUMP: op::jump(ctx); break;
      case op::JUMPI: op::jumpi(ctx); break;
      case op::PC: op::pc(ctx); break;
      case op::MSIZE: op::msize(ctx); break;
      case op::GAS: op::gas(ctx); break;
      case op::JUMPDEST: op::jumpdest(ctx); break;

      case op::PUSH1: op::push<1>(ctx); break;
      case op::PUSH2: op::push<2>(ctx); break;
      case op::PUSH3: op::push<3>(ctx); break;
      case op::PUSH4: op::push<4>(ctx); break;
      case op::PUSH5: op::push<5>(ctx); break;
      case op::PUSH6: op::push<6>(ctx); break;
      case op::PUSH7: op::push<7>(ctx); break;
      case op::PUSH8: op::push<8>(ctx); break;
      case op::PUSH9: op::push<9>(ctx); break;
      case op::PUSH10: op::push<10>(ctx); break;
      case op::PUSH11: op::push<11>(ctx); break;
      case op::PUSH12: op::push<12>(ctx); break;
      case op::PUSH13: op::push<13>(ctx); break;
      case op::PUSH14: op::push<14>(ctx); break;
      case op::PUSH15: op::push<15>(ctx); break;
      case op::PUSH16: op::push<16>(ctx); break;
      case op::PUSH17: op::push<17>(ctx); break;
      case op::PUSH18: op::push<18>(ctx); break;
      case op::PUSH19: op::push<19>(ctx); break;
      case op::PUSH20: op::push<20>(ctx); break;
      case op::PUSH21: op::push<21>(ctx); break;
      case op::PUSH22: op::push<22>(ctx); break;
      case op::PUSH23: op::push<23>(ctx); break;
      case op::PUSH24: op::push<24>(ctx); break;
      case op::PUSH25: op::push<25>(ctx); break;
      case op::PUSH26: op::push<26>(ctx); break;
      case op::PUSH27: op::push<27>(ctx); break;
      case op::PUSH28: op::push<28>(ctx); break;
      case op::PUSH29: op::push<29>(ctx); break;
      case op::PUSH30: op::push<30>(ctx); break;
      case op::PUSH31: op::push<31>(ctx); break;
      case op::PUSH32: op::push<32>(ctx); break;

      case op::DUP1: op::dup<1>(ctx); break;
      case op::DUP2: op::dup<2>(ctx); break;
      case op::DUP3: op::dup<3>(ctx); break;
      case op::DUP4: op::dup<4>(ctx); break;
      case op::DUP5: op::dup<5>(ctx); break;
      case op::DUP6: op::dup<6>(ctx); break;
      case op::DUP7: op::dup<7>(ctx); break;
      case op::DUP8: op::dup<8>(ctx); break;
      case op::DUP9: op::dup<9>(ctx); break;
      case op::DUP10: op::dup<10>(ctx); break;
      case op::DUP11: op::dup<11>(ctx); break;
      case op::DUP12: op::dup<12>(ctx); break;
      case op::DUP13: op::dup<13>(ctx); break;
      case op::DUP14: op::dup<14>(ctx); break;
      case op::DUP15: op::dup<15>(ctx); break;
      case op::DUP16: op::dup<16>(ctx); break;

      case op::SWAP1: op::swap<1>(ctx); break;
      case op::SWAP2: op::swap<2>(ctx); break;
      case op::SWAP3: op::swap<3>(ctx); break;
      case op::SWAP4: op::swap<4>(ctx); break;
      case op::SWAP5: op::swap<5>(ctx); break;
      case op::SWAP6: op::swap<6>(ctx); break;
      case op::SWAP7: op::swap<7>(ctx); break;
      case op::SWAP8: op::swap<8>(ctx); break;
      case op::SWAP9: op::swap<9>(ctx); break;
      case op::SWAP10: op::swap<10>(ctx); break;
      case op::SWAP11: op::swap<11>(ctx); break;
      case op::SWAP12: op::swap<12>(ctx); break;
      case op::SWAP13: op::swap<13>(ctx); break;
      case op::SWAP14: op::swap<14>(ctx); break;
      case op::SWAP15: op::swap<15>(ctx); break;
      case op::SWAP16: op::swap<16>(ctx); break;

      case op::LOG0: op::log<0>(ctx); break;
      case op::LOG1: op::log<1>(ctx); break;
      case op::LOG2: op::log<2>(ctx); break;
      case op::LOG3: op::log<3>(ctx); break;
      case op::LOG4: op::log<4>(ctx); break;

      case op::CREATE: op::create_impl<op::CREATE>(ctx); break;
      case op::CREATE2: op::create_impl<op::CREATE2>(ctx); break;

      case op::RETURN: op::return_op<RunState::kReturn>(ctx); break;
      case op::REVERT: op::return_op<RunState::kRevert>(ctx); break;

      case op::CALL: op::call_impl<op::CALL>(ctx); break;
      case op::CALLCODE: op::call_impl<op::CALLCODE>(ctx); break;
      case op::DELEGATECALL: op::call_impl<op::DELEGATECALL>(ctx); break;
      case op::STATICCALL: op::call_impl<op::STATICCALL>(ctx); break;

      case op::INVALID: op::invalid(ctx); break;
      case op::SELFDESTRUCT: op::selfdestruct(ctx); break;
      // clang-format on
      default:
        ctx.state = RunState::kErrorOpcode;
    }
  }

  if (!IsSuccess(ctx.state)) {
    ctx.gas = 0;
  }

  // Keep return data only when we are supposed to return something.
  if (ctx.state != RunState::kReturn && ctx.state != RunState::kRevert) {
    ctx.return_data.clear();
  }
}

template void RunInterpreter<false>(Context&);
template void RunInterpreter<true>(Context&);

}  // namespace internal

}  // namespace tosca::evmzero

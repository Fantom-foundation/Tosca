#include "vm/evmzero/interpreter.h"

#include <bit>
#include <cstdio>
#include <intx/intx.hpp>
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

// Padding the code with additional STOP bytes so we don't have to continuously
// check for end-of-code. We use multiple STOP bytes in case one of the last
// instructions is a PUSH with too few arguments.
constexpr int kStopBytePadding = 33;

template <bool LoggingEnabled, bool ProfilingEnabled>
InterpreterResult Interpret(const InterpreterArgs& args) {
  evmc::HostContext host(*args.host_interface, args.host_context);

  internal::Context ctx{
      .is_static_call = static_cast<bool>(args.message->flags & EVMC_STATIC),
      .gas = args.message->gas,
      .valid_jump_targets = args.valid_jump_targets,
      .message = args.message,
      .host = &host,
      .revision = args.revision,
  };
  ctx.code.reserve(args.code.size() + kStopBytePadding);
  ctx.code.assign(args.code.begin(), args.code.end());

  auto& profiler = *static_cast<Profiler<ProfilingEnabled>*>(args.profiler);

  internal::RunInterpreter<LoggingEnabled, ProfilingEnabled>(ctx, profiler);

  return {
      .state = ctx.state,
      .remaining_gas = ctx.gas,
      .refunded_gas = ctx.gas_refunds,
      .return_data = ctx.return_data,
  };
}

template InterpreterResult Interpret<false, false>(const InterpreterArgs&);
template InterpreterResult Interpret<true, false>(const InterpreterArgs&);
template InterpreterResult Interpret<false, true>(const InterpreterArgs&);
template InterpreterResult Interpret<true, true>(const InterpreterArgs&);

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

  ctx.stack.Push(ctx.memory.CalculateHash(offset, size));
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
  ctx.stack.Push(ctx.code.size() - kStopBytePadding);
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

  auto memory_span = ctx.memory.GetSpan(memory_offset, size);
  if (code_offset_u256 <= std::numeric_limits<uint64_t>::max()) {
    uint64_t code_offset = static_cast<uint64_t>(code_offset_u256);
    size_t bytes_written = ctx.host->copy_code(address, code_offset, memory_span.data(), memory_span.size());
    memory_span = memory_span.subspan(bytes_written);
  }
  std::fill(memory_span.begin(), memory_span.end(), 0);

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

  constexpr auto num_full_words = N / sizeof(uint64_t);
  constexpr auto num_partial_bytes = N % sizeof(uint64_t);
  auto data = &ctx.code[ctx.pc + 1];

  uint256_t value = 0;
  if constexpr (num_partial_bytes != 0) {
    uint64_t word = 0;
    for (unsigned i = 0; i < num_partial_bytes; i++) {
      word = word << 8 | data[i];
    }
    value[num_full_words] = word;
    data += num_partial_bytes;
  }

  for (size_t i = 0; i < num_full_words; ++i) {
    if constexpr (std::endian::native == std::endian::little) {
      value[num_full_words - 1 - i] = intx::bswap(*reinterpret_cast<uint64_t*>(data));
    } else {
      value[num_full_words - 1 - i] = *reinterpret_cast<uint64_t*>(data);
    }
    data += sizeof(uint64_t);
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
  ctx.stack.Dup<N>();
  ctx.pc++;
}

template <uint64_t N>
static void swap(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(N + 1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  ctx.stack.Swap<N>();
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

  auto data = ctx.memory.GetSpan(offset, size);

  ctx.host->emit_log(ctx.message->recipient, data.data(), data.size(), topics.data(), topics.size());
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

  auto init_code = ctx.memory.GetSpan(init_code_offset, init_code_size);

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

  auto input_data = ctx.memory.GetSpan(input_offset, input_size);

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

inline bool Context::CheckStackAvailable(uint64_t elements_needed) noexcept {
  if (stack.GetSize() < elements_needed) [[unlikely]] {
    state = RunState::kErrorStackUnderflow;
    return false;
  } else {
    return true;
  }
}

inline bool Context::CheckStackOverflow(uint64_t slots_needed) noexcept {
  if (stack.GetMaxSize() - stack.GetSize() < slots_needed) [[unlikely]] {
    state = RunState::kErrorStackOverflow;
    return false;
  } else {
    return true;
  }
}

bool Context::CheckJumpDest(uint256_t index_u256) noexcept {
  if (index_u256 >= valid_jump_targets.size()) [[unlikely]] {
    state = RunState::kErrorJump;
    return false;
  }

  const uint64_t index = static_cast<uint64_t>(index_u256);

  if (!valid_jump_targets[index]) [[unlikely]] {
    state = RunState::kErrorJump;
    return false;
  }

  return true;
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

inline bool Context::ApplyGasCost(int64_t gas_cost) noexcept {
  TOSCA_ASSERT(gas_cost >= 0);

  if (gas < gas_cost) [[unlikely]] {
    state = RunState::kErrorGas;
    return false;
  }

  gas -= gas_cost;

  return true;
}

template <bool LoggingEnabled, bool ProfilingEnabled>
void RunInterpreter(Context& ctx, Profiler<ProfilingEnabled>& profiler) {
  ctx.code.resize(ctx.code.size() + kStopBytePadding, op::STOP);

#define PROFILE_START(marker) profiler.template Start<Markers::marker>()
#define PROFILE_END(marker) profiler.template End<Markers::marker>()
#define PROFILE_SCOPED(marker) const auto scope_##marker = profiler.template Scoped<Markers::marker>()

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

    PROFILE_START(DISPATCH);
    switch (ctx.code[ctx.pc]) {
        // clang-format off
      case op::STOP: PROFILE_START(STOP); op::stop(ctx); PROFILE_END(STOP); break;

      case op::ADD: PROFILE_START(ADD); op::add(ctx); PROFILE_END(ADD); break;
      case op::MUL: PROFILE_START(MUL); op::mul(ctx); PROFILE_END(MUL); break;
      case op::SUB: PROFILE_START(SUB); op::sub(ctx); PROFILE_END(SUB); break;
      case op::DIV: PROFILE_START(DIV); op::div(ctx); PROFILE_END(DIV); break;
      case op::SDIV: PROFILE_START(SDIV); op::sdiv(ctx); PROFILE_END(SDIV); break;
      case op::MOD: PROFILE_START(MOD); op::mod(ctx); PROFILE_END(MOD); break;
      case op::SMOD: PROFILE_START(SMOD); op::smod(ctx); PROFILE_END(SMOD); break;
      case op::ADDMOD: PROFILE_START(ADDMOD); op::addmod(ctx); PROFILE_END(ADDMOD); break;
      case op::MULMOD: PROFILE_START(MULMOD); op::mulmod(ctx); PROFILE_END(MULMOD); break;
      case op::EXP: PROFILE_START(EXP); op::exp(ctx); PROFILE_END(EXP); break;
      case op::SIGNEXTEND: PROFILE_START(SIGNEXTEND); op::signextend(ctx); PROFILE_END(SIGNEXTEND); break;
      case op::LT: PROFILE_START(LT); op::lt(ctx); PROFILE_END(LT); break;
      case op::GT: PROFILE_START(GT); op::gt(ctx); PROFILE_END(GT); break;
      case op::SLT: PROFILE_START(SLT); op::slt(ctx); PROFILE_END(SLT); break;
      case op::SGT: PROFILE_START(SGT); op::sgt(ctx); PROFILE_END(SGT); break;
      case op::EQ: PROFILE_START(EQ); op::eq(ctx); PROFILE_END(EQ); break;
      case op::ISZERO: PROFILE_START(ISZERO); op::iszero(ctx); PROFILE_END(ISZERO); break;
      case op::AND: PROFILE_START(AND); op::bit_and(ctx); PROFILE_END(AND); break;
      case op::OR: PROFILE_START(OR); op::bit_or(ctx); PROFILE_END(OR); break;
      case op::XOR: PROFILE_START(XOR); op::bit_xor(ctx); PROFILE_END(XOR); break;
      case op::NOT: PROFILE_START(NOT); op::bit_not(ctx); PROFILE_END(NOT); break;
      case op::BYTE: PROFILE_START(BYTE); op::byte(ctx); PROFILE_END(BYTE); break;
      case op::SHL: PROFILE_START(SHL); op::shl(ctx); PROFILE_END(SHL); break;
      case op::SHR: PROFILE_START(SHR); op::shr(ctx); PROFILE_END(SHR); break;
      case op::SAR: PROFILE_START(SAR); op::sar(ctx); PROFILE_END(SAR); break;
      case op::SHA3: PROFILE_START(SHA3); op::sha3(ctx); PROFILE_END(SHA3); break;
      case op::ADDRESS: PROFILE_START(ADDRESS); op::address(ctx); PROFILE_END(ADDRESS); break;
      case op::BALANCE: PROFILE_START(BALANCE); op::balance(ctx); PROFILE_END(BALANCE); break;
      case op::ORIGIN: PROFILE_START(ORIGIN); op::origin(ctx); PROFILE_END(ORIGIN); break;
      case op::CALLER: PROFILE_START(CALLER); op::caller(ctx); PROFILE_END(CALLER); break;
      case op::CALLVALUE: PROFILE_START(CALLVALUE); op::callvalue(ctx); PROFILE_END(CALLVALUE); break;
      case op::CALLDATALOAD: PROFILE_START(CALLDATALOAD); op::calldataload(ctx); PROFILE_END(CALLDATALOAD); break;
      case op::CALLDATASIZE: PROFILE_START(CALLDATASIZE); op::calldatasize(ctx); PROFILE_END(CALLDATASIZE); break;
      case op::CALLDATACOPY: PROFILE_START(CALLDATACOPY); op::calldatacopy(ctx); PROFILE_END(CALLDATACOPY); break;
      case op::CODESIZE: PROFILE_START(CODESIZE); op::codesize(ctx); PROFILE_END(CODESIZE); break;
      case op::CODECOPY: PROFILE_START(CODECOPY); op::codecopy(ctx); PROFILE_END(CODECOPY); break;
      case op::GASPRICE: PROFILE_START(GASPRICE); op::gasprice(ctx); PROFILE_END(GASPRICE); break;
      case op::EXTCODESIZE: PROFILE_START(EXTCODESIZE); op::extcodesize(ctx); PROFILE_END(EXTCODESIZE); break;
      case op::EXTCODECOPY: PROFILE_START(EXTCODECOPY); op::extcodecopy(ctx); PROFILE_END(EXTCODECOPY); break;
      case op::RETURNDATASIZE: PROFILE_START(RETURNDATASIZE); op::returndatasize(ctx); PROFILE_END(RETURNDATASIZE); break;
      case op::RETURNDATACOPY: PROFILE_START(RETURNDATACOPY); op::returndatacopy(ctx); PROFILE_END(RETURNDATACOPY); break;
      case op::EXTCODEHASH: PROFILE_START(EXTCODEHASH); op::extcodehash(ctx); PROFILE_END(EXTCODEHASH); break;
      case op::BLOCKHASH: PROFILE_START(BLOCKHASH); op::blockhash(ctx); PROFILE_END(BLOCKHASH); break;
      case op::COINBASE: PROFILE_START(COINBASE); op::coinbase(ctx); PROFILE_END(COINBASE); break;
      case op::TIMESTAMP: PROFILE_START(TIMESTAMP); op::timestamp(ctx); PROFILE_END(TIMESTAMP); break;
      case op::NUMBER: PROFILE_START(NUMBER); op::blocknumber(ctx); PROFILE_END(NUMBER); break;
      case op::DIFFICULTY: PROFILE_START(DIFFICULTY); op::prevrandao(ctx); PROFILE_END(DIFFICULTY); break; // intentional
      case op::GASLIMIT: PROFILE_START(GASLIMIT); op::gaslimit(ctx); PROFILE_END(GASLIMIT); break;
      case op::CHAINID: PROFILE_START(CHAINID); op::chainid(ctx); PROFILE_END(CHAINID); break;
      case op::SELFBALANCE: PROFILE_START(SELFBALANCE); op::selfbalance(ctx); PROFILE_END(SELFBALANCE); break;
      case op::BASEFEE: PROFILE_START(BASEFEE); op::basefee(ctx); PROFILE_END(BASEFEE); break;

      case op::POP: PROFILE_START(POP); op::pop(ctx); PROFILE_END(POP); break;
      case op::MLOAD: PROFILE_START(MLOAD); op::mload(ctx); PROFILE_END(MLOAD); break;
      case op::MSTORE: PROFILE_START(MSTORE); op::mstore(ctx); PROFILE_END(MSTORE); break;
      case op::MSTORE8: PROFILE_START(MSTORE8); op::mstore8(ctx); PROFILE_END(MSTORE8); break;
      case op::SLOAD: PROFILE_START(SLOAD); op::sload(ctx); PROFILE_END(SLOAD); break;
      case op::SSTORE: PROFILE_START(SSTORE); op::sstore(ctx); PROFILE_END(SSTORE); break;

      case op::JUMP: PROFILE_START(JUMP); op::jump(ctx); PROFILE_END(JUMP); break;
      case op::JUMPI: PROFILE_START(JUMPI); op::jumpi(ctx); PROFILE_END(JUMPI); break;
      case op::PC: PROFILE_START(PC); op::pc(ctx); PROFILE_END(PC); break;
      case op::MSIZE: PROFILE_START(MSIZE); op::msize(ctx); PROFILE_END(MSIZE); break;
      case op::GAS: PROFILE_START(GAS); op::gas(ctx); PROFILE_END(GAS); break;
      case op::JUMPDEST: PROFILE_START(JUMPDEST); op::jumpdest(ctx); PROFILE_END(JUMPDEST); break;

      case op::PUSH1: PROFILE_START(PUSH1); op::push<1>(ctx); PROFILE_END(PUSH1); break;
      case op::PUSH2: PROFILE_START(PUSH2); op::push<2>(ctx); PROFILE_END(PUSH2); break;
      case op::PUSH3: PROFILE_START(PUSH3); op::push<3>(ctx); PROFILE_END(PUSH3); break;
      case op::PUSH4: PROFILE_START(PUSH4); op::push<4>(ctx); PROFILE_END(PUSH4); break;
      case op::PUSH5: PROFILE_START(PUSH5); op::push<5>(ctx); PROFILE_END(PUSH5); break;
      case op::PUSH6: PROFILE_START(PUSH6); op::push<6>(ctx); PROFILE_END(PUSH6); break;
      case op::PUSH7: PROFILE_START(PUSH7); op::push<7>(ctx); PROFILE_END(PUSH7); break;
      case op::PUSH8: PROFILE_START(PUSH8); op::push<8>(ctx); PROFILE_END(PUSH8); break;
      case op::PUSH9: PROFILE_START(PUSH9); op::push<9>(ctx); PROFILE_END(PUSH9); break;
      case op::PUSH10: PROFILE_START(PUSH10); op::push<10>(ctx); PROFILE_END(PUSH10); break;
      case op::PUSH11: PROFILE_START(PUSH11); op::push<11>(ctx); PROFILE_END(PUSH11); break;
      case op::PUSH12: PROFILE_START(PUSH12); op::push<12>(ctx); PROFILE_END(PUSH12); break;
      case op::PUSH13: PROFILE_START(PUSH13); op::push<13>(ctx); PROFILE_END(PUSH13); break;
      case op::PUSH14: PROFILE_START(PUSH14); op::push<14>(ctx); PROFILE_END(PUSH14); break;
      case op::PUSH15: PROFILE_START(PUSH15); op::push<15>(ctx); PROFILE_END(PUSH15); break;
      case op::PUSH16: PROFILE_START(PUSH16); op::push<16>(ctx); PROFILE_END(PUSH16); break;
      case op::PUSH17: PROFILE_START(PUSH17); op::push<17>(ctx); PROFILE_END(PUSH17); break;
      case op::PUSH18: PROFILE_START(PUSH18); op::push<18>(ctx); PROFILE_END(PUSH18); break;
      case op::PUSH19: PROFILE_START(PUSH19); op::push<19>(ctx); PROFILE_END(PUSH19); break;
      case op::PUSH20: PROFILE_START(PUSH20); op::push<20>(ctx); PROFILE_END(PUSH20); break;
      case op::PUSH21: PROFILE_START(PUSH21); op::push<21>(ctx); PROFILE_END(PUSH21); break;
      case op::PUSH22: PROFILE_START(PUSH22); op::push<22>(ctx); PROFILE_END(PUSH22); break;
      case op::PUSH23: PROFILE_START(PUSH23); op::push<23>(ctx); PROFILE_END(PUSH23); break;
      case op::PUSH24: PROFILE_START(PUSH24); op::push<24>(ctx); PROFILE_END(PUSH24); break;
      case op::PUSH25: PROFILE_START(PUSH25); op::push<25>(ctx); PROFILE_END(PUSH25); break;
      case op::PUSH26: PROFILE_START(PUSH26); op::push<26>(ctx); PROFILE_END(PUSH26); break;
      case op::PUSH27: PROFILE_START(PUSH27); op::push<27>(ctx); PROFILE_END(PUSH27); break;
      case op::PUSH28: PROFILE_START(PUSH28); op::push<28>(ctx); PROFILE_END(PUSH28); break;
      case op::PUSH29: PROFILE_START(PUSH29); op::push<29>(ctx); PROFILE_END(PUSH29); break;
      case op::PUSH30: PROFILE_START(PUSH30); op::push<30>(ctx); PROFILE_END(PUSH30); break;
      case op::PUSH31: PROFILE_START(PUSH31); op::push<31>(ctx); PROFILE_END(PUSH31); break;
      case op::PUSH32: PROFILE_START(PUSH32); op::push<32>(ctx); PROFILE_END(PUSH32); break;

      case op::DUP1: PROFILE_START(DUP1); op::dup<1>(ctx); PROFILE_END(DUP1); break;
      case op::DUP2: PROFILE_START(DUP2); op::dup<2>(ctx); PROFILE_END(DUP2); break;
      case op::DUP3: PROFILE_START(DUP3); op::dup<3>(ctx); PROFILE_END(DUP3); break;
      case op::DUP4: PROFILE_START(DUP4); op::dup<4>(ctx); PROFILE_END(DUP4); break;
      case op::DUP5: PROFILE_START(DUP5); op::dup<5>(ctx); PROFILE_END(DUP5); break;
      case op::DUP6: PROFILE_START(DUP6); op::dup<6>(ctx); PROFILE_END(DUP6); break;
      case op::DUP7: PROFILE_START(DUP7); op::dup<7>(ctx); PROFILE_END(DUP7); break;
      case op::DUP8: PROFILE_START(DUP8); op::dup<8>(ctx); PROFILE_END(DUP8); break;
      case op::DUP9: PROFILE_START(DUP9); op::dup<9>(ctx); PROFILE_END(DUP9); break;
      case op::DUP10: PROFILE_START(DUP10); op::dup<10>(ctx); PROFILE_END(DUP10); break;
      case op::DUP11: PROFILE_START(DUP11); op::dup<11>(ctx); PROFILE_END(DUP11); break;
      case op::DUP12: PROFILE_START(DUP12); op::dup<12>(ctx); PROFILE_END(DUP12); break;
      case op::DUP13: PROFILE_START(DUP13); op::dup<13>(ctx); PROFILE_END(DUP13); break;
      case op::DUP14: PROFILE_START(DUP14); op::dup<14>(ctx); PROFILE_END(DUP14); break;
      case op::DUP15: PROFILE_START(DUP15); op::dup<15>(ctx); PROFILE_END(DUP15); break;
      case op::DUP16: PROFILE_START(DUP16); op::dup<16>(ctx); PROFILE_END(DUP16); break;

      case op::SWAP1: PROFILE_START(SWAP1); op::swap<1>(ctx); PROFILE_END(SWAP1); break;
      case op::SWAP2: PROFILE_START(SWAP2); op::swap<2>(ctx); PROFILE_END(SWAP2); break;
      case op::SWAP3: PROFILE_START(SWAP3); op::swap<3>(ctx); PROFILE_END(SWAP3); break;
      case op::SWAP4: PROFILE_START(SWAP4); op::swap<4>(ctx); PROFILE_END(SWAP4); break;
      case op::SWAP5: PROFILE_START(SWAP5); op::swap<5>(ctx); PROFILE_END(SWAP5); break;
      case op::SWAP6: PROFILE_START(SWAP6); op::swap<6>(ctx); PROFILE_END(SWAP6); break;
      case op::SWAP7: PROFILE_START(SWAP7); op::swap<7>(ctx); PROFILE_END(SWAP7); break;
      case op::SWAP8: PROFILE_START(SWAP8); op::swap<8>(ctx); PROFILE_END(SWAP8); break;
      case op::SWAP9: PROFILE_START(SWAP9); op::swap<9>(ctx); PROFILE_END(SWAP9); break;
      case op::SWAP10: PROFILE_START(SWAP10); op::swap<10>(ctx); PROFILE_END(SWAP10); break;
      case op::SWAP11: PROFILE_START(SWAP11); op::swap<11>(ctx); PROFILE_END(SWAP11); break;
      case op::SWAP12: PROFILE_START(SWAP12); op::swap<12>(ctx); PROFILE_END(SWAP12); break;
      case op::SWAP13: PROFILE_START(SWAP13); op::swap<13>(ctx); PROFILE_END(SWAP13); break;
      case op::SWAP14: PROFILE_START(SWAP14); op::swap<14>(ctx); PROFILE_END(SWAP14); break;
      case op::SWAP15: PROFILE_START(SWAP15); op::swap<15>(ctx); PROFILE_END(SWAP15); break;
      case op::SWAP16: PROFILE_START(SWAP16); op::swap<16>(ctx); PROFILE_END(SWAP16); break;

      case op::LOG0: PROFILE_START(LOG0); op::log<0>(ctx); PROFILE_END(LOG0); break;
      case op::LOG1: PROFILE_START(LOG1); op::log<1>(ctx); PROFILE_END(LOG1); break;
      case op::LOG2: PROFILE_START(LOG2); op::log<2>(ctx); PROFILE_END(LOG2); break;
      case op::LOG3: PROFILE_START(LOG3); op::log<3>(ctx); PROFILE_END(LOG3); break;
      case op::LOG4: PROFILE_START(LOG4); op::log<4>(ctx); PROFILE_END(LOG4); break;

      case op::CREATE: PROFILE_START(CREATE); op::create_impl<op::CREATE>(ctx); PROFILE_END(CREATE); break;
      case op::CREATE2: PROFILE_START(CREATE2); op::create_impl<op::CREATE2>(ctx); PROFILE_END(CREATE2); break;

      case op::RETURN: PROFILE_START(RETURN); op::return_op<RunState::kReturn>(ctx); PROFILE_END(RETURN); break;
      case op::REVERT: PROFILE_START(REVERT); op::return_op<RunState::kRevert>(ctx); PROFILE_END(REVERT); break;

      case op::CALL: PROFILE_START(CALL); op::call_impl<op::CALL>(ctx); PROFILE_END(CALL); break;
      case op::CALLCODE: PROFILE_START(CALLCODE); op::call_impl<op::CALLCODE>(ctx); PROFILE_END(CALLCODE); break;
      case op::DELEGATECALL: PROFILE_START(DELEGATECALL); op::call_impl<op::DELEGATECALL>(ctx); PROFILE_END(DELEGATECALL); break;
      case op::STATICCALL: PROFILE_START(STATICCALL); op::call_impl<op::STATICCALL>(ctx); PROFILE_END(STATICCALL); break;

      case op::INVALID: PROFILE_START(INVALID); op::invalid(ctx); PROFILE_END(INVALID); break;
      case op::SELFDESTRUCT: PROFILE_START(SELFDESTRUCT); op::selfdestruct(ctx); PROFILE_END(SELFDESTRUCT); break;
      // clang-format on
      default:
        ctx.state = RunState::kErrorOpcode;
    }
  }
  PROFILE_END(DISPATCH);

  if (!IsSuccess(ctx.state)) {
    ctx.gas = 0;
  }

  // Keep return data only when we are supposed to return something.
  if (ctx.state != RunState::kReturn && ctx.state != RunState::kRevert) {
    ctx.return_data.clear();
  }

#undef PROFILE_START
#undef PROFILE_END
#undef PROFILE_SCOPED
}

template void RunInterpreter<false, false>(Context&, Profiler<false>&);
template void RunInterpreter<true, false>(Context&, Profiler<false>&);
template void RunInterpreter<false, true>(Context&, Profiler<true>&);
template void RunInterpreter<true, true>(Context&, Profiler<true>&);

}  // namespace internal

}  // namespace tosca::evmzero

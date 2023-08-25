#include "vm/evmzero/interpreter.h"

#include <bit>
#include <cstdio>
#include <intx/intx.hpp>
#include <iostream>
#include <type_traits>

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
InterpreterResult Interpret(const InterpreterArgs<ProfilingEnabled>& args) {
  auto profiler = Profiler<ProfilingEnabled>{};
  profiler.template Start<Marker::INTERPRETER>();

  evmc::HostContext host(*args.host_interface, args.host_context);

  internal::Context ctx{
      .is_static_call = static_cast<bool>(args.message->flags & EVMC_STATIC),
      .gas = args.message->gas,
      .padded_code = args.padded_code,
      .valid_jump_targets = args.valid_jump_targets,
      .message = args.message,
      .host = &host,
      .revision = args.revision,
      .sha3_cache = args.sha3_cache,
  };

  internal::RunInterpreter<LoggingEnabled, ProfilingEnabled>(ctx, profiler);

  profiler.template End<Marker::INTERPRETER>();

  auto& vm_profiler = args.profiler;
  vm_profiler.Merge(profiler.Collect());

  return {
      .state = ctx.state,
      .remaining_gas = ctx.gas,
      .refunded_gas = ctx.gas_refunds,
      .return_data = ctx.return_data,
  };
}

template InterpreterResult Interpret<false, false>(const InterpreterArgs<false>&);
template InterpreterResult Interpret<true, false>(const InterpreterArgs<false>&);
template InterpreterResult Interpret<false, true>(const InterpreterArgs<true>&);
template InterpreterResult Interpret<true, true>(const InterpreterArgs<true>&);

///////////////////////////////////////////////////////////

namespace op {

using internal::Context;
using internal::kMaxGas;

struct OpInfo {
  int32_t pops;
  int32_t pushes;
  int32_t staticGas;
  int32_t instructionLength = 1;
  bool isJump = false;

  constexpr int32_t GetStackDelta() const { return pushes - pops; }
};

template <OpCode op_code, typename = void>
struct Impl : public std::false_type {};

template <>
struct Impl<OpCode::STOP> : public std::true_type {
  constexpr static OpInfo kInfo{};
  static RunState Run() noexcept { return RunState::kDone; }
};

static void stop(Context& ctx) noexcept { ctx.state = RunState::kDone; }

template <>
struct Impl<OpCode::ADD> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] += top[0]; }
};

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

template <>
struct Impl<OpCode::MUL> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 5,
  };

  static void Run(uint256_t* top) noexcept { top[1] *= top[0]; }
};

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

template <>
struct Impl<OpCode::SUB> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] = top[0] - top[1]; }
};

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

template <>
struct Impl<OpCode::DIV> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 5,
  };

  static void Run(uint256_t* top) noexcept {
    if (top[1] != 0) {
      top[1] = top[0] / top[1];
    }
  }
};

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

template <>
struct Impl<OpCode::SDIV> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 5,
  };

  static void Run(uint256_t* top) noexcept {
    if (top[1] != 0) {
      top[1] = intx::sdivrem(top[0], top[1]).quot;
    }
  }
};

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

template <>
struct Impl<OpCode::MOD> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 5,
  };

  static void Run(uint256_t* top) noexcept {
    if (top[1] != 0) {
      top[1] = top[0] % top[1];
    }
  }
};

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

template <>
struct Impl<OpCode::LT> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] = top[0] < top[1] ? 1 : 0; }
};

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

template <>
struct Impl<OpCode::GT> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] = top[0] > top[1] ? 1 : 0; }
};

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

template <>
struct Impl<OpCode::SLT> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] = intx::slt(top[0], top[1]) ? 1 : 0; }
};

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

template <>
struct Impl<OpCode::EQ> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] = top[0] == top[1] ? 1 : 0; }
};

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

template <>
struct Impl<OpCode::ISZERO> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 1,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[0] = top[0] == 0; }
};

static void iszero(Context& ctx) noexcept {
  if (!ctx.CheckStackAvailable(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;
  uint256_t val = ctx.stack.Pop();
  ctx.stack.Push(val == 0);
  ctx.pc++;
}

template <>
struct Impl<OpCode::AND> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] = top[0] & top[1]; }
};

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

template <>
struct Impl<OpCode::SHR> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { top[1] >>= top[0]; }
};

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

  auto memory_span = ctx.memory.GetSpan(offset, size);
  if (ctx.sha3_cache) {
    ctx.stack.Push(ctx.sha3_cache->Hash(memory_span));
  } else {
    ctx.stack.Push(ToUint256(ethash::keccak256(memory_span.data(), memory_span.size())));
  }

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
  ctx.stack.Push(ctx.padded_code.size() - kStopBytePadding);
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
  if (code_offset_u256 < ctx.padded_code.size() - kStopBytePadding) {
    code_view = std::span(ctx.padded_code).subspan(static_cast<uint64_t>(code_offset_u256));
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

template <>
struct Impl<OpCode::POP> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 1,
      .pushes = 0,
      .staticGas = 2,
  };

  static void Run() noexcept {}
};

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

template <>
struct Impl<OpCode::JUMP> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 1,
      .pushes = 0,
      .staticGas = 8,
      .isJump = true,
  };

  static bool Run(uint256_t*) noexcept { return true; }
};

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

template <>
struct Impl<OpCode::JUMPI> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 0,
      .staticGas = 10,
      .isJump = true,
  };

  static bool Run(uint256_t* top) noexcept {
    const uint256_t& b = top[1];
    return b != 0;
  }
};

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

template <>
struct Impl<OpCode::JUMPDEST> : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 0,
      .pushes = 0,
      .staticGas = 1,
  };

  static void Run() noexcept {}
};

static void jumpdest(Context& ctx) noexcept {
  if (!ctx.ApplyGasCost(1)) [[unlikely]]
    return;
  ctx.pc++;
}

template <uint64_t N>
struct PushImpl : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = 0,
      .pushes = 1,
      .staticGas = 3,
      .instructionLength = 1 + N,
  };

  static void Run(uint256_t* top, const uint8_t* data) noexcept {
    constexpr auto num_full_words = N / sizeof(uint64_t);
    constexpr auto num_partial_bytes = N % sizeof(uint64_t);

    // TODO: hide stack details.
    uint256_t& value = *(--top);
    value = 0;
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
        value[num_full_words - 1 - i] = intx::bswap(*reinterpret_cast<const uint64_t*>(data));
      } else {
        value[num_full_words - 1 - i] = *reinterpret_cast<const uint64_t*>(data);
      }
      data += sizeof(uint64_t);
    }
  }
};

template <op::OpCode op_code>
struct Impl<op_code, std::enable_if_t<OpCode::PUSH1 <= op_code && op_code <= OpCode::PUSH32>>
    : public PushImpl<static_cast<uint64_t>(op_code - OpCode::PUSH1 + 1)> {};

template <uint64_t N>
static void push(Context& ctx) noexcept {
  if (!ctx.CheckStackOverflow(1)) [[unlikely]]
    return;
  if (!ctx.ApplyGasCost(3)) [[unlikely]]
    return;

  constexpr auto num_full_words = N / sizeof(uint64_t);
  constexpr auto num_partial_bytes = N % sizeof(uint64_t);
  auto data = &ctx.padded_code[ctx.pc + 1];

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
      value[num_full_words - 1 - i] = intx::bswap(*reinterpret_cast<const uint64_t*>(data));
    } else {
      value[num_full_words - 1 - i] = *reinterpret_cast<const uint64_t*>(data);
    }
    data += sizeof(uint64_t);
  }

  ctx.stack.Push(value);
  ctx.pc += 1 + N;
}

template <uint64_t N>
struct DupImpl : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = N,
      .pushes = N + 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept { *(top - 1) = top[N - 1]; }
};

template <op::OpCode op_code>
struct Impl<op_code, std::enable_if_t<OpCode::DUP1 <= op_code && op_code <= OpCode::DUP16>>
    : public DupImpl<static_cast<uint64_t>(op_code - OpCode::DUP1 + 1)> {};

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
struct SwapImpl : public std::true_type {
  constexpr static OpInfo kInfo{
      .pops = N + 1,
      .pushes = N + 1,
      .staticGas = 3,
  };

  static void Run(uint256_t* top) noexcept {
    auto tmp = top[N];
    top[N] = top[0];
    top[0] = tmp;
  }
};

template <op::OpCode op_code>
struct Impl<op_code, std::enable_if_t<OpCode::SWAP1 <= op_code && op_code <= OpCode::SWAP16>>
    : public SwapImpl<static_cast<uint64_t>(op_code - OpCode::SWAP1 + 1)> {};

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

template <op::OpCode Op>
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

template <op::OpCode Op>
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

inline bool Context::CheckJumpDest(uint256_t index_u256) noexcept {
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

std::vector<uint8_t> PadCode(std::span<const uint8_t> code) {
  std::vector<uint8_t> padded;
  padded.reserve(code.size() + kStopBytePadding);
  padded.assign(code.begin(), code.end());
  padded.resize(code.size() + kStopBytePadding, op::STOP);
  return padded;
}

inline RunState Invoke(uint256_t*, const uint8_t*, void (*op)() noexcept) noexcept { 
  op(); 
  return RunState::kRunning;
}

inline RunState Invoke(uint256_t*, const uint8_t*, RunState (*op)() noexcept) noexcept { return op(); }

inline RunState Invoke(uint256_t* top, const uint8_t*, void (*op)(uint256_t*) noexcept) noexcept { 
  op(top); 
  return RunState::kRunning;
}

inline RunState Invoke(uint256_t* top, const uint8_t* data,
                   void (*op)(uint256_t* top, const uint8_t* data) noexcept) noexcept {
  op(top, data + 1);  // Data of push is off-set by 1
  return RunState::kRunning;
}

struct Result {
  RunState state;
  uint32_t pc;
  int64_t gas_left;
  uint256_t* top;
};

template <op::OpCode op_code>
constexpr static bool kHasImplType = op::Impl<op_code>::value;

template <op::OpCode op_code>
inline Result Run(uint32_t pc, int64_t gas, uint256_t* base, uint256_t* top, const uint8_t* code, Context& ctx,
                  void (*legacy)(Context&) noexcept) {
  // If the new experimental operator implementation is available use that one.
  if constexpr (kHasImplType<op_code>) {
    // TODO: factor out stack implementation details.
    using Impl = op::Impl<op_code>;

    // Check stack requirements.
    auto size = base - top;
    if constexpr (Impl::kInfo.pops > 0) {
      if (size < Impl::kInfo.pops) [[unlikely]] {
        return Result{.state = RunState::kErrorStackUnderflow};
      }
    }
    if constexpr (Impl::kInfo.GetStackDelta() > 0) {
      if (1024 - size < Impl::kInfo.GetStackDelta()) [[unlikely]] {
        return Result{.state = RunState::kErrorStackOverflow};
      }
    }
    // Charge static gas costs.
    if (gas < Impl::kInfo.staticGas) [[unlikely]] {
      return Result{.state = RunState::kErrorGas};
    }
    gas -= Impl::kInfo.staticGas;

    // Run the operation.
    RunState state = RunState::kRunning;
    if constexpr (Impl::kInfo.isJump) {
      if (Impl::Run(top)) {
        if (!ctx.CheckJumpDest(*top)) [[unlikely]] {
          return Result{.state = ctx.state};
        }
        pc = static_cast<uint32_t>(*top);
      } else {
        pc += 1;
      }
    } else {
      state = Invoke(top, code + pc, Impl::Run);
      pc += Impl::kInfo.instructionLength;
    }

    // Update the stack.
    top -= Impl::kInfo.GetStackDelta();
    return Result{
        .state = state,
        .pc = pc,
        .gas_left = gas,
        .top = top,
    };
  }

  // If there is no type-based implementation, fall back to the legacy version.
  else {
    // std::cout << "Missing: " << ToString(op_code) << "\n";
    //  Update context.
    ctx.stack.SetTop(top);
    ctx.pc = pc;
    ctx.gas = gas;
    // Run legacy version of operation.
    legacy(ctx);
    // Extract information from context.
    return Result{
        .state = ctx.state,
        .pc = static_cast<uint32_t>(ctx.pc),
        .gas_left = ctx.gas,
        .top = ctx.stack.Peek(),
    };
  }
}

template <bool LoggingEnabled, bool ProfilingEnabled>
void RunInterpreter(Context& ctx, Profiler<ProfilingEnabled>& profiler) {
  EVMZERO_PROFILE_ZONE();

  // The state, pc, and stack state are owned by this function and
  // should not escape this function.
  RunState state = RunState::kRunning;
  uint32_t pc = 0;
  int64_t gas = ctx.gas;
  uint256_t* base = ctx.stack.Base();
  uint256_t* top = ctx.stack.Peek();

#define PROFILE_START(marker) profiler.template Start<Marker::marker>()
#define PROFILE_END(marker) profiler.template End<Marker::marker>()
#define RUN(opcode, impl)                                                       \
  {                                                                             \
    auto res = Run<op::opcode>(pc, gas, base, top, padded_code, ctx, op::impl); \
    state = res.state;                                                          \
    pc = res.pc;                                                                \
    gas = res.gas_left;                                                         \
    top = res.top;                                                              \
  }
#define OPCODE(opcode, impl)         \
  op::opcode : {                     \
    EVMZERO_PROFILE_ZONE_N(#opcode); \
    PROFILE_START(opcode);           \
    RUN(opcode, impl);               \
    PROFILE_END(opcode);             \
  }

  auto padded_code = ctx.padded_code.data();
  while (state == RunState::kRunning) {
    if constexpr (LoggingEnabled) {
      // log format: <op>, <gas>, <top-of-stack>\n
      std::cout << ToString(static_cast<op::OpCode>(padded_code[pc])) << ", "  //
                << ctx.gas << ", ";
      if (ctx.stack.GetSize() == 0) {
        std::cout << "-empty-";
      } else {
        std::cout << ctx.stack[0];
      }
      std::cout << "\n" << std::flush;
    }

    switch (padded_code[pc]) {
      // clang-format off
      case OPCODE(STOP, stop); break;

      case OPCODE(ADD, add); break;
      case OPCODE(MUL, mul); break;
      case OPCODE(SUB, sub); break;
      case OPCODE(DIV, div); break;
      case OPCODE(SDIV, sdiv); break;
      case OPCODE(MOD, mod); break;
      case OPCODE(SMOD, smod); break;
      case OPCODE(ADDMOD, addmod); break;
      case OPCODE(MULMOD, mulmod); break;
      case OPCODE(EXP, exp); break;
      case OPCODE(SIGNEXTEND, signextend); break;
      case OPCODE(LT, lt); break;
      case OPCODE(GT, gt); break;
      case OPCODE(SLT, slt); break;
      case OPCODE(SGT, sgt); break;
      case OPCODE(EQ, eq); break;
      case OPCODE(ISZERO, iszero); break;
      case OPCODE(AND, bit_and); break;
      case OPCODE(OR, bit_or); break;
      case OPCODE(XOR, bit_xor); break;
      case OPCODE(NOT, bit_not); break;
      case OPCODE(BYTE, byte); break;
      case OPCODE(SHL, shl); break;
      case OPCODE(SHR, shr); break;
      case OPCODE(SAR, sar); break;
      case OPCODE(SHA3, sha3); break;
      case OPCODE(ADDRESS, address); break;
      case OPCODE(BALANCE, balance); break;
      case OPCODE(ORIGIN, origin); break;
      case OPCODE(CALLER, caller); break;
      case OPCODE(CALLVALUE, callvalue); break;
      case OPCODE(CALLDATALOAD, calldataload); break;
      case OPCODE(CALLDATASIZE, calldatasize); break;
      case OPCODE(CALLDATACOPY, calldatacopy); break;
      case OPCODE(CODESIZE, codesize); break;
      case OPCODE(CODECOPY, codecopy); break;
      case OPCODE(GASPRICE, gasprice); break;
      case OPCODE(EXTCODESIZE, extcodesize); break;
      case OPCODE(EXTCODECOPY, extcodecopy); break;
      case OPCODE(RETURNDATASIZE, returndatasize); break;
      case OPCODE(RETURNDATACOPY, returndatacopy); break;
      case OPCODE(EXTCODEHASH, extcodehash); break;
      case OPCODE(BLOCKHASH, blockhash); break;
      case OPCODE(COINBASE, coinbase); break;
      case OPCODE(TIMESTAMP, timestamp); break;
      case OPCODE(NUMBER, blocknumber); break;
      case OPCODE(DIFFICULTY, prevrandao); break; // intentional
      case OPCODE(GASLIMIT, gaslimit); break;
      case OPCODE(CHAINID, chainid); break;
      case OPCODE(SELFBALANCE, selfbalance); break;
      case OPCODE(BASEFEE, basefee); break;

      case OPCODE(POP, pop); break;
      case OPCODE(MLOAD, mload); break;
      case OPCODE(MSTORE, mstore); break;
      case OPCODE(MSTORE8, mstore8); break;
      case OPCODE(SLOAD, sload); break;
      case OPCODE(SSTORE, sstore); break;

      case OPCODE(JUMP, jump); break;
      case OPCODE(JUMPI, jumpi); break;
      case OPCODE(PC, pc); break;
      case OPCODE(MSIZE, msize); break;
      case OPCODE(GAS, gas); break;
      case OPCODE(JUMPDEST, jumpdest); break;

      case OPCODE(PUSH1, push<1>); break;
      case OPCODE(PUSH2, push<2>); break;
      case OPCODE(PUSH3, push<3>); break;
      case OPCODE(PUSH4, push<4>); break;
      case OPCODE(PUSH5, push<5>); break;
      case OPCODE(PUSH6, push<6>); break;
      case OPCODE(PUSH7, push<7>); break;
      case OPCODE(PUSH8, push<8>); break;
      case OPCODE(PUSH9, push<9>); break;
      case OPCODE(PUSH10, push<10>); break;
      case OPCODE(PUSH11, push<11>); break;
      case OPCODE(PUSH12, push<12>); break;
      case OPCODE(PUSH13, push<13>); break;
      case OPCODE(PUSH14, push<14>); break;
      case OPCODE(PUSH15, push<15>); break;
      case OPCODE(PUSH16, push<16>); break;
      case OPCODE(PUSH17, push<17>); break;
      case OPCODE(PUSH18, push<18>); break;
      case OPCODE(PUSH19, push<19>); break;
      case OPCODE(PUSH20, push<20>); break;
      case OPCODE(PUSH21, push<21>); break;
      case OPCODE(PUSH22, push<22>); break;
      case OPCODE(PUSH23, push<23>); break;
      case OPCODE(PUSH24, push<24>); break;
      case OPCODE(PUSH25, push<25>); break;
      case OPCODE(PUSH26, push<26>); break;
      case OPCODE(PUSH27, push<27>); break;
      case OPCODE(PUSH28, push<28>); break;
      case OPCODE(PUSH29, push<29>); break;
      case OPCODE(PUSH30, push<30>); break;
      case OPCODE(PUSH31, push<31>); break;
      case OPCODE(PUSH32, push<32>); break;

      case OPCODE(DUP1, dup<1>); break;
      case OPCODE(DUP2, dup<2>); break;
      case OPCODE(DUP3, dup<3>); break;
      case OPCODE(DUP4, dup<4>); break;
      case OPCODE(DUP5, dup<5>); break;
      case OPCODE(DUP6, dup<6>); break;
      case OPCODE(DUP7, dup<7>); break;
      case OPCODE(DUP8, dup<8>); break;
      case OPCODE(DUP9, dup<9>); break;
      case OPCODE(DUP10, dup<10>); break;
      case OPCODE(DUP11, dup<11>); break;
      case OPCODE(DUP12, dup<12>); break;
      case OPCODE(DUP13, dup<13>); break;
      case OPCODE(DUP14, dup<14>); break;
      case OPCODE(DUP15, dup<15>); break;
      case OPCODE(DUP16, dup<16>); break;

      case OPCODE(SWAP1, swap<1>); break;
      case OPCODE(SWAP2, swap<2>); break;
      case OPCODE(SWAP3, swap<3>); break;
      case OPCODE(SWAP4, swap<4>); break;
      case OPCODE(SWAP5, swap<5>); break;
      case OPCODE(SWAP6, swap<6>); break;
      case OPCODE(SWAP7, swap<7>); break;
      case OPCODE(SWAP8, swap<8>); break;
      case OPCODE(SWAP9, swap<9>); break;
      case OPCODE(SWAP10, swap<10>); break;
      case OPCODE(SWAP11, swap<11>); break;
      case OPCODE(SWAP12, swap<12>); break;
      case OPCODE(SWAP13, swap<13>); break;
      case OPCODE(SWAP14, swap<14>); break;
      case OPCODE(SWAP15, swap<15>); break;
      case OPCODE(SWAP16, swap<16>); break;

      case OPCODE(LOG0, log<0>); break;
      case OPCODE(LOG1, log<1>); break;
      case OPCODE(LOG2, log<2>); break;
      case OPCODE(LOG3, log<3>); break;
      case OPCODE(LOG4, log<4>); break;

      case op::CREATE: RUN(CREATE, create_impl<op::CREATE>); break;
      case op::CREATE2: RUN(CREATE2, create_impl<op::CREATE2>); break;

      case OPCODE(RETURN, return_op<RunState::kReturn>); break;
      case OPCODE(REVERT, return_op<RunState::kRevert>); break;

      case op::CALL: RUN(CALL, call_impl<op::CALL>); break;
      case op::CALLCODE: RUN(CALLCODE, call_impl<op::CALLCODE>); break;
      case op::DELEGATECALL: RUN(DELEGATECALL, call_impl<op::DELEGATECALL>); break;
      case op::STATICCALL: RUN(STATICCALL, call_impl<op::STATICCALL>); break;

      case OPCODE(INVALID, invalid); break;
      case OPCODE(SELFDESTRUCT, selfdestruct); break;

      default:
        state = RunState::kErrorOpcode;

        // clang-format on
    }
  }

  if (IsSuccess(state)) {
    ctx.gas = gas;
  } else {
    ctx.gas = 0;
  }

  // Keep return data only when we are supposed to return something.
  if (state != RunState::kReturn && state != RunState::kRevert) {
    ctx.return_data.clear();
  }

  ctx.state = state;
  ctx.stack.SetTop(top);

#undef PROFILE_START
#undef PROFILE_END
#undef RUN
#undef OPCODE
}

template void RunInterpreter<false, false>(Context&, Profiler<false>&);
template void RunInterpreter<true, false>(Context&, Profiler<false>&);
template void RunInterpreter<false, true>(Context&, Profiler<true>&);
template void RunInterpreter<true, true>(Context&, Profiler<true>&);

}  // namespace internal

}  // namespace tosca::evmzero

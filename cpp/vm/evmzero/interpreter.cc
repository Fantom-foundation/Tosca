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

struct OpResult {
  RunState state = RunState::kRunning;
  uint32_t pc = 0;
  int64_t gas_left = 0;
  uint256_t* stack_top = 0;
};

#define CHECK_STACK_AVAILABLE(stack_size, elements_needed) \
  if ((stack_size) < (elements_needed)) [[unlikely]] {     \
    return {.state = RunState::kErrorStackUnderflow};      \
  }

#define CHECK_STACK_OVERFLOW(stack_size, slots_needed)     \
  if (1024 - (stack_size) < (slots_needed)) [[unlikely]] { \
    return {.state = RunState::kErrorStackOverflow};       \
  }

#define CHECK_STATIC_CALL_CONFORMANCE(context)    \
  if ((context).is_static_call) [[unlikely]] {    \
    return {.state = RunState::kErrorStaticCall}; \
  }

#define APPLY_GAS_COST(gas_counter, amount)                        \
  if ((gas_counter) -= (amount); (gas_counter) < 0) [[unlikely]] { \
    return {.state = RunState::kErrorGas};                         \
  }

static OpResult stop(uint32_t pc, int64_t gas, uint256_t* stack_top) noexcept {
  return {
      .state = RunState::kDone,
      .pc = pc,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult add(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] += stack_top[0];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult mul(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 5);

  stack_top[1] *= stack_top[0];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult sub(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[0] - stack_top[1];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult div(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 5);

  if (stack_top[1] == 0) {
    stack_top[1] = 0;
  } else {
    stack_top[1] = stack_top[0] / stack_top[1];
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult sdiv(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 5);

  if (stack_top[1] == 0) {
    stack_top[1] = 0;
  } else {
    stack_top[1] = intx::sdivrem(stack_top[0], stack_top[1]).quot;
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult mod(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 5);

  if (stack_top[1] == 0) {
    stack_top[1] = 0;
  } else {
    stack_top[1] = stack_top[0] % stack_top[1];
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult smod(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 5);

  if (stack_top[1] == 0) {
    stack_top[1] = 0;
  } else {
    stack_top[1] = intx::sdivrem(stack_top[0], stack_top[1]).rem;
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult addmod(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 3);
  APPLY_GAS_COST(gas, 8);

  if (stack_top[2] == 0) {
    stack_top[2] = 0;
  } else {
    stack_top[2] = intx::addmod(stack_top[0], stack_top[1], stack_top[2]);
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 2,
  };
}

static OpResult mulmod(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 3);
  APPLY_GAS_COST(gas, 8);

  if (stack_top[2] == 0) {
    stack_top[2] = 0;
  } else {
    stack_top[2] = intx::mulmod(stack_top[0], stack_top[1], stack_top[2]);
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 2,
  };
}

static OpResult exp(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 10);

  APPLY_GAS_COST(gas, 50 * intx::count_significant_bytes(stack_top[1]))

  stack_top[1] = intx::exp(stack_top[0], stack_top[1]);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult signextend(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 5);

  uint8_t leading_byte_index = static_cast<uint8_t>(stack_top[0]);
  if (leading_byte_index > 31) {
    leading_byte_index = 31;
  }

  bool is_negative = ToByteArrayLe(stack_top[1])[leading_byte_index] & 0b1000'0000;
  if (is_negative) {
    auto mask = kUint256Max << (8 * (leading_byte_index + 1));
    stack_top[1] = mask | stack_top[1];
  } else {
    auto mask = kUint256Max >> (8 * (31 - leading_byte_index));
    stack_top[1] = mask & stack_top[1];
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult lt(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[0] < stack_top[1] ? 1 : 0;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult gt(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[0] > stack_top[1] ? 1 : 0;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult slt(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = intx::slt(stack_top[0], stack_top[1]) ? 1 : 0;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult sgt(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = intx::slt(stack_top[1], stack_top[0]) ? 1 : 0;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult eq(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[0] == stack_top[1] ? 1 : 0;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult iszero(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 3);

  stack_top[0] = stack_top[0] == 0;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult bit_and(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[0] & stack_top[1];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult bit_or(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[0] | stack_top[1];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult bit_xor(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[0] ^ stack_top[1];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult bit_not(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 3);

  stack_top[0] = ~stack_top[0];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult byte(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  if (stack_top[0] < 32) {
    stack_top[1] = ToByteArrayLe(stack_top[1])[31 - static_cast<uint8_t>(stack_top[0])];
  } else {
    stack_top[1] = 0;
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult shl(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] = stack_top[1] << stack_top[0];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult shr(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  stack_top[1] >>= stack_top[0];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult sar(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  const bool is_negative = ToByteArrayLe(stack_top[1])[31] & 0b1000'0000;

  if (stack_top[0] <= 255) {
    stack_top[1] >>= stack_top[0];
    if (is_negative) {
      stack_top[1] |= (kUint256Max << (255 - stack_top[0]));
    }
  } else {
    stack_top[1] = 0;
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult sha3(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                     Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 30);

  const uint256_t offset_u256 = stack_top[0];
  const uint256_t size_u256 = stack_top[1];

  const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  APPLY_GAS_COST(gas, 6 * minimum_word_size);

  auto memory_span = ctx.memory.GetSpan(offset, size);
  if (ctx.sha3_cache) {
    stack_top[1] = ctx.sha3_cache->Hash(memory_span);
  } else {
    stack_top[1] = ToUint256(ethash::keccak256(memory_span.data(), memory_span.size()));
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult address(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                        Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.message->recipient);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult balance(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                        Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);

  evmc::address address = ToEvmcAddress(stack_top[0]);

  int64_t dynamic_gas_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2600;
    }
  }
  APPLY_GAS_COST(gas, dynamic_gas_cost);

  stack_top[0] = ToUint256(ctx.host->get_balance(address));

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult origin(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                       Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.host->get_tx_context().tx_origin);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult caller(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                       Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.message->sender);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult callvalue(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                          Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.message->value);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult calldataload(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                             Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 3);

  const uint256_t offset_u256 = stack_top[0];

  std::span<const uint8_t> input_view;
  if (offset_u256 < ctx.message->input_size) {
    input_view = std::span(ctx.message->input_data, ctx.message->input_size)  //
                     .subspan(static_cast<uint64_t>(offset_u256));
  }

  evmc::bytes32 value{};
  std::copy_n(input_view.begin(), std::min<size_t>(input_view.size(), 32), value.bytes);

  stack_top[0] = ToUint256(value);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult calldatasize(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                             Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ctx.message->input_size;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult calldatacopy(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                             Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 3);
  APPLY_GAS_COST(gas, 3);

  const uint256_t memory_offset_u256 = stack_top[0];
  const uint256_t data_offset_u256 = stack_top[1];
  const uint256_t size_u256 = stack_top[2];

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  APPLY_GAS_COST(gas, 3 * minimum_word_size);

  std::span<const uint8_t> data_view;
  if (data_offset_u256 < ctx.message->input_size) {
    data_view = std::span(ctx.message->input_data, ctx.message->input_size)  //
                    .subspan(static_cast<uint64_t>(data_offset_u256));
  }

  ctx.memory.ReadFromWithSize(data_view, memory_offset, size);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 3,
  };
}

static OpResult codesize(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                         Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ctx.padded_code.size() - kStopBytePadding;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult codecopy(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                         Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 3);
  APPLY_GAS_COST(gas, 3);

  const uint256_t memory_offset_u256 = stack_top[0];
  const uint256_t code_offset_u256 = stack_top[1];
  const uint256_t size_u256 = stack_top[2];

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  APPLY_GAS_COST(gas, 3 * minimum_word_size);

  std::span<const uint8_t> code_view;
  if (code_offset_u256 < ctx.padded_code.size() - kStopBytePadding) {
    code_view = std::span(ctx.padded_code).subspan(static_cast<uint64_t>(code_offset_u256));
  }

  ctx.memory.ReadFromWithSize(code_view, memory_offset, size);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 3,
  };
}

static OpResult gasprice(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                         Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.host->get_tx_context().tx_gas_price);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult extcodesize(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                            Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);

  auto address = ToEvmcAddress(stack_top[0]);

  int64_t dynamic_gas_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2600;
    }
  }
  APPLY_GAS_COST(gas, dynamic_gas_cost);

  stack_top[0] = ctx.host->get_code_size(address);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult extcodecopy(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                            Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 4);

  const auto address = ToEvmcAddress(stack_top[0]);
  const uint256_t memory_offset_u256 = stack_top[1];
  const uint256_t code_offset_u256 = stack_top[2];
  const uint256_t size_u256 = stack_top[3];

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  int64_t address_access_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      address_access_cost = 100;
    } else {
      address_access_cost = 2600;
    }
  }
  APPLY_GAS_COST(gas, 3 * minimum_word_size + address_access_cost);

  auto memory_span = ctx.memory.GetSpan(memory_offset, size);
  if (code_offset_u256 <= std::numeric_limits<uint64_t>::max()) {
    uint64_t code_offset = static_cast<uint64_t>(code_offset_u256);
    size_t bytes_written = ctx.host->copy_code(address, code_offset, memory_span.data(), memory_span.size());
    memory_span = memory_span.subspan(bytes_written);
  }
  std::fill(memory_span.begin(), memory_span.end(), 0);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 4,
  };
}

static OpResult returndatasize(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                               Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ctx.return_data.size();

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult returndatacopy(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                               Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 3);
  APPLY_GAS_COST(gas, 3);

  const uint256_t memory_offset_u256 = stack_top[0];
  const uint256_t return_data_offset_u256 = stack_top[1];
  const uint256_t size_u256 = stack_top[2];

  const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
  APPLY_GAS_COST(gas, 3 * minimum_word_size);

  {
    const auto [end_u256, carry] = intx::addc(return_data_offset_u256, size_u256);
    if (carry || end_u256 > ctx.return_data.size()) {
      return {.state = RunState::kErrorReturnDataCopyOutOfBounds};
    }
  }

  std::span<const uint8_t> return_data_view = std::span(ctx.return_data)  //
                                                  .subspan(static_cast<uint64_t>(return_data_offset_u256));
  ctx.memory.ReadFromWithSize(return_data_view, memory_offset, size);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 3,
  };
}

static OpResult extcodehash(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                            Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);

  auto address = ToEvmcAddress(stack_top[0]);

  int64_t dynamic_gas_cost = 700;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2600;
    }
  }
  APPLY_GAS_COST(gas, dynamic_gas_cost);

  stack_top[0] = ToUint256(ctx.host->get_code_hash(address));

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult blockhash(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                          Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 20);

  int64_t number = static_cast<int64_t>(stack_top[0]);
  stack_top[0] = ToUint256(ctx.host->get_block_hash(number));

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult coinbase(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                         Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.host->get_tx_context().block_coinbase);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult timestamp(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                          Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ctx.host->get_tx_context().block_timestamp;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult blocknumber(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                            Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ctx.host->get_tx_context().block_number;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult prevrandao(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                           Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.host->get_tx_context().block_prev_randao);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult gaslimit(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                         Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ctx.host->get_tx_context().block_gas_limit;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult chainid(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                        Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.host->get_tx_context().chain_id);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult selfbalance(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                            Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 5);

  stack_top[-1] = ToUint256(ctx.host->get_balance(ctx.message->recipient));

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult basefee(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                        Context& ctx) noexcept {
  if (ctx.revision < EVMC_LONDON) [[unlikely]]
    return {.state = RunState::kErrorOpcode};

  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ToUint256(ctx.host->get_tx_context().block_base_fee);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult pop(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);
  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult mload(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                      Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 3);

  const uint256_t offset_u256 = stack_top[0];

  const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 32);
  APPLY_GAS_COST(gas, mem_cost);

  uint256_t value;
  ctx.memory.WriteTo({ToBytes(value), 32}, offset);

  if constexpr (std::endian::native == std::endian::little) {
    value = intx::bswap(value);
  }

  stack_top[0] = value;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult mstore(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                       Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  const uint256_t offset_u256 = stack_top[0];
  uint256_t value = stack_top[1];

  const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 32);
  APPLY_GAS_COST(gas, mem_cost);

  if constexpr (std::endian::native == std::endian::little) {
    value = intx::bswap(value);
  }

  ctx.memory.ReadFrom({ToBytes(value), 32}, offset);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 2,
  };
}

static OpResult mstore8(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                        Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 3);

  const uint256_t offset_u256 = stack_top[0];
  const uint8_t value = static_cast<uint8_t>(stack_top[1]);

  const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 1);
  APPLY_GAS_COST(gas, mem_cost);

  ctx.memory.ReadFrom({&value, 1}, offset);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 2,
  };
}

static OpResult sload(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                      Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);

  const uint256_t key = stack_top[0];

  int64_t dynamic_gas_cost = 800;
  if (ctx.revision >= EVMC_BERLIN) {
    if (ctx.host->access_storage(ctx.message->recipient, ToEvmcBytes(key)) == EVMC_ACCESS_WARM) {
      dynamic_gas_cost = 100;
    } else {
      dynamic_gas_cost = 2100;
    }
  }
  APPLY_GAS_COST(gas, dynamic_gas_cost);

  stack_top[0] = ToUint256(ctx.host->get_storage(ctx.message->recipient, ToEvmcBytes(key)));

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult sstore(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                       Context& ctx) noexcept {
  // EIP-2200
  if (gas <= 2300) [[unlikely]] {
    return {.state = RunState::kErrorGas};
  }

  CHECK_STATIC_CALL_CONFORMANCE(ctx);
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);

  const uint256_t key = stack_top[0];
  const uint256_t value = stack_top[1];

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

  APPLY_GAS_COST(gas, dynamic_gas_cost);

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

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 2,
  };
}

static OpResult jump(uint32_t, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top, Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 8);

  if (!ctx.CheckJumpDest(stack_top[0])) [[unlikely]]
    return {.state = RunState::kErrorJump};

  return {
      .pc = static_cast<uint32_t>(stack_top[0]),
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

static OpResult jumpi(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                      Context& ctx) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);
  APPLY_GAS_COST(gas, 10);

  if (stack_top[1] != 0) {
    if (!ctx.CheckJumpDest(stack_top[0])) [[unlikely]]
      return {.state = RunState::kErrorJump};
    return {
        .pc = static_cast<uint32_t>(stack_top[0]),
        .gas_left = gas,
        .stack_top = stack_top + 2,
    };
  } else {
    return {
        .pc = pc + 1,
        .gas_left = gas,
        .stack_top = stack_top + 2,
    };
  }
}

static OpResult pc(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = pc;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult msize(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                      Context& ctx) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = ctx.memory.GetSize();

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult gas(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 2);

  stack_top[-1] = gas;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

static OpResult jumpdest(uint32_t pc, int64_t gas, const uint256_t*, uint256_t* stack_top) noexcept {
  APPLY_GAS_COST(gas, 1);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

template <uint32_t N>
static OpResult push(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                     const uint8_t* padded_code) noexcept {
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 3);

  constexpr auto num_full_words = N / sizeof(uint64_t);
  constexpr auto num_partial_bytes = N % sizeof(uint64_t);
  auto data = padded_code + pc + 1;

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

  stack_top[-1] = value;

  return {
      .pc = pc + 1 + N,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

template <uint32_t N>
static OpResult dup(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, N);
  CHECK_STACK_OVERFLOW(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 3);

  stack_top[-1] = stack_top[N - 1];

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

template <uint32_t N>
static OpResult swap(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top) noexcept {
  CHECK_STACK_AVAILABLE(stack_base - stack_top, N + 1);
  APPLY_GAS_COST(gas, 3);

  std::swap(stack_top[0], stack_top[N]);

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

template <uint32_t N>
static OpResult log(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                    Context& ctx) noexcept {
  CHECK_STATIC_CALL_CONFORMANCE(ctx);
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2 + N);
  APPLY_GAS_COST(gas, 375);

  const uint256_t offset_u256 = stack_top[0];
  const uint256_t size_u256 = stack_top[1];

  const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  std::array<evmc::bytes32, N> topics;
  for (unsigned i = 0; i < N; ++i) {
    topics[i] = ToEvmcBytes(stack_top[2 + i]);
  }

  APPLY_GAS_COST(gas, static_cast<int64_t>(375 * N + 8 * size));

  auto data = ctx.memory.GetSpan(offset, size);

  ctx.host->emit_log(ctx.message->recipient, data.data(), data.size(), topics.data(), topics.size());

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top + 2 + N,
  };
}

template <RunState result_state>
static OpResult return_op(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                          Context& ctx) noexcept {
  static_assert(result_state == RunState::kReturn || result_state == RunState::kRevert);

  CHECK_STACK_AVAILABLE(stack_base - stack_top, 2);

  const uint256_t offset_u256 = stack_top[0];
  const uint256_t size_u256 = stack_top[1];

  const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  ctx.return_data.resize(size);
  ctx.memory.WriteTo(ctx.return_data, offset);

  return {
      .state = result_state,
      .pc = pc,
      .gas_left = gas,
      .stack_top = stack_top + 2,
  };
}

static OpResult invalid(uint32_t pc, int64_t gas, const uint256_t*, uint256_t* stack_top) noexcept {
  return {
      .state = RunState::kInvalid,
      .pc = pc,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

static OpResult selfdestruct(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                             Context& ctx) noexcept {
  CHECK_STATIC_CALL_CONFORMANCE(ctx);
  CHECK_STACK_AVAILABLE(stack_base - stack_top, 1);
  APPLY_GAS_COST(gas, 5000);

  auto account = ToEvmcAddress(stack_top[0]);

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
    APPLY_GAS_COST(gas, dynamic_gas_cost);
  }

  if (ctx.host->selfdestruct(ctx.message->recipient, account)) {
    if (ctx.revision < EVMC_LONDON) {
      ctx.gas_refunds += 24000;
    }
  }

  return {
      .state = RunState::kDone,
      .pc = pc,
      .gas_left = gas,
      .stack_top = stack_top + 1,
  };
}

template <op::OpCodes Op>
static OpResult create_impl(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                            Context& ctx) noexcept {
  static_assert(Op == op::CREATE || Op == op::CREATE2);

  if (ctx.message->depth >= 1024) [[unlikely]] {
    return {.state = RunState::kErrorCreate};
  }

  CHECK_STATIC_CALL_CONFORMANCE(ctx);
  CHECK_STACK_AVAILABLE(stack_base - stack_top, (Op == op::CREATE2) ? 4 : 3);
  APPLY_GAS_COST(gas, 32000);

  // TODO Refactor using dedicated stack pop.

  const auto endowment = stack_top[0];
  const uint256_t init_code_offset_u256 = stack_top[1];
  const uint256_t init_code_size_u256 = stack_top[2];
  const auto salt = (Op == op::CREATE2) ? stack_top[3] : uint256_t{0};

  // Set up stack pointer for result value.
  stack_top += (Op == op::CREATE2) ? 3 : 2;

  const auto [mem_cost, init_code_offset, init_code_size] =
      ctx.MemoryExpansionCost(init_code_offset_u256, init_code_size_u256);
  APPLY_GAS_COST(gas, mem_cost);

  if constexpr (Op == op::CREATE2) {
    const int64_t minimum_word_size = static_cast<int64_t>((init_code_size + 31) / 32);
    APPLY_GAS_COST(gas, 6 * minimum_word_size);
  }

  ctx.return_data.clear();

  if (endowment != 0 && ToUint256(ctx.host->get_balance(ctx.message->recipient)) < endowment) {
    return {.state = RunState::kErrorCreate};
  }

  auto init_code = ctx.memory.GetSpan(init_code_offset, init_code_size);

  evmc_message msg{
      .kind = (Op == op::CREATE) ? EVMC_CREATE : EVMC_CREATE2,
      .depth = ctx.message->depth + 1,
      .gas = gas - gas / 64,
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

  APPLY_GAS_COST(gas, msg.gas - result.gas_left);

  ctx.gas_refunds += result.gas_refund;

  if (result.status_code == EVMC_SUCCESS) {
    stack_top[0] = ToUint256(result.create_address);
  } else {
    stack_top[0] = 0;
  }

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top,
  };
}

template <op::OpCodes Op>
static OpResult call_impl(uint32_t pc, int64_t gas, const uint256_t* stack_base, uint256_t* stack_top,
                          Context& ctx) noexcept {
  static_assert(Op == op::CALL || Op == op::CALLCODE || Op == op::DELEGATECALL || Op == op::STATICCALL);

  if (ctx.message->depth >= 1024) [[unlikely]] {
    return {.state = RunState::kErrorCall};
  }

  CHECK_STACK_AVAILABLE(stack_base - stack_top, (Op == op::STATICCALL || Op == op::DELEGATECALL) ? 6 : 7);

  // TODO Refactor using dedicated stack pop.
  ctx.stack.top_ = stack_top;

  const uint256_t call_gas_u256 = ctx.stack.Pop();
  const auto account = ToEvmcAddress(ctx.stack.Pop());
  const auto value = (Op == op::STATICCALL || Op == op::DELEGATECALL) ? 0 : ctx.stack.Pop();
  const bool has_value = value != 0;
  const uint256_t input_offset_u256 = ctx.stack.Pop();
  const uint256_t input_size_u256 = ctx.stack.Pop();
  const uint256_t output_offset_u256 = ctx.stack.Pop();
  const uint256_t output_size_u256 = ctx.stack.Pop();

  stack_top = ctx.stack.top_;

  const auto [input_mem_cost, input_offset, input_size] = ctx.MemoryExpansionCost(input_offset_u256, input_size_u256);
  const auto [output_mem_cost, output_offset, output_size] =
      ctx.MemoryExpansionCost(output_offset_u256, output_size_u256);

  APPLY_GAS_COST(gas, std::max(input_mem_cost, output_mem_cost));

  if constexpr (Op == op::CALL) {
    if (has_value) {
      CHECK_STATIC_CALL_CONFORMANCE(ctx);
    }
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

    APPLY_GAS_COST(gas, address_access_cost + positive_value_cost + value_to_empty_account_cost);
  }

  ctx.return_data.clear();

  auto input_data = ctx.memory.GetSpan(input_offset, input_size);

  int64_t call_gas = kMaxGas;
  if (call_gas_u256 < kMaxGas) {
    call_gas = static_cast<int64_t>(call_gas_u256);
  }

  evmc_message msg{
      .kind = (Op == op::DELEGATECALL) ? EVMC_DELEGATECALL
              : (Op == op::CALLCODE)   ? EVMC_CALLCODE
                                       : EVMC_CALL,
      .flags = (Op == op::STATICCALL) ? uint32_t{EVMC_STATIC} : ctx.message->flags,
      .depth = ctx.message->depth + 1,
      .gas = std::min(call_gas, gas - gas / 64),
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
    gas += 2300;
  }

  if (has_value && ToUint256(ctx.host->get_balance(ctx.message->recipient)) < value) {
    stack_top[-1] = 0;
    return {
        .pc = pc + 1,
        .gas_left = gas,
        .stack_top = stack_top - 1,
    };
  }

  const evmc::Result result = ctx.host->call(msg);
  ctx.return_data.assign(result.output_data, result.output_data + result.output_size);

  ctx.memory.Grow(output_offset, output_size);
  if (ctx.return_data.size() > 0) {
    ctx.memory.ReadFromWithSize(ctx.return_data, output_offset, output_size);
  }

  APPLY_GAS_COST(gas, msg.gas - result.gas_left);

  ctx.gas_refunds += result.gas_refund;

  stack_top[-1] = result.status_code == EVMC_SUCCESS;

  return {
      .pc = pc + 1,
      .gas_left = gas,
      .stack_top = stack_top - 1,
  };
}

}  // namespace op

///////////////////////////////////////////////////////////

namespace internal {

bool Context::CheckJumpDest(uint256_t index_u256) noexcept {
  if (index_u256 >= valid_jump_targets.size()) [[unlikely]] {
    return false;
  }

  const uint64_t index = static_cast<uint64_t>(index_u256);

  if (!valid_jump_targets[index]) [[unlikely]] {
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

std::vector<uint8_t> PadCode(std::span<const uint8_t> code) {
  std::vector<uint8_t> padded;
  padded.reserve(code.size() + kStopBytePadding);
  padded.assign(code.begin(), code.end());
  padded.resize(code.size() + kStopBytePadding, op::STOP);
  return padded;
}

template <bool LoggingEnabled, bool ProfilingEnabled>
void RunInterpreter(Context& ctx, Profiler<ProfilingEnabled>&) {
  RunState state = RunState::kRunning;
  uint32_t pc = 0;
  int64_t gas = ctx.gas;
  uint256_t* base = ctx.stack.end_;
  uint256_t* top = ctx.stack.top_;

  while (state == RunState::kRunning) {
    switch (ctx.padded_code[pc]) {
#define OPCODE(opcode, impl)    \
  case op::opcode: {            \
    op::OpResult result = impl; \
    state = result.state;       \
    pc = result.pc;             \
    gas = result.gas_left;      \
    top = result.stack_top;     \
    break;                      \
  }

      OPCODE(STOP, op::stop(pc, gas, top));

      OPCODE(ADD, op::add(pc, gas, base, top));
      OPCODE(MUL, op::mul(pc, gas, base, top));
      OPCODE(SUB, op::sub(pc, gas, base, top));
      OPCODE(DIV, op::div(pc, gas, base, top));
      OPCODE(SDIV, op::sdiv(pc, gas, base, top));
      OPCODE(MOD, op::mod(pc, gas, base, top));
      OPCODE(SMOD, op::smod(pc, gas, base, top));
      OPCODE(ADDMOD, op::addmod(pc, gas, base, top));
      OPCODE(MULMOD, op::mulmod(pc, gas, base, top));
      OPCODE(EXP, op::exp(pc, gas, base, top));
      OPCODE(SIGNEXTEND, op::signextend(pc, gas, base, top));
      OPCODE(LT, op::lt(pc, gas, base, top));
      OPCODE(GT, op::gt(pc, gas, base, top));
      OPCODE(SLT, op::slt(pc, gas, base, top));
      OPCODE(SGT, op::sgt(pc, gas, base, top));
      OPCODE(EQ, op::eq(pc, gas, base, top));
      OPCODE(ISZERO, op::iszero(pc, gas, base, top));
      OPCODE(AND, op::bit_and(pc, gas, base, top));
      OPCODE(OR, op::bit_or(pc, gas, base, top));
      OPCODE(XOR, op::bit_xor(pc, gas, base, top));
      OPCODE(NOT, op::bit_not(pc, gas, base, top));
      OPCODE(BYTE, op::byte(pc, gas, base, top));
      OPCODE(SHL, op::shl(pc, gas, base, top));
      OPCODE(SHR, op::shr(pc, gas, base, top));
      OPCODE(SAR, op::sar(pc, gas, base, top));
      OPCODE(SHA3, op::sha3(pc, gas, base, top, ctx));

      OPCODE(ADDRESS, op::address(pc, gas, base, top, ctx));
      OPCODE(BALANCE, op::balance(pc, gas, base, top, ctx));
      OPCODE(ORIGIN, op::origin(pc, gas, base, top, ctx));
      OPCODE(CALLER, op::caller(pc, gas, base, top, ctx));
      OPCODE(CALLVALUE, op::callvalue(pc, gas, base, top, ctx));
      OPCODE(CALLDATALOAD, op::calldataload(pc, gas, base, top, ctx));
      OPCODE(CALLDATASIZE, op::calldatasize(pc, gas, base, top, ctx));
      OPCODE(CALLDATACOPY, op::calldatacopy(pc, gas, base, top, ctx));
      OPCODE(CODESIZE, op::codesize(pc, gas, base, top, ctx));
      OPCODE(CODECOPY, op::codecopy(pc, gas, base, top, ctx));
      OPCODE(GASPRICE, op::gasprice(pc, gas, base, top, ctx));
      OPCODE(EXTCODESIZE, op::extcodesize(pc, gas, base, top, ctx));
      OPCODE(EXTCODECOPY, op::extcodecopy(pc, gas, base, top, ctx));
      OPCODE(RETURNDATASIZE, op::returndatasize(pc, gas, base, top, ctx));
      OPCODE(RETURNDATACOPY, op::returndatacopy(pc, gas, base, top, ctx));
      OPCODE(EXTCODEHASH, op::extcodehash(pc, gas, base, top, ctx));
      OPCODE(BLOCKHASH, op::blockhash(pc, gas, base, top, ctx));
      OPCODE(COINBASE, op::coinbase(pc, gas, base, top, ctx));
      OPCODE(TIMESTAMP, op::timestamp(pc, gas, base, top, ctx));
      OPCODE(NUMBER, op::blocknumber(pc, gas, base, top, ctx));
      OPCODE(DIFFICULTY, op::prevrandao(pc, gas, base, top, ctx));  // intentional
      OPCODE(GASLIMIT, op::gaslimit(pc, gas, base, top, ctx));
      OPCODE(CHAINID, op::chainid(pc, gas, base, top, ctx));
      OPCODE(SELFBALANCE, op::selfbalance(pc, gas, base, top, ctx));
      OPCODE(BASEFEE, op::basefee(pc, gas, base, top, ctx));

      OPCODE(POP, op::pop(pc, gas, base, top));

      OPCODE(MLOAD, op::mload(pc, gas, base, top, ctx));
      OPCODE(MSTORE, op::mstore(pc, gas, base, top, ctx));
      OPCODE(MSTORE8, op::mstore8(pc, gas, base, top, ctx));

      OPCODE(SLOAD, op::sload(pc, gas, base, top, ctx));
      OPCODE(SSTORE, op::sstore(pc, gas, base, top, ctx));

      OPCODE(JUMP, op::jump(pc, gas, base, top, ctx));
      OPCODE(JUMPI, op::jumpi(pc, gas, base, top, ctx));

      OPCODE(PC, op::pc(pc, gas, base, top));
      OPCODE(MSIZE, op::msize(pc, gas, base, top, ctx));
      OPCODE(GAS, op::gas(pc, gas, base, top));

      OPCODE(JUMPDEST, op::jumpdest(pc, gas, base, top));

      OPCODE(PUSH1, op::push<1>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH2, op::push<2>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH3, op::push<3>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH4, op::push<4>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH5, op::push<5>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH6, op::push<6>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH7, op::push<7>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH8, op::push<8>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH9, op::push<9>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH10, op::push<10>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH11, op::push<11>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH12, op::push<12>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH13, op::push<13>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH14, op::push<14>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH15, op::push<15>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH16, op::push<16>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH17, op::push<17>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH18, op::push<18>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH19, op::push<19>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH20, op::push<20>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH21, op::push<21>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH22, op::push<22>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH23, op::push<23>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH24, op::push<24>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH25, op::push<25>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH26, op::push<26>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH27, op::push<27>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH28, op::push<28>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH29, op::push<29>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH30, op::push<30>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH31, op::push<31>(pc, gas, base, top, ctx.padded_code.data()));
      OPCODE(PUSH32, op::push<32>(pc, gas, base, top, ctx.padded_code.data()));

      OPCODE(DUP1, op::dup<1>(pc, gas, base, top));
      OPCODE(DUP2, op::dup<2>(pc, gas, base, top));
      OPCODE(DUP3, op::dup<3>(pc, gas, base, top));
      OPCODE(DUP4, op::dup<4>(pc, gas, base, top));
      OPCODE(DUP5, op::dup<5>(pc, gas, base, top));
      OPCODE(DUP6, op::dup<6>(pc, gas, base, top));
      OPCODE(DUP7, op::dup<7>(pc, gas, base, top));
      OPCODE(DUP8, op::dup<8>(pc, gas, base, top));
      OPCODE(DUP9, op::dup<9>(pc, gas, base, top));
      OPCODE(DUP10, op::dup<10>(pc, gas, base, top));
      OPCODE(DUP11, op::dup<11>(pc, gas, base, top));
      OPCODE(DUP12, op::dup<12>(pc, gas, base, top));
      OPCODE(DUP13, op::dup<13>(pc, gas, base, top));
      OPCODE(DUP14, op::dup<14>(pc, gas, base, top));
      OPCODE(DUP15, op::dup<15>(pc, gas, base, top));
      OPCODE(DUP16, op::dup<16>(pc, gas, base, top));

      OPCODE(SWAP1, op::swap<1>(pc, gas, base, top));
      OPCODE(SWAP2, op::swap<2>(pc, gas, base, top));
      OPCODE(SWAP3, op::swap<3>(pc, gas, base, top));
      OPCODE(SWAP4, op::swap<4>(pc, gas, base, top));
      OPCODE(SWAP5, op::swap<5>(pc, gas, base, top));
      OPCODE(SWAP6, op::swap<6>(pc, gas, base, top));
      OPCODE(SWAP7, op::swap<7>(pc, gas, base, top));
      OPCODE(SWAP8, op::swap<8>(pc, gas, base, top));
      OPCODE(SWAP9, op::swap<9>(pc, gas, base, top));
      OPCODE(SWAP10, op::swap<10>(pc, gas, base, top));
      OPCODE(SWAP11, op::swap<11>(pc, gas, base, top));
      OPCODE(SWAP12, op::swap<12>(pc, gas, base, top));
      OPCODE(SWAP13, op::swap<13>(pc, gas, base, top));
      OPCODE(SWAP14, op::swap<14>(pc, gas, base, top));
      OPCODE(SWAP15, op::swap<15>(pc, gas, base, top));
      OPCODE(SWAP16, op::swap<16>(pc, gas, base, top));

      OPCODE(LOG0, op::log<0>(pc, gas, base, top, ctx));
      OPCODE(LOG1, op::log<1>(pc, gas, base, top, ctx));
      OPCODE(LOG2, op::log<2>(pc, gas, base, top, ctx));
      OPCODE(LOG3, op::log<3>(pc, gas, base, top, ctx));
      OPCODE(LOG4, op::log<4>(pc, gas, base, top, ctx));

      OPCODE(CREATE, op::create_impl<op::CREATE>(pc, gas, base, top, ctx));
      OPCODE(CREATE2, op::create_impl<op::CREATE2>(pc, gas, base, top, ctx));

      OPCODE(RETURN, op::return_op<RunState::kReturn>(pc, gas, base, top, ctx));
      OPCODE(REVERT, op::return_op<RunState::kRevert>(pc, gas, base, top, ctx));

      OPCODE(CALL, op::call_impl<op::CALL>(pc, gas, base, top, ctx));
      OPCODE(CALLCODE, op::call_impl<op::CALLCODE>(pc, gas, base, top, ctx));
      OPCODE(DELEGATECALL, op::call_impl<op::DELEGATECALL>(pc, gas, base, top, ctx));
      OPCODE(STATICCALL, op::call_impl<op::STATICCALL>(pc, gas, base, top, ctx));

      OPCODE(INVALID, op::invalid(pc, gas, base, top));
      OPCODE(SELFDESTRUCT, op::selfdestruct(pc, gas, base, top, ctx));

#undef PROFILE_START
#undef PROFILE_END
#undef OPCODE

      default:
        state = RunState::kErrorOpcode;
    }
  }

  if (!IsSuccess(state)) {
    gas = 0;
  }

  // Keep return data only when we are supposed to return something.
  if (state != RunState::kReturn && state != RunState::kRevert) {
    ctx.return_data.clear();
  }

  ctx.state = state;
  ctx.stack.top_ = top;
  ctx.gas = gas;
}

template void RunInterpreter<false, false>(Context&, Profiler<false>&);
template void RunInterpreter<true, false>(Context&, Profiler<false>&);
template void RunInterpreter<false, true>(Context&, Profiler<true>&);
template void RunInterpreter<true, true>(Context&, Profiler<true>&);

}  // namespace internal

}  // namespace tosca::evmzero

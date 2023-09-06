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

struct OpInfo {
  int32_t pops = 0;
  int32_t pushes = 0;
  int32_t static_gas = 0;
  int32_t instruction_length = 1;
  bool is_jump = false;
  bool disallowed_in_static_call = false;

  std::optional<evmc_revision> introduced_in;

  constexpr int32_t GetStackDelta() const { return pushes - pops; }
};

constexpr OpInfo NullaryOp(int32_t static_gas) { return OpInfo{.pops = 0, .pushes = 1, .static_gas = static_gas}; }
constexpr OpInfo UnaryOp(int32_t static_gas) { return OpInfo{.pops = 1, .pushes = 1, .static_gas = static_gas}; }
constexpr OpInfo BinaryOp(int32_t static_gas) { return OpInfo{.pops = 2, .pushes = 1, .static_gas = static_gas}; }

struct OpResult {
  RunState state = RunState::kRunning;
  int64_t dynamic_gas_costs = 0;
};

template <OpCode op_code>
struct Impl {
  constexpr static OpInfo kInfo{};
  static OpResult Run() noexcept {
    static_assert(op_code != op_code, "Not implemented!");
    return {};
  }
};

template <>
struct Impl<OpCode::STOP> {
  constexpr static OpInfo kInfo{};
  static OpResult Run() noexcept { return {.state = RunState::kDone}; }
};

template <>
struct Impl<OpCode::ADD> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] += top[0];
    return {};
  }
};

template <>
struct Impl<OpCode::MUL> {
  constexpr static OpInfo kInfo = BinaryOp(5);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] *= top[0];
    return {};
  }
};

template <>
struct Impl<OpCode::SUB> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = top[0] - top[1];
    return {};
  }
};

template <>
struct Impl<OpCode::DIV> {
  constexpr static OpInfo kInfo = BinaryOp(5);

  static OpResult Run(uint256_t* top) noexcept {
    if (top[1] != 0) {
      top[1] = top[0] / top[1];
    }
    return {};
  }
};

template <>
struct Impl<OpCode::SDIV> {
  constexpr static OpInfo kInfo = BinaryOp(5);

  static OpResult Run(uint256_t* top) noexcept {
    if (top[1] != 0) {
      top[1] = intx::sdivrem(top[0], top[1]).quot;
    }
    return {};
  }
};

template <>
struct Impl<OpCode::MOD> {
  constexpr static OpInfo kInfo = BinaryOp(5);

  static OpResult Run(uint256_t* top) noexcept {
    if (top[1] != 0) {
      top[1] = top[0] % top[1];
    }
    return {};
  }
};

template <>
struct Impl<OpCode::SMOD> {
  constexpr static OpInfo kInfo = BinaryOp(5);

  static OpResult Run(uint256_t* top) noexcept {
    if (top[1] != 0) {
      top[1] = intx::sdivrem(top[0], top[1]).rem;
    }
    return {};
  }
};

template <>
struct Impl<OpCode::ADDMOD> {
  constexpr static OpInfo kInfo{
      .pops = 3,
      .pushes = 1,
      .static_gas = 8,
  };

  static OpResult Run(uint256_t* top) noexcept {
    if (top[2] != 0) {
      top[2] = intx::addmod(top[0], top[1], top[2]);
    }
    return {};
  }
};

template <>
struct Impl<OpCode::MULMOD> {
  constexpr static OpInfo kInfo{
      .pops = 3,
      .pushes = 1,
      .static_gas = 8,
  };

  static OpResult Run(uint256_t* top) noexcept {
    if (top[2] != 0) {
      top[2] = intx::mulmod(top[0], top[1], top[2]);
    }
    return {};
  }
};

template <>
struct Impl<OpCode::EXP> {
  constexpr static OpInfo kInfo = BinaryOp(10);

  static OpResult Run(uint256_t* top, int64_t gas) noexcept {
    uint256_t& a = top[0];
    uint256_t& exponent = top[1];
    int64_t dynamic_gas = 50 * intx::count_significant_bytes(exponent);
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    top[1] = intx::exp(a, exponent);
    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::SIGNEXTEND> {
  constexpr static OpInfo kInfo = BinaryOp(5);

  static OpResult Run(uint256_t* top) noexcept {
    uint8_t leading_byte_index = static_cast<uint8_t>(top[0]);
    if (leading_byte_index > 31) {
      leading_byte_index = 31;
    }

    uint256_t value = top[1];

    bool is_negative = ToByteArrayLe(value)[leading_byte_index] & 0b1000'0000;
    if (is_negative) {
      auto mask = kUint256Max << (8 * (leading_byte_index + 1));
      top[1] = mask | value;
    } else {
      auto mask = kUint256Max >> (8 * (31 - leading_byte_index));
      top[1] = mask & value;
    }

    return {};
  }
};

template <>
struct Impl<OpCode::LT> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = top[0] < top[1] ? 1 : 0;
    return {};
  }
};

template <>
struct Impl<OpCode::GT> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = top[0] > top[1] ? 1 : 0;
    return {};
  }
};

template <>
struct Impl<OpCode::SLT> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = intx::slt(top[0], top[1]) ? 1 : 0;
    return {};
  }
};

template <>
struct Impl<OpCode::SGT> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = intx::slt(top[1], top[0]) ? 1 : 0;
    return {};
  }
};

template <>
struct Impl<OpCode::EQ> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = top[0] == top[1] ? 1 : 0;
    return {};
  }
};

template <>
struct Impl<OpCode::ISZERO> {
  constexpr static OpInfo kInfo = UnaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[0] = top[0] == 0;
    return {};
  }
};

template <>
struct Impl<OpCode::AND> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = top[0] & top[1];
    return {};
  }
};

template <>
struct Impl<OpCode::OR> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = top[0] | top[1];
    return {};
  }
};

template <>
struct Impl<OpCode::XOR> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] = top[0] ^ top[1];
    return {};
  }
};

template <>
struct Impl<OpCode::NOT> {
  constexpr static OpInfo kInfo = UnaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[0] = ~top[0];
    return {};
  }
};

template <>
struct Impl<OpCode::BYTE> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    if (top[0] < 32) {
      top[1] = ToByteArrayLe(top[1])[31 - static_cast<uint8_t>(top[0])];
    } else {
      top[1] = 0;
    }

    return {};
  }
};

template <>
struct Impl<OpCode::SHL> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] <<= top[0];
    return {};
  }
};

template <>
struct Impl<OpCode::SHR> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    top[1] >>= top[0];
    return {};
  }
};

template <>
struct Impl<OpCode::SAR> {
  constexpr static OpInfo kInfo = BinaryOp(3);

  static OpResult Run(uint256_t* top) noexcept {
    const bool is_negative = ToByteArrayLe(top[1])[31] & 0b1000'0000;

    if (top[0] <= 255) {
      top[1] >>= top[0];
      if (is_negative) {
        top[1] |= (kUint256Max << (255 - top[0]));
      }
    } else {
      top[1] = 0;
    }

    return {};
  }
};

template <>
struct Impl<OpCode::SHA3> {
  constexpr static OpInfo kInfo = BinaryOp(30);

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t offset_u256 = top[0];
    const uint256_t size_u256 = top[1];

    const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
    const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
    int64_t dynamic_gas = mem_cost + 6 * minimum_word_size;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    auto memory_span = ctx.memory.GetSpan(offset, size);
    if (ctx.sha3_cache) {
      top[1] = ctx.sha3_cache->Hash(memory_span);
    } else {
      top[1] = ToUint256(ethash::keccak256(memory_span.data(), memory_span.size()));
    }

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::ADDRESS> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.message->recipient);
    return {};
  }
};

template <>
struct Impl<OpCode::BALANCE> {
  constexpr static OpInfo kInfo = UnaryOp(0);

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    evmc::address address = ToEvmcAddress(top[0]);

    int64_t dynamic_gas = 700;
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
        dynamic_gas = 100;
      } else {
        dynamic_gas = 2600;
      }
    }
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    top[0] = ToUint256(ctx.host->get_balance(address));

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::ORIGIN> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.host->get_tx_context().tx_origin);
    return {};
  }
};

template <>
struct Impl<OpCode::CALLER> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.message->sender);
    return {};
  }
};

template <>
struct Impl<OpCode::CALLVALUE> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.message->value);
    return {};
  }
};

template <>
struct Impl<OpCode::CALLDATALOAD> {
  constexpr static OpInfo kInfo = UnaryOp(3);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    const uint256_t offset_u256 = top[0];

    std::span<const uint8_t> input_view;
    if (offset_u256 < ctx.message->input_size) {
      input_view = std::span(ctx.message->input_data, ctx.message->input_size)  //
                       .subspan(static_cast<uint64_t>(offset_u256));
    }

    evmc::bytes32 value{};
    std::copy_n(input_view.begin(), std::min<size_t>(input_view.size(), 32), value.bytes);

    top[0] = ToUint256(value);

    return {};
  }
};

template <>
struct Impl<OpCode::CALLDATASIZE> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ctx.message->input_size;
    return {};
  }
};

template <>
struct Impl<OpCode::CALLDATACOPY> {
  constexpr static OpInfo kInfo{
      .pops = 3,
      .pushes = 0,
      .static_gas = 3,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t memory_offset_u256 = top[0];
    const uint256_t data_offset_u256 = top[1];
    const uint256_t size_u256 = top[2];

    const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
    const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
    int64_t dynamic_gas = mem_cost + 3 * minimum_word_size;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    std::span<const uint8_t> data_view;
    if (data_offset_u256 < ctx.message->input_size) {
      data_view = std::span(ctx.message->input_data, ctx.message->input_size)  //
                      .subspan(static_cast<uint64_t>(data_offset_u256));
    }

    ctx.memory.ReadFromWithSize(data_view, memory_offset, size);

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::CODESIZE> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ctx.padded_code.size() - kStopBytePadding;
    return {};
  }
};

template <>
struct Impl<OpCode::CODECOPY> {
  constexpr static OpInfo kInfo{
      .pops = 3,
      .pushes = 0,
      .static_gas = 3,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t memory_offset_u256 = top[0];
    const uint256_t code_offset_u256 = top[1];
    const uint256_t size_u256 = top[2];

    const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
    const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
    int64_t dynamic_gas = mem_cost + 3 * minimum_word_size;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    std::span<const uint8_t> code_view;
    if (code_offset_u256 < ctx.padded_code.size() - kStopBytePadding) {
      code_view = std::span(ctx.padded_code).subspan(static_cast<uint64_t>(code_offset_u256));
    }

    ctx.memory.ReadFromWithSize(code_view, memory_offset, size);

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::GASPRICE> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.host->get_tx_context().tx_gas_price);
    return {};
  }
};

template <>
struct Impl<OpCode::EXTCODESIZE> {
  constexpr static OpInfo kInfo = UnaryOp(0);

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    auto address = ToEvmcAddress(top[0]);

    int64_t dynamic_gas = 700;
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
        dynamic_gas = 100;
      } else {
        dynamic_gas = 2600;
      }
    }
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    top[0] = ctx.host->get_code_size(address);

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::EXTCODECOPY> {
  constexpr static OpInfo kInfo{
      .pops = 4,
      .pushes = 0,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const auto address = ToEvmcAddress(top[0]);
    const uint256_t memory_offset_u256 = top[1];
    const uint256_t code_offset_u256 = top[2];
    const uint256_t size_u256 = top[3];

    const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
    int64_t dynamic_gas = mem_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
    int64_t address_access_cost = 700;
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
        address_access_cost = 100;
      } else {
        address_access_cost = 2600;
      }
    }
    dynamic_gas += 3 * minimum_word_size + address_access_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    auto memory_span = ctx.memory.GetSpan(memory_offset, size);
    if (code_offset_u256 <= std::numeric_limits<uint64_t>::max()) {
      uint64_t code_offset = static_cast<uint64_t>(code_offset_u256);
      size_t bytes_written = ctx.host->copy_code(address, code_offset, memory_span.data(), memory_span.size());
      memory_span = memory_span.subspan(bytes_written);
    }
    std::fill(memory_span.begin(), memory_span.end(), 0);

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::RETURNDATASIZE> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ctx.return_data.size();
    return {};
  }
};

template <>
struct Impl<OpCode::RETURNDATACOPY> {
  constexpr static OpInfo kInfo{
      .pops = 3,
      .pushes = 0,
      .static_gas = 3,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t memory_offset_u256 = top[0];
    const uint256_t return_data_offset_u256 = top[1];
    const uint256_t size_u256 = top[2];

    const auto [mem_cost, memory_offset, size] = ctx.MemoryExpansionCost(memory_offset_u256, size_u256);
    int64_t dynamic_gas = mem_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    const int64_t minimum_word_size = static_cast<int64_t>((size + 31) / 32);
    dynamic_gas += 3 * minimum_word_size;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    {
      const auto [end_u256, carry] = intx::addc(return_data_offset_u256, size_u256);
      if (carry || end_u256 > ctx.return_data.size()) {
        return {.state = RunState::kErrorReturnDataCopyOutOfBounds};
      }
    }

    std::span<const uint8_t> return_data_view = std::span(ctx.return_data)  //
                                                    .subspan(static_cast<uint64_t>(return_data_offset_u256));
    ctx.memory.ReadFromWithSize(return_data_view, memory_offset, size);

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::EXTCODEHASH> {
  constexpr static OpInfo kInfo = UnaryOp(0);

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    auto address = ToEvmcAddress(top[0]);

    int64_t dynamic_gas = 700;
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_account(address) == EVMC_ACCESS_WARM) {
        dynamic_gas = 100;
      } else {
        dynamic_gas = 2600;
      }
    }
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    top[0] = ToUint256(ctx.host->get_code_hash(address));

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::BLOCKHASH> {
  constexpr static OpInfo kInfo = UnaryOp(20);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    int64_t number = static_cast<int64_t>(top[0]);
    top[0] = ToUint256(ctx.host->get_block_hash(number));
    return {};
  }
};

template <>
struct Impl<OpCode::COINBASE> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.host->get_tx_context().block_coinbase);
    return {};
  }
};

template <>
struct Impl<OpCode::TIMESTAMP> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ctx.host->get_tx_context().block_timestamp;
    return {};
  }
};

template <>
struct Impl<OpCode::NUMBER> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ctx.host->get_tx_context().block_number;
    return {};
  }
};

template <>
struct Impl<OpCode::DIFFICULTY> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.host->get_tx_context().block_prev_randao);
    return {};
  }
};

template <>
struct Impl<OpCode::GASLIMIT> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ctx.host->get_tx_context().block_gas_limit;
    return {};
  }
};

template <>
struct Impl<OpCode::CHAINID> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.host->get_tx_context().chain_id);
    return {};
  }
};

template <>
struct Impl<OpCode::SELFBALANCE> {
  constexpr static OpInfo kInfo = NullaryOp(5);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.host->get_balance(ctx.message->recipient));
    return {};
  }
};

template <>
struct Impl<OpCode::BASEFEE> {
  constexpr static OpInfo kInfo{
      .pops = 0,
      .pushes = 1,
      .static_gas = 2,
      .introduced_in = EVMC_LONDON,
  };

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ToUint256(ctx.host->get_tx_context().block_base_fee);
    return {};
  }
};

template <>
struct Impl<OpCode::POP> {
  constexpr static OpInfo kInfo{
      .pops = 1,
      .pushes = 0,
      .static_gas = 2,
  };

  static OpResult Run() noexcept { return {}; }
};

template <>
struct Impl<OpCode::MLOAD> {
  constexpr static OpInfo kInfo = UnaryOp(3);

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t offset_u256 = top[0];

    const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 32);
    int64_t dynamic_gas = mem_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    uint256_t value;
    ctx.memory.WriteTo({ToBytes(value), 32}, offset);

    if constexpr (std::endian::native == std::endian::little) {
      value = intx::bswap(value);
    }

    top[0] = value;

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::MSTORE> {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 0,
      .static_gas = 3,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t offset_u256 = top[0];
    uint256_t value = top[1];

    const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 32);
    int64_t dynamic_gas = mem_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    if constexpr (std::endian::native == std::endian::little) {
      value = intx::bswap(value);
    }

    ctx.memory.ReadFrom({ToBytes(value), 32}, offset);

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::MSTORE8> {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 0,
      .static_gas = 3,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t offset_u256 = top[0];
    const uint8_t value = static_cast<uint8_t>(top[1]);

    const auto [mem_cost, offset, _] = ctx.MemoryExpansionCost(offset_u256, 1);
    int64_t dynamic_gas = mem_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    ctx.memory.ReadFrom({&value, 1}, offset);

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::SLOAD> {
  constexpr static OpInfo kInfo = UnaryOp(0);

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t key = top[0];

    int64_t dynamic_gas = 800;
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_storage(ctx.message->recipient, ToEvmcBytes(key)) == EVMC_ACCESS_WARM) {
        dynamic_gas = 100;
      } else {
        dynamic_gas = 2100;
      }
    }
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    top[0] = ToUint256(ctx.host->get_storage(ctx.message->recipient, ToEvmcBytes(key)));

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::SSTORE> {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 0,
      .disallowed_in_static_call = true,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    // EIP-2200
    if (gas <= 2300) [[unlikely]] {
      return {.state = RunState::kErrorGas};
    }

    const uint256_t key = top[0];
    const uint256_t value = top[1];

    bool key_is_warm = false;
    if (ctx.revision >= EVMC_BERLIN) {
      key_is_warm = ctx.host->access_storage(ctx.message->recipient, ToEvmcBytes(key)) == EVMC_ACCESS_WARM;
    }

    int64_t dynamic_gas = 800;
    if (ctx.revision >= EVMC_BERLIN) {
      dynamic_gas = 100;
    }

    const auto storage_status = ctx.host->set_storage(ctx.message->recipient, ToEvmcBytes(key), ToEvmcBytes(value));

    // Dynamic gas cost depends on the current value in storage. set_storage
    // provides the relevant information we need.
    if (storage_status == EVMC_STORAGE_ADDED) {
      dynamic_gas = 20000;
    }
    if (storage_status == EVMC_STORAGE_MODIFIED || storage_status == EVMC_STORAGE_DELETED) {
      if (ctx.revision >= EVMC_BERLIN) {
        dynamic_gas = 2900;
      } else {
        dynamic_gas = 5000;
      }
    }

    if (ctx.revision >= EVMC_BERLIN && !key_is_warm) {
      dynamic_gas += 2100;
    }

    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

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

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <>
struct Impl<OpCode::JUMP> {
  constexpr static OpInfo kInfo{
      .pops = 1,
      .pushes = 0,
      .static_gas = 8,
      .is_jump = true,
  };

  static bool RunJump(uint256_t*) noexcept { return true; }
};

template <>
struct Impl<OpCode::JUMPI> {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 0,
      .static_gas = 10,
      .is_jump = true,
  };

  static bool RunJump(uint256_t* top) noexcept {
    const uint256_t& b = top[1];
    return b != 0;
  }
};

template <>
struct Impl<OpCode::PC> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, const uint8_t* pc, Context& ctx) noexcept {
    top[-1] = pc - ctx.padded_code.data();
    return {};
  }
};

template <>
struct Impl<OpCode::MSIZE> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, Context& ctx) noexcept {
    top[-1] = ctx.memory.GetSize();
    return {};
  }
};

template <>
struct Impl<OpCode::GAS> {
  constexpr static OpInfo kInfo = NullaryOp(2);

  static OpResult Run(uint256_t* top, int64_t gas) noexcept {
    top[-1] = gas;
    return {};
  }
};

template <>
struct Impl<OpCode::JUMPDEST> {
  constexpr static OpInfo kInfo{
      .pops = 0,
      .pushes = 0,
      .static_gas = 1,
  };

  static OpResult Run() noexcept { return {}; }
};

template <uint64_t N>
struct PushImpl {
  constexpr static OpInfo kInfo{
      .pops = 0,
      .pushes = 1,
      .static_gas = 3,
      .instruction_length = 1 + N,
  };

  static OpResult Run(uint256_t* top, const uint8_t* pc) noexcept {
    constexpr auto num_full_words = N / sizeof(uint64_t);
    constexpr auto num_partial_bytes = N % sizeof(uint64_t);

    const uint8_t* data = pc + 1;

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

    return {};
  }
};

template <op::OpCode op_code>
requires(OpCode::PUSH1 <= op_code && op_code <= OpCode::PUSH32)  //
    struct Impl<op_code> : PushImpl<static_cast<uint64_t>(op_code - OpCode::PUSH1 + 1)> {
};

template <uint64_t N>
struct DupImpl {
  constexpr static OpInfo kInfo{
      .pops = N,
      .pushes = N + 1,
      .static_gas = 3,
  };

  static OpResult Run(uint256_t* top) noexcept {
    *(top - 1) = top[N - 1];
    return {};
  }
};

template <op::OpCode op_code>
requires(OpCode::DUP1 <= op_code && op_code <= OpCode::DUP16)  //
    struct Impl<op_code> : DupImpl<static_cast<uint64_t>(op_code - OpCode::DUP1 + 1)> {
};

template <uint64_t N>
struct SwapImpl {
  constexpr static OpInfo kInfo{
      .pops = N + 1,
      .pushes = N + 1,
      .static_gas = 3,
  };

  static OpResult Run(uint256_t* top) noexcept {
    std::swap(top[0], top[N]);
    return {};
  }
};

template <op::OpCode op_code>
requires(OpCode::SWAP1 <= op_code && op_code <= OpCode::SWAP16)  //
    struct Impl<op_code> : SwapImpl<static_cast<uint64_t>(op_code - OpCode::SWAP1 + 1)> {
};

template <uint64_t N>
struct LogImpl {
  constexpr static OpInfo kInfo{
      .pops = N + 2,
      .pushes = 0,
      .static_gas = 375 + 375 * N,
      .disallowed_in_static_call = true,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t offset_u256 = top[0];
    const uint256_t size_u256 = top[1];

    const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
    int64_t dynamic_gas = mem_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    std::array<evmc::bytes32, N> topics;
    for (unsigned i = 0; i < N; ++i) {
      topics[i] = ToEvmcBytes(top[2 + i]);
    }

    dynamic_gas += static_cast<int64_t>(8 * size);
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    auto data = ctx.memory.GetSpan(offset, size);

    ctx.host->emit_log(ctx.message->recipient, data.data(), data.size(), topics.data(), topics.size());

    return {.dynamic_gas_costs = dynamic_gas};
  }
};

template <op::OpCode op_code>
requires(OpCode::LOG0 <= op_code && op_code <= OpCode::LOG4)  //
    struct Impl<op_code> : LogImpl<static_cast<uint64_t>(op_code - OpCode::LOG0)> {
};

template <RunState result_state>
struct ReturnImpl {
  constexpr static OpInfo kInfo{
      .pops = 2,
      .pushes = 0,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    const uint256_t offset_u256 = top[0];
    const uint256_t size_u256 = top[1];

    const auto [mem_cost, offset, size] = ctx.MemoryExpansionCost(offset_u256, size_u256);
    int64_t dynamic_gas = mem_cost;
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    ctx.return_data.resize(size);
    ctx.memory.WriteTo(ctx.return_data, offset);

    return {
        .state = result_state,
        .dynamic_gas_costs = dynamic_gas,
    };
  }
};

template <>
struct Impl<OpCode::RETURN> : ReturnImpl<RunState::kReturn> {};
template <>
struct Impl<OpCode::REVERT> : ReturnImpl<RunState::kRevert> {};

template <>
struct Impl<OpCode::INVALID> {
  constexpr static OpInfo kInfo{};

  static OpResult Run() noexcept { return {.state = RunState::kInvalid}; }
};

template <>
struct Impl<OpCode::SELFDESTRUCT> {
  constexpr static OpInfo kInfo{
      .pops = 1,
      .pushes = 0,
      .static_gas = 5000,
      .disallowed_in_static_call = true,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    auto account = ToEvmcAddress(top[0]);

    int64_t dynamic_gas = 0;
    if (ctx.host->get_balance(ctx.message->recipient) && !ctx.host->account_exists(account)) {
      dynamic_gas += 25000;
    }
    if (ctx.revision >= EVMC_BERLIN) {
      if (ctx.host->access_account(account) == EVMC_ACCESS_COLD) {
        dynamic_gas += 2600;
      }
    }
    if (gas < dynamic_gas) [[unlikely]]
      return {.dynamic_gas_costs = dynamic_gas};

    if (ctx.host->selfdestruct(ctx.message->recipient, account)) {
      if (ctx.revision < EVMC_LONDON) {
        ctx.gas_refunds += 24000;
      }
    }

    return {
        .state = RunState::kDone,
        .dynamic_gas_costs = dynamic_gas,
    };
  }
};

template <OpCode Op>
struct CreateImpl {
  constexpr static OpInfo kInfo{
      .pops = Op == op::CREATE ? 3 : 4,
      .pushes = 1,
      .static_gas = 32000,
      .disallowed_in_static_call = true,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    if (ctx.message->depth >= 1024) [[unlikely]]
      return {.state = RunState::kErrorCreate};

    const int64_t initial_gas = gas;

    const auto endowment = top[0];
    const uint256_t init_code_offset_u256 = top[1];
    const uint256_t init_code_size_u256 = top[2];
    const auto salt = (Op == op::CREATE2) ? top[3] : uint256_t{0};

    // Set up stack pointer for result value.
    top += (Op == op::CREATE) ? 2 : 3;

    const auto [mem_cost, init_code_offset, init_code_size] =
        ctx.MemoryExpansionCost(init_code_offset_u256, init_code_size_u256);
    if (gas -= mem_cost; gas < 0) [[unlikely]]
      return {.dynamic_gas_costs = initial_gas - gas};

    if constexpr (Op == op::CREATE2) {
      const int64_t minimum_word_size = static_cast<int64_t>((init_code_size + 31) / 32);
      if (gas -= 6 * minimum_word_size; gas < 0) [[unlikely]]
        return {.dynamic_gas_costs = initial_gas - gas};
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

    gas -= msg.gas - result.gas_left;
    ctx.gas_refunds += result.gas_refund;

    if (result.status_code == EVMC_SUCCESS) {
      top[0] = ToUint256(result.create_address);
    } else {
      top[0] = 0;
    }

    return {.dynamic_gas_costs = initial_gas - gas};
  }
};

template <>
struct Impl<OpCode::CREATE> : CreateImpl<OpCode::CREATE> {};
template <>
struct Impl<OpCode::CREATE2> : CreateImpl<OpCode::CREATE2> {};

template <OpCode Op>
struct CallImpl {
  constexpr static OpInfo kInfo{
      .pops = (Op == op::STATICCALL || Op == op::DELEGATECALL) ? 6 : 7,
      .pushes = 1,
  };

  static OpResult Run(uint256_t* top, int64_t gas, Context& ctx) noexcept {
    if (ctx.message->depth >= 1024) [[unlikely]]
      return {.state = RunState::kErrorCall};

    const int64_t initial_gas = gas;

    uint256_t call_gas_u256;
    evmc::address account;
    uint256_t value;
    uint256_t input_offset_u256;
    uint256_t input_size_u256;
    uint256_t output_offset_u256;
    uint256_t output_size_u256;

    if constexpr (Op == op::STATICCALL || Op == op::DELEGATECALL) {
      call_gas_u256 = top[0];
      account = ToEvmcAddress(top[1]);
      value = 0;
      input_offset_u256 = top[2];
      input_size_u256 = top[3];
      output_offset_u256 = top[4];
      output_size_u256 = top[5];

      // Set up stack pointer for result value.
      top += 5;

    } else {
      call_gas_u256 = top[0];
      account = ToEvmcAddress(top[1]);
      value = top[2];
      input_offset_u256 = top[3];
      input_size_u256 = top[4];
      output_offset_u256 = top[5];
      output_size_u256 = top[6];

      // Set up stack pointer for result value.
      top += 6;
    }

    const bool has_value = value != 0;

    const auto [input_mem_cost, input_offset, input_size] = ctx.MemoryExpansionCost(input_offset_u256, input_size_u256);
    const auto [output_mem_cost, output_offset, output_size] =
        ctx.MemoryExpansionCost(output_offset_u256, output_size_u256);

    if (gas -= std::max(input_mem_cost, output_mem_cost); gas < 0) [[unlikely]]
      return {.dynamic_gas_costs = initial_gas - gas};

    if constexpr (Op == op::CALL) {
      if (has_value && ctx.is_static_call) {
        return {.state = RunState::kErrorStaticCall};
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

      if (gas -= address_access_cost + positive_value_cost + value_to_empty_account_cost; gas < 0) [[unlikely]]
        return {.dynamic_gas_costs = initial_gas - gas};
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
      top[0] = 0;
      return {.dynamic_gas_costs = initial_gas - gas};
    }

    const evmc::Result result = ctx.host->call(msg);
    ctx.return_data.assign(result.output_data, result.output_data + result.output_size);

    ctx.memory.Grow(output_offset, output_size);
    if (ctx.return_data.size() > 0) {
      ctx.memory.ReadFromWithSize(ctx.return_data, output_offset, output_size);
    }

    if (gas -= msg.gas - result.gas_left; gas < 0) [[unlikely]]
      return {.dynamic_gas_costs = initial_gas - gas};

    ctx.gas_refunds += result.gas_refund;

    top[0] = result.status_code == EVMC_SUCCESS;

    return {.dynamic_gas_costs = initial_gas - gas};
  }
};

template <>
struct Impl<OpCode::CALL> : CallImpl<OpCode::CALL> {};
template <>
struct Impl<OpCode::CALLCODE> : CallImpl<OpCode::CALLCODE> {};
template <>
struct Impl<OpCode::DELEGATECALL> : CallImpl<OpCode::DELEGATECALL> {};
template <>
struct Impl<OpCode::STATICCALL> : CallImpl<OpCode::STATICCALL> {};

inline OpResult Invoke(uint256_t*, const uint8_t*, int64_t, Context&,  //
                       OpResult (*op)() noexcept                       //
                       ) noexcept {
  return op();
}

inline OpResult Invoke(uint256_t* top, const uint8_t*, int64_t, Context&,  //
                       OpResult (*op)(uint256_t* top) noexcept             //
                       ) noexcept {
  return op(top);
}

inline OpResult Invoke(uint256_t* top, const uint8_t*, int64_t gas, Context&,  //
                       OpResult (*op)(uint256_t* top, int64_t gas) noexcept    //
                       ) noexcept {
  return op(top, gas);
}

inline OpResult Invoke(uint256_t* top, const uint8_t* pc, int64_t, Context&,       //
                       OpResult (*op)(uint256_t* top, const uint8_t* pc) noexcept  //
                       ) noexcept {
  return op(top, pc);
}

inline OpResult Invoke(uint256_t* top, const uint8_t*, int64_t, Context& ctx,  //
                       OpResult (*op)(uint256_t* top, Context&) noexcept       //
                       ) noexcept {
  return op(top, ctx);
}

inline OpResult Invoke(uint256_t* top, const uint8_t* pc, int64_t, Context& ctx,             //
                       OpResult (*op)(uint256_t* top, const uint8_t* pc, Context&) noexcept  //
                       ) noexcept {
  return op(top, pc, ctx);
}

inline OpResult Invoke(uint256_t* top, const uint8_t*, int64_t gas, Context& ctx,      //
                       OpResult (*op)(uint256_t* top, int64_t gas, Context&) noexcept  //
                       ) noexcept {
  return op(top, gas, ctx);
}

}  // namespace op

///////////////////////////////////////////////////////////

namespace internal {

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

std::vector<uint8_t> PadCode(std::span<const uint8_t> code) {
  std::vector<uint8_t> padded;
  padded.reserve(code.size() + kStopBytePadding);
  padded.assign(code.begin(), code.end());
  padded.resize(code.size() + kStopBytePadding, op::STOP);
  return padded;
}

struct Result {
  RunState state = RunState::kDone;
  const uint8_t* pc = nullptr;
  int64_t gas_left = 0;
  uint256_t* top = nullptr;
};

template <op::OpCode op_code>
inline Result Run(const uint8_t* pc, int64_t gas, uint256_t* top, const uint8_t* code, Context& ctx) {
  using Impl = op::Impl<op_code>;

  if constexpr (Impl::kInfo.introduced_in) {
    if (ctx.revision < Impl::kInfo.introduced_in) [[unlikely]] {
      return {.state = RunState::kErrorOpcode};
    }
  }

  if constexpr (Impl::kInfo.disallowed_in_static_call) {
    if (ctx.is_static_call) [[unlikely]]
      return {.state = RunState::kErrorStaticCall};
  }

  // Check stack requirements. Since the stack is aligned to 64k boundaries, we
  // can compute the stack size directly from the stack pointer.
  auto size = Stack::kStackSize - (reinterpret_cast<size_t>(top) & 0xFFFF) / sizeof(*top);

  if constexpr (Impl::kInfo.pops > 0) {
    if (size < Impl::kInfo.pops) [[unlikely]] {
      return Result{.state = RunState::kErrorStackUnderflow};
    }
  }
  if constexpr (Impl::kInfo.GetStackDelta() > 0) {
    if (Stack::kStackSize - size < Impl::kInfo.GetStackDelta()) [[unlikely]] {
      return Result{.state = RunState::kErrorStackOverflow};
    }
  }
  // Charge static gas costs.
  if (gas < Impl::kInfo.static_gas) [[unlikely]] {
    return Result{.state = RunState::kErrorGas};
  }
  gas -= Impl::kInfo.static_gas;

  // Run the operation.
  RunState state = RunState::kRunning;
  if constexpr (Impl::kInfo.is_jump) {
    if (Impl::RunJump(top)) {
      if (!ctx.CheckJumpDest(*top)) [[unlikely]] {
        return Result{.state = ctx.state};
      }
      pc = code + static_cast<uint32_t>(*top);
    } else {
      pc += 1;
    }
  } else {
    auto res = Invoke(top, pc, gas, ctx, Impl::Run);
    state = res.state;
    if (res.dynamic_gas_costs > 0) {
      if (res.dynamic_gas_costs > gas) {
        return Result{.state = RunState::kErrorGas};
      }
      gas -= res.dynamic_gas_costs;
    }
    pc += Impl::kInfo.instruction_length;
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

template <bool LoggingEnabled, bool ProfilingEnabled>
void RunInterpreter(Context& ctx, Profiler<ProfilingEnabled>& profiler) {
  EVMZERO_PROFILE_ZONE();

  // The state, pc, and stack state are owned by this function and
  // should not escape this function.
  RunState state = RunState::kRunning;
  int64_t gas = ctx.gas;
  uint256_t* top = ctx.stack.Top();

  auto* padded_code = ctx.padded_code.data();
  auto* pc = padded_code;

#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wpedantic"

  // The dispatch mechanism uses "computed gotos". The dispatch table is defined
  // here, enumerating _all_ possible opcodes.
  static constexpr std::array dispatch_table = {
#define EVMZERO_OPCODE(name, value) &&op_##name,
#define EVMZERO_OPCODE_UNUSED(value) &&op_INVALID,
#include "opcodes.inc"
  };
  static_assert(dispatch_table.size() == 256);

// On each dispatch, the dispatch_table is used to resolve the target address
// for the handling code.
#define DISPATCH()                                                                      \
  do {                                                                                  \
    if (state == RunState::kRunning) {                                                  \
      if constexpr (LoggingEnabled) {                                                   \
        std::cout << ToString(static_cast<op::OpCode>(*pc)) << ", " << ctx.gas << ", "; \
        if (ctx.stack.GetSize() == 0) {                                                 \
          std::cout << "-empty-";                                                       \
        } else {                                                                        \
          std::cout << ctx.stack[0];                                                    \
        }                                                                               \
        std::cout << "\n" << std::flush;                                                \
      }                                                                                 \
      goto* dispatch_table[*pc];                                                        \
    } else {                                                                            \
      goto end;                                                                         \
    }                                                                                   \
  } while (0)

  // Initial dispatch is executed here!
  DISPATCH();

// A valid op code is executed, followed by another DISPATCH call. Since the
// profiler currently doesn't work with recursive op codes, we don't profile
// CREATE and CALL op codes.
#define RUN_OPCODE(opcode)                                      \
  op_##opcode : {                                               \
    EVMZERO_PROFILE_ZONE_N(#opcode);                            \
    profiler.template Start<Marker::opcode>();                  \
    auto res = Run<op::opcode>(pc, gas, top, padded_code, ctx); \
    state = res.state;                                          \
    pc = res.pc;                                                \
    gas = res.gas_left;                                         \
    top = res.top;                                              \
    profiler.template End<Marker::opcode>();                    \
  }                                                             \
  DISPATCH();

#define RUN_OPCODE_NO_PROFILE(opcode)                           \
  op_##opcode : {                                               \
    EVMZERO_PROFILE_ZONE();                                     \
    auto res = Run<op::opcode>(pc, gas, top, padded_code, ctx); \
    state = res.state;                                          \
    pc = res.pc;                                                \
    gas = res.gas_left;                                         \
    top = res.top;                                              \
  }                                                             \
  DISPATCH();

#define EVMZERO_OPCODE(name, value) RUN_OPCODE(name)
#define EVMZERO_OPCODE_CREATE(name, value) RUN_OPCODE_NO_PROFILE(name)
#define EVMZERO_OPCODE_CALL(name, value) RUN_OPCODE_NO_PROFILE(name)
#include "opcodes.inc"

#undef RUN_OPCODE
#undef RUN_OPCODE_NO_PROFILE
#undef DISPATCH

#pragma GCC diagnostic pop

end:
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
}

template void RunInterpreter<false, false>(Context&, Profiler<false>&);
template void RunInterpreter<true, false>(Context&, Profiler<false>&);
template void RunInterpreter<false, true>(Context&, Profiler<true>&);
template void RunInterpreter<true, true>(Context&, Profiler<true>&);

}  // namespace internal

}  // namespace tosca::evmzero

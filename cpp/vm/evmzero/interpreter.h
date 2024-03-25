#pragma once

#include <cstdint>
#include <ostream>
#include <span>
#include <vector>

#include <evmc/evmc.hpp>

#include "vm/evmzero/memory.h"
#include "vm/evmzero/observer.h"
#include "vm/evmzero/sha3_cache.h"
#include "vm/evmzero/stack.h"

namespace tosca::evmzero {

enum class RunState {
  kRunning,
  kDone,
  kReturn,
  kRevert,
  kInvalid,
  kErrorOpcode,
  kErrorGas,
  kErrorStackUnderflow,
  kErrorStackOverflow,
  kErrorJump,
  kErrorReturnDataCopyOutOfBounds,
  kErrorCall,
  kErrorCreate,
  kErrorStaticCall,
};

bool IsSuccess(RunState);

const char* ToString(RunState);
std::ostream& operator<<(std::ostream&, RunState);

struct InterpreterArgs {
  std::span<const uint8_t> padded_code;
  std::span<const uint8_t> valid_jump_targets;
  const evmc_message* message = nullptr;
  const evmc_host_interface* host_interface = nullptr;
  evmc_host_context* host_context = nullptr;
  evmc_revision revision = EVMC_ISTANBUL;
  Sha3Cache* sha3_cache = nullptr;
};

struct InterpreterResult {
  RunState state = RunState::kDone;
  int64_t remaining_gas = 0;
  int64_t refunded_gas = 0;
  std::vector<uint8_t> return_data;
};

template <Observer Observer>
extern InterpreterResult Interpret(const InterpreterArgs&, Observer&);

struct SteppingArgs : InterpreterArgs {
  RunState state = RunState::kRunning;
  uint64_t pc = 0;
  int64_t gas_refunds = 0;
  Stack stack;
  Memory memory;
  int steps = -1;
  std::vector<uint8_t> last_call_return_data;
};

struct SteppingResult : InterpreterResult {
  uint64_t pc = 0;
  Stack stack;
  Memory memory;
  std::vector<uint8_t> last_call_return_data;
};

extern SteppingResult InterpretNSteps(const SteppingArgs& args);

namespace internal {

constexpr int64_t kMaxGas = std::numeric_limits<int64_t>::max();
constexpr int32_t kMaxCallDepth = 1024;

struct Context {
  RunState state = RunState::kRunning;
  bool is_static_call = false;

  uint64_t pc = 0;
  int64_t gas = kMaxGas;
  int64_t gas_refunds = 0;

  std::span<const uint8_t> padded_code;
  std::vector<uint8_t> return_data;
  std::span<const uint8_t> valid_jump_targets;

  Memory memory;
  Stack stack;

  const evmc_message* message = nullptr;

  evmc::HostInterface* host = nullptr;

  evmc_revision revision = EVMC_ISTANBUL;

  Sha3Cache* sha3_cache = nullptr;

  bool CheckJumpDest(uint256_t index) noexcept;

  struct MemoryExpansionCostResult {
    // Resulting memory expansion costs.
    int64_t gas_cost = 0;

    // MemoryExpansionCost also converts the given offset and size parameters
    // from uint256_t to uint64_t, given they are <= UINT64_MAX each.
    uint64_t offset = 0;
    uint64_t size = 0;
  };

  MemoryExpansionCostResult MemoryExpansionCost(uint256_t offset, uint256_t size) noexcept;
};

// Pads the given code with extra STOP/zero bytes to make sure that no operations are exceeding
// the end-of-code boundaries when being executed. By padding the code before executing it,
// bound checks during the execution can be avoided.
std::vector<uint8_t> PadCode(std::span<const uint8_t> code);

template <Observer Observer, bool Stepping = false>
extern void RunInterpreter(Context&, Observer&, int steps = -1);

}  // namespace internal

}  // namespace tosca::evmzero

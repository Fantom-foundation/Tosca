#pragma once

#include <cstdint>
#include <ostream>
#include <span>
#include <vector>

#include <evmc/evmc.hpp>

#include "vm/evmzero/memory.h"
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
  std::span<const uint8_t> code;
  const evmc_message* message = nullptr;
  const evmc_host_interface* host_interface = nullptr;
  evmc_host_context* host_context = nullptr;
  evmc_revision revision = EVMC_ISTANBUL;
};

struct InterpreterResult {
  RunState state = RunState::kDone;
  int64_t remaining_gas = 0;
  int64_t refunded_gas = 0;
  std::vector<uint8_t> return_data;
};

template <bool TracingEnabled>
extern InterpreterResult Interpret(const InterpreterArgs&);

namespace internal {

struct Context {
  RunState state = RunState::kRunning;
  bool is_static_call = false;

  uint64_t pc = 0;
  int64_t gas = std::numeric_limits<int64_t>::max();
  int64_t gas_refunds = 0;

  std::vector<uint8_t> code;
  std::vector<uint8_t> return_data;
  std::vector<uint8_t> valid_jump_targets;

  Memory memory;
  Stack stack;

  const evmc_message* message = nullptr;

  evmc::HostInterface* host = nullptr;

  evmc_revision revision = EVMC_ISTANBUL;

  bool CheckOpcodeAvailable(evmc_revision introduced_in) noexcept;
  bool CheckStaticCallConformance() noexcept;
  bool CheckStackAvailable(uint64_t elements_needed) noexcept;
  bool CheckStackOverflow(uint64_t slots_needed) noexcept;
  bool ApplyGasCost(int64_t gas_cost) noexcept;

  bool CheckJumpDest(uint256_t index) noexcept;
  void FillValidJumpTargetsUpTo(uint64_t index) noexcept;

  struct MemoryExpansionCostResult {
    int64_t gas_cost = 0;
    uint64_t offset = 0;
    uint64_t size = 0;
  };

  MemoryExpansionCostResult MemoryExpansionCost(uint256_t offset, uint256_t size) noexcept;
};

template <bool LoggingEnabled>
extern void RunInterpreter(Context&);

}  // namespace internal

}  // namespace tosca::evmzero

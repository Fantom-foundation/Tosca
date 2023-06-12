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
  kRevert,
  kInvalid,
  kErrorOpcode,
  kErrorGas,
  kErrorStackUnderflow,
  kErrorStackOverflow,
  kErrorJump,
  kErrorCall,
  kErrorCreate,
};

const char* ToString(RunState);
std::ostream& operator<<(std::ostream&, RunState);

struct InterpreterArgs {
  std::span<const uint8_t> code;
  const evmc_message* message = nullptr;
  const evmc_host_interface* host_interface = nullptr;
  evmc_host_context* host_context = nullptr;
};

struct InterpreterResult {
  RunState state = RunState::kDone;
  uint64_t remaining_gas = 0;
  std::vector<uint8_t> return_data;
};

InterpreterResult Interpret(const InterpreterArgs&);

namespace internal {

struct Context {
  RunState state = RunState::kRunning;

  uint64_t pc = 0;
  uint64_t gas = 100000000000llu;

  std::vector<uint8_t> code;
  std::vector<uint8_t> return_data;
  std::vector<uint8_t> valid_jump_targets;

  Memory memory;
  Stack stack;

  const evmc_message* message = nullptr;

  evmc::HostInterface* host = nullptr;

  bool CheckStackAvailable(uint64_t elements_needed) noexcept;
  bool CheckStackOverflow(uint64_t slots_needed) noexcept;
  bool ApplyGasCost(uint64_t gas_cost) noexcept;

  bool CheckJumpDest(uint64_t index) noexcept;
  void FillValidJumpTargetsUpTo(uint64_t index) noexcept;

  uint64_t MemoryExpansionCost(uint64_t new_size) noexcept;
};

void RunInterpreter(Context&);

}  // namespace internal

}  // namespace tosca::evmzero

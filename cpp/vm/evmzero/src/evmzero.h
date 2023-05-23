#pragma once

#include <cstdint>
#include <functional>
#include <string>
#include <unordered_map>
#include <vector>

#include <evmc/evmc.hpp>

#include "evmzero_code_utils.h"
#include "evmzero_uint256.h"

#define PERFORM_GAS_CHECKS true
#define PERFORM_STACK_CHECKS true

namespace tosca::evmzero {

enum class RunState {
  kRunning,
  kDone,
  kInvalid,
  kErrorOpcode,
  kErrorGas,
  kErrorStack,
  kErrorJump,
  kErrorCall,
  kErrorCreate,
};

inline constexpr const char* ToString(RunState state) {
  switch (state) {
    case RunState::kRunning:
      return "Running";
    case RunState::kDone:
      return "Done";
    case RunState::kErrorOpcode:
      return "ErrorOpcode";
    case RunState::kErrorGas:
      return "ErrorGas";
    case RunState::kErrorStack:
      return "ErrorStack";
    case RunState::kErrorJump:
      return "ErrorJump";
    case RunState::kInvalid:
      return "Invalid";
    case RunState::kErrorCall:
      return "ErrorCall";
    case RunState::kErrorCreate:
      return "ErrorCreate";
  }
  return "UNKNOWN_STATE";
}

namespace op {
enum OpCodes : uint8_t {
  STOP = 0x00,
  ADD = 0x01,
  MUL = 0x02,
  SUB = 0x03,
  DIV = 0x04,
  SDIV = 0x05,
  MOD = 0x06,
  SMOD = 0x07,
  ADDMOD = 0x08,
  MULMOD = 0x09,
  EXP = 0x0A,
  SIGNEXTEND = 0x0B,
  LT = 0x10,
  GT = 0x11,
  SLT = 0x12,
  SGT = 0x13,
  EQ = 0x14,
  ISZERO = 0x15,
  AND = 0x16,
  OR = 0x17,
  XOR = 0x18,
  NOT = 0x19,
  BYTE = 0x1A,
  SHL = 0x1B,
  SHR = 0x1C,
  SAR = 0x1D,
  SHA3 = 0x20,
  ADDRESS = 0x30,
  BALANCE = 0x31,
  ORIGIN = 0x32,
  CALLER = 0x33,
  CALLVALUE = 0x34,
  CALLDATALOAD = 0x35,
  CALLDATASIZE = 0x36,
  CALLDATACOPY = 0x37,
  CODESIZE = 0x38,
  CODECOPY = 0x39,
  GASPRICE = 0x3A,
  EXTCODESIZE = 0x3B,
  EXTCODECOPY = 0x3C,
  RETURNDATASIZE = 0x3D,
  RETURNDATACOPY = 0x3E,
  EXTCODEHASH = 0x3F,
  BLOCKHASH = 0x40,
  COINBASE = 0x41,
  TIMESTAMP = 0x42,
  NUMBER = 0x43,
  DIFFICULTY = 0x44,
  GASLIMIT = 0x45,
  CHAINID = 0x46,
  SELFBALANCE = 0x47,
  BASEFEE = 0x48,
  POP = 0x50,
  MLOAD = 0x51,
  MSTORE = 0x52,
  MSTORE8 = 0x53,
  SLOAD = 0x54,
  SSTORE = 0x55,
  JUMP = 0x56,
  JUMPI = 0x57,
  PC = 0x58,
  MSIZE = 0x59,
  GAS = 0x5A,
  JUMPDEST = 0x5B,

  // clang-format off
  PUSH1 = 0x60,
  PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11,
  PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20,
  PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29,
  PUSH30, PUSH31, PUSH32,

  DUP1 = 0x80,
  DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12,
  DUP13, DUP14, DUP15, DUP16,

  SWAP1 = 0x90,
  SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11,
  SWAP12, SWAP13, SWAP14, SWAP15, SWAP16,

  LOG0 = 0xA0,
  LOG1, LOG2, LOG3, LOG4,
  // clang-format on

  CREATE = 0xF0,
  CALL = 0xF1,
  CALLCODE = 0xF2,
  RETURN = 0xF3,
  DELEGATECALL = 0xF4,
  CREATE2 = 0xF5,
  STATICCALL = 0xFA,
  REVERT = 0xFD,
  INVALID = 0xFE,
  SELFDESTRUCT = 0xFF,
};
static_assert(PUSH32 == 0x7F);
static_assert(DUP16 == 0x8F);
static_assert(SWAP16 == 0x9F);
static_assert(LOG4 == 0xA4);
}  // namespace op

struct Context {
  uint64_t pc = 0;
  uint64_t gas = 100000000000llu;
  uint64_t highest_known_code_pc = 0;
  uint64_t max_stack_size = 10000;
  uint64_t current_mem_cost = 0;
  uint64_t stack_pos = 0;
  std::vector<uint256_t> stack;
  std::vector<uint8_t> memory;
  std::vector<uint8_t> code;
  std::vector<uint8_t> valid_jump_target;
  RunState state = RunState::kRunning;

  const evmc_message* message = nullptr;
  evmc::HostContext host;
  std::vector<uint8_t> return_data;

  // Putting anything extra in here (or doing anything to the layout) is really
  // dangerous for performance!

  Context(const std::string& code_string, const evmc_message* message, const evmc_host_interface& host_interface,
          evmc_host_context* host_context)
      : message(message), host(host_interface, host_context) {
    code = ByteCodeStringToBinary(code_string);
    valid_jump_target.resize(code.size());
    stack.resize(max_stack_size);
  }
};

// This is a global since putting it in the context is really slow; only used to
// silence tests.
inline bool g_global_interpreter_state_report_errors = true;

}  // namespace tosca::evmzero

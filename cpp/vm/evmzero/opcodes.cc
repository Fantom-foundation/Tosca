#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero::op {

ValidJumpTargetsBuffer CalculateValidJumpTargets(std::span<const uint8_t> code) {
  ValidJumpTargetsBuffer valid_jump_targets(code.size());

  for (size_t i = 0; i < code.size(); ++i) {
    const auto instruction = code[i];
    if (op::PUSH1 <= instruction && instruction <= op::PUSH32) {
      i += instruction - op::PUSH1 + 1;  // skip arguments
    } else {
      valid_jump_targets[i] = instruction == op::JUMPDEST;
    }
  }

  return valid_jump_targets;
}

}  // namespace tosca::evmzero::op

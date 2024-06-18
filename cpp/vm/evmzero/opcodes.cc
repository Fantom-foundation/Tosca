// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

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

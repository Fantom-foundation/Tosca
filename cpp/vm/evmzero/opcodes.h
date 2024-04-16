//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of 
// this software will be governed by the GNU Lesser General Public Licence v3.
//

#pragma once

#include <cstddef>
#include <cstdint>
#include <span>
#include <vector>

namespace tosca::evmzero::op {

enum OpCode : uint8_t {
#define EVMZERO_OPCODE(name, value) name = value,
#include "opcodes.inc"
};

constexpr inline size_t kNumOpCodes = 0
#define EVMZERO_OPCODE(name, value) +1
#include "opcodes.inc"
    ;

constexpr inline size_t kNumUnusedOpCodes = 0
#define EVMZERO_OPCODE_UNUSED(value) +1
#include "opcodes.inc"
    ;

constexpr inline size_t kNumUsedAndUnusedOpCodes = kNumOpCodes + kNumUnusedOpCodes;

constexpr inline bool IsUsedOpCode(OpCode op) {
#define EVMZERO_OPCODE_UNUSED(value)      \
  if (static_cast<OpCode>(value) == op) { \
    return false;                         \
  }
#include "opcodes.inc"
  return true;
}

constexpr inline bool IsCallOpCode(OpCode op) {
#define EVMZERO_OPCODE_CREATE(name, value) \
  if (static_cast<OpCode>(value) == op) {  \
    return true;                           \
  }
#define EVMZERO_OPCODE_CALL(name, value) EVMZERO_OPCODE_CREATE(name, value)
#include "opcodes.inc"
  return false;
}

constexpr inline bool IsExternalOpCode(OpCode op) {
#define EVMZERO_OPCODE_EXTERNAL(name, value) \
  if (static_cast<OpCode>(value) == op) {    \
    return true;                             \
  }
#include "opcodes.inc"
  return false;
}

constexpr inline const char* ToString(OpCode op) {
  switch (op) {
#define EVMZERO_OPCODE(name, value) \
  case op::name:                    \
    return #name;
#include "opcodes.inc"
  }
  return "UNKNOWN";
}

using ValidJumpTargetsBuffer = std::vector<uint8_t>;
ValidJumpTargetsBuffer CalculateValidJumpTargets(std::span<const uint8_t> code);

}  // namespace tosca::evmzero::op

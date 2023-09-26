#pragma once

#include <cstdint>
#include <span>
#include <vector>

namespace tosca::evmzero::op {

enum OpCode : uint8_t {
#define EVMZERO_OPCODE(name, value) name = value,
#include "opcodes.inc"
};

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

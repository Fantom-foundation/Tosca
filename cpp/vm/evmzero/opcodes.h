#pragma once

#include <cstdint>

namespace tosca::evmzero::op {

enum OpCodes : uint8_t {
#define EVMZERO_OPCODE(name, value) name = value,
#include "opcodes.inc"
};

constexpr const char* ToString(OpCodes op) {
  switch (op) {
#define EVMZERO_OPCODE(name, value) \
  case op::name:                    \
    return #name;
#include "opcodes.inc"
  }
  return "UNKNOWN";
}

}  // namespace tosca::evmzero::op

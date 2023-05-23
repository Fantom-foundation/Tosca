#pragma once

#include <cinttypes>
#include <cstdint>
#include <cstdio>
#include <unordered_map>

#include <evmc/evmc.hpp>

#include "evmzero.h"

#ifndef NDEBUG
#define DBG_TRACE_INSTRUCTION TraceInstruction(ctx, __FUNCTION__);
#define DBG_TRACE_INSTRUCTION_TEMPLATE(N) TraceInstruction(ctx, __FUNCTION__, N);
#else
#define DBG_TRACE_INSTRUCTION
#define DBG_TRACE_INSTRUCTION_TEMPLATE(N)
#endif

namespace tosca::evmzero {

inline void PrintStack(const Context& ctx) {
  for (uint64_t i = 1; i <= ctx.stack_pos; ++i) {
    fprintf(stderr, "%s, ", ToString(ctx.stack[i]).c_str());
  }
}

inline void PrintMemory(const std::vector<uint8_t>& memory) {
  for (const auto& elem : memory) {
    fprintf(stderr, "%d, ", elem);
  }
}

inline void PrintStorage(const std::unordered_map<evmc::bytes32, evmc::bytes32>& storage) {
  for (const auto& [k, v] : storage) {
    uint256_t key = ToUint256(k);
    uint256_t value = ToUint256(v);
    fprintf(stderr, "\t%s: %s\n", ToString(key).c_str(), ToString(value).c_str());
  }
}

inline void TraceInstruction(const Context& ctx, const char* name, int n = 0) noexcept {
  fprintf(stderr, "%4" PRIu64 ": (0x%02hhx) ", ctx.pc, ctx.code[ctx.pc]);
  if (n == 0) {
    fprintf(stderr, "%10s", name);
  } else {
    fprintf(stderr, "%8s%-2d", name, n);
  }
  fprintf(stderr, " | ");
  PrintStack(ctx);
  fprintf(stderr, "\n");
}

}  // namespace tosca::evmzero

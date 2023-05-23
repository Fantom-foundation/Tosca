#pragma once

#include <cinttypes>
#include <cstdio>

#include <tosca/macro_utils.h>

#include "evmzero.h"

namespace tosca::evmzero {

inline void OutOfGas(Context& ctx) {
  if (g_global_interpreter_state_report_errors) {
    printf("Out of gas at %" PRIu64 "\n", ctx.pc);
  }
  ctx.state = RunState::kErrorGas;
}

[[nodiscard]] TOSCA_FORCEINLINE inline bool ApplyGasCost(Context& ctx, uint64_t gas_cost) noexcept {
#if PERFORM_GAS_CHECKS
  if (ctx.gas < gas_cost) [[unlikely]] {
    OutOfGas(ctx);
    return false;
  }
  ctx.gas -= gas_cost;
#else
  (void)(ctx);
  (void)(gas_cost);
#endif
  return true;
}

[[maybe_unused]] inline void OutOfStack(Context& ctx) {
  if (g_global_interpreter_state_report_errors) {
    printf("Out of stack at %" PRIu64 "\n", ctx.pc);
  }
  ctx.state = RunState::kErrorStack;
}

[[nodiscard]] TOSCA_FORCEINLINE inline bool CheckStackAvailable(Context& ctx, uint64_t elems_needed) noexcept {
#if PERFORM_STACK_CHECKS
  if (ctx.stack_pos < elems_needed) [[unlikely]] {
    OutOfStack(ctx);
    return false;
  }
#else
  (void)(ctx);
  (void)(elems_needed);
#endif
  return true;
}

[[nodiscard]] TOSCA_FORCEINLINE inline bool CheckStackOverflow(Context& ctx, uint64_t elems_needed) noexcept {
#if PERFORM_STACK_CHECKS
  if (ctx.max_stack_size - ctx.stack_pos < elems_needed) [[unlikely]] {
    OutOfStack(ctx);
    return false;
  }
#else
  (void)(ctx);
  (void)(elems_needed);
#endif
  return true;
}

inline void WrongJumpDest(Context& ctx, uint64_t counter) noexcept {
  if (g_global_interpreter_state_report_errors) {
    printf("Wrong jump dest at %" PRIu64 ", trying to jump to %" PRIu64 ", which is 0x%02hhX\n", ctx.pc, counter,
           ctx.code[counter]);
  }
  ctx.state = RunState::kErrorJump;
}

TOSCA_FORCEINLINE inline void CheckJumpDestUpTo(Context& ctx, uint64_t counter) {
  if (ctx.highest_known_code_pc >= counter) [[likely]]
    return;
  uint64_t cur = ctx.highest_known_code_pc;
  while (cur <= counter) {
    uint8_t op = ctx.code[cur];
    if (op < op::PUSH1 || op > op::PUSH31) {
      ctx.valid_jump_target[cur] = op == op::JUMPDEST;
#ifndef NDEBUG
      if (ctx.valid_jump_target[cur]) {
        printf("Marked 0x%02hhX at %" PRIu64 " as valid jump dest\n", ctx.code[cur], cur);
      }
#endif
      cur++;
    } else {
      cur += op - op::PUSH1 + 2;
    }
  }
  ctx.highest_known_code_pc = counter;
}

[[nodiscard]] TOSCA_FORCEINLINE inline bool CheckJumpDest(Context& ctx, uint64_t counter) noexcept {
  CheckJumpDestUpTo(ctx, counter);
  if (counter >= ctx.code.size() || !ctx.valid_jump_target[counter]) [[unlikely]] {
    WrongJumpDest(ctx, counter);
    return false;
  }
  return true;
}

[[nodiscard]] TOSCA_FORCEINLINE inline uint256_t StackPop(Context& ctx) noexcept { return ctx.stack[ctx.stack_pos--]; }

TOSCA_FORCEINLINE inline void StackPush(Context& ctx, uint256_t value) noexcept { ctx.stack[++ctx.stack_pos] = value; }

[[nodiscard]] inline uint64_t MemoryCost(uint64_t memory_byte_size) noexcept {
  uint64_t memory_size_word = (memory_byte_size + 31) / 32;
  return (memory_size_word * memory_size_word) / 512 + (3 * memory_size_word);
}

[[nodiscard]] inline uint64_t DynamicMemoryCost(uint64_t new_address, uint64_t current_mem_cost) noexcept {
  auto new_cost = MemoryCost(new_address);
  return new_cost > current_mem_cost ? new_cost - current_mem_cost : 0;
}

inline void GrowMemory(Context& ctx, uint64_t new_size) {
  ctx.current_mem_cost = DynamicMemoryCost(new_size, ctx.current_mem_cost);
  if (ctx.memory.size() < new_size) {
    ctx.memory.resize(new_size);
  }
}

}  // namespace tosca::evmzero

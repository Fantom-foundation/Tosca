#include <gtest/gtest.h>

#include <algorithm>
#include <cinttypes>
#include <cstdint>
#include <cstdio>
#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

#include <evmc/evmc.hpp>

#include "evmzero_dummy_host.h"
#include "evmzero_interpreter.h"
#include "evmzero_uint256.h"

namespace tosca::evmzero {

struct ErrorSilencer {
  ErrorSilencer() { g_global_interpreter_state_report_errors = false; }
  ~ErrorSilencer() { g_global_interpreter_state_report_errors = true; }
};

static void SetStack(Context& ctx, const std::vector<uint256_t>& stack) {
  std::copy(stack.begin(), stack.end(), ctx.stack.begin() + 1);
  ctx.stack_pos = stack.size();
}

static void PrintTestStack(const std::vector<uint256_t>& stack) {
  for (const auto& elem : stack) {
    fprintf(stderr, "%s, ", ToString(elem).c_str());
  }
}

static bool TestStackEqual(const Context& ctx, const std::vector<uint256_t>& stack) {
  if (ctx.stack_pos != stack.size()) {
    return false;
  }

  uint64_t i = 1;
  for (const auto& elem : stack) {
    if (ctx.stack[i++] != elem) {
      return false;
    }
  }
  return true;
}

static void SetMemory(Context& ctx, const std::vector<uint8_t>& memory) {
  ctx.memory = memory;

  // While we assume the gas costs have already been handled, we need to keep
  // track of the memory costs.
  ctx.current_mem_cost = DynamicMemoryCost(ctx.memory.size() - 1, 0);
}

// A basic test description describes the interpreter state before and after a
// code fragment is executed memory is ignored / optional.
struct BasicTestDesc {
  std::string code;
  std::vector<uint256_t> stack_before;
  std::vector<uint256_t> stack_after;
  uint64_t gas_before;
  uint64_t gas_after;
  RunState expected_state;
  std::vector<uint8_t> memory_before;
  std::vector<uint8_t> memory_after;
  std::unordered_map<evmc::address, DummyHost::AccountData> accounts;
  std::unordered_map<evmc::address, DummyHost::AccountData> accounts_after;
  evmc_message message{};
  evmc_tx_context tx_context{};
  std::vector<uint8_t> return_data;
};

static void RunBasicTest(const BasicTestDesc& test) {
  ErrorSilencer silence;

  evmc_message message = test.message;
  message.gas = static_cast<int64_t>(test.gas_before);

  DummyHost host(test.accounts);
  if (!host.GetAccountData(message.recipient)) {
    host.SetAccountData(message.recipient, {});
  }
  host.SetTxContext(test.tx_context);

  auto evmc_host_interface = host.GetEvmcHostInterface();
  Context ctx(test.code, &message, evmc_host_interface, host);
  ctx.gas = test.gas_before;
  SetStack(ctx, test.stack_before);
  SetMemory(ctx, test.memory_before);
  ctx.return_data = test.return_data;
  RunInterpreter(ctx);

  if (ctx.state != test.expected_state) {
    fprintf(stderr, "expected state mismatch!\n");
    fprintf(stderr, "Expected: %s\n", ToString(test.expected_state));
    fprintf(stderr, "     Got: %s\n", ToString(ctx.state));
    FAIL();
  }

  EXPECT_EQ(ctx.gas, test.gas_after);

  if (ctx.gas != test.gas_after) {
    fprintf(stderr, "gas mismatch!\n");
    fprintf(stderr, "Expected: %" PRIu64 "\n", test.gas_after);
    fprintf(stderr, "     Got: %" PRIu64 "\n", ctx.gas);
    FAIL();
  }

  if (!TestStackEqual(ctx, test.stack_after)) {
    fprintf(stderr, "stack mismatch!\n");
    fprintf(stderr, "Expected: ");
    PrintTestStack(test.stack_after);
    fprintf(stderr, "\n");
    fprintf(stderr, "     Got: ");
    PrintStack(ctx);
    fprintf(stderr, "\n");
    FAIL();
  }

  if (ctx.memory != test.memory_after) {
    fprintf(stderr, "expected memory mismatch!\n");
    fprintf(stderr, "Expected: ");
    PrintMemory(test.memory_after);
    fprintf(stderr, "\n");
    fprintf(stderr, "     Got: ");
    PrintMemory(ctx.memory);
    fprintf(stderr, "\n");
    FAIL();
  }

  if (!test.accounts_after.empty() && test.accounts_after != host.GetAccounts()) {
    fprintf(stderr, "account data mismatch!\n");
    FAIL();
  }
}

///////////////////////////////////////////////////////////
// STOP
TEST(InterpreterTests, STOP) {
  RunBasicTest({
      .code = "00",
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kDone,
  });
}

///////////////////////////////////////////////////////////
// ADD
TEST(InterpreterTests, ADD) {
  RunBasicTest({
      .code = "01",
      .stack_before = {1, 6},
      .stack_after = {7},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, ADD_Overflow) {
  RunBasicTest({
      .code = "01",
      .stack_before = {1, kUint256Max},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, ADD_OutOfGas) {
  RunBasicTest({
      .code = "01",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, ADD_StackError) {
  RunBasicTest({
      .code = "01",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// MUL
TEST(InterpreterTests, MUL) {
  RunBasicTest({
      .code = "02",
      .stack_before = {10, 0},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "02",
      .stack_before = {5, 4},
      .stack_after = {20},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, MUL_Overflow) {
  RunBasicTest({
      .code = "02",
      .stack_before = {kUint256Max, 2},
      .stack_after = {kUint256Max - 1},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, MUL_OutOfGas) {
  RunBasicTest({
      .code = "02",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, MUL_StackError) {
  RunBasicTest({
      .code = "02",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SUB
TEST(InterpreterTests, SUB) {
  RunBasicTest({
      .code = "03",
      .stack_before = {10, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "03",
      .stack_before = {5, 10},
      .stack_after = {5},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SUB_Underflow) {
  RunBasicTest({
      .code = "03",
      .stack_before = {1, 0},
      .stack_after = {kUint256Max},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SUB_OutOfGas) {
  RunBasicTest({
      .code = "03",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SUB_StackError) {
  RunBasicTest({
      .code = "03",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// DIV
TEST(InterpreterTests, DIV) {
  RunBasicTest({
      .code = "04",
      .stack_before = {10, 10},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "04",
      .stack_before = {2, 1},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, DIV_ByZero) {
  RunBasicTest({
      .code = "04",
      .stack_before = {0, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, DIV_OutOfGas) {
  RunBasicTest({
      .code = "04",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, DIV_StackError) {
  RunBasicTest({
      .code = "04",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SDIV
TEST(InterpreterTests, SDIV) {
  RunBasicTest({
      .code = "05",
      .stack_before = {10, 10},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "05",
      .stack_before = {kUint256Max, kUint256Max - 1},
      .stack_after = {2},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SDIV_ByZero) {
  RunBasicTest({
      .code = "05",
      .stack_before = {0, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SDIV_OutOfGas) {
  RunBasicTest({
      .code = "05",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SDIV_StackError) {
  RunBasicTest({
      .code = "05",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// MOD
TEST(InterpreterTests, MOD) {
  RunBasicTest({
      .code = "06",
      .stack_before = {3, 10},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "06",
      .stack_before = {5, 17},
      .stack_after = {2},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, MOD_ByZero) {
  RunBasicTest({
      .code = "06",
      .stack_before = {0, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, MOD_OutOfGas) {
  RunBasicTest({
      .code = "06",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, MOD_StackError) {
  RunBasicTest({
      .code = "06",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SMOD
TEST(InterpreterTests, SMOD) {
  RunBasicTest({
      .code = "07",
      .stack_before = {3, 10},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "07",
      .stack_before = {kUint256Max - 2, kUint256Max - 7},
      .stack_after = {kUint256Max - 1},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SMOD_ByZero) {
  RunBasicTest({
      .code = "07",
      .stack_before = {0, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SMOD_OutOfGas) {
  RunBasicTest({
      .code = "07",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SMOD_StackError) {
  RunBasicTest({
      .code = "07",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// ADDMOD
TEST(InterpreterTests, ADDMOD) {
  RunBasicTest({
      .code = "08",
      .stack_before = {8, 10, 10},
      .stack_after = {4},
      .gas_before = 10,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "08",
      .stack_before = {2, 2, kUint256Max},
      .stack_after = {1},
      .gas_before = 10,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, ADDMOD_ByZero) {
  RunBasicTest({
      .code = "08",
      .stack_before = {0, 10, 10},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, ADDMOD_OutOfGas) {
  RunBasicTest({
      .code = "08",
      .stack_before = {1, 10, 10},
      .stack_after = {1, 10, 10},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, ADDMOD_StackError) {
  RunBasicTest({
      .code = "08",
      .stack_before = {1, 2},
      .stack_after = {1, 2},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// MULMOD
TEST(InterpreterTests, MULMOD) {
  RunBasicTest({
      .code = "09",
      .stack_before = {8, 10, 10},
      .stack_after = {4},
      .gas_before = 10,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "09",
      .stack_before = {12, kUint256Max, kUint256Max},
      .stack_after = {9},
      .gas_before = 10,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, MULMOD_ByZero) {
  RunBasicTest({
      .code = "09",
      .stack_before = {0, 10, 10},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, MULMOD_OutOfGas) {
  RunBasicTest({
      .code = "09",
      .stack_before = {8, 10, 10},
      .stack_after = {8, 10, 10},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, MULMOD_StackError) {
  RunBasicTest({
      .code = "09",
      .stack_before = {1, 2},
      .stack_after = {1, 2},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// EXP
TEST(InterpreterTests, EXP) {
  RunBasicTest({
      .code = "0A",
      .stack_before = {2, 10},
      .stack_after = {100},
      .gas_before = 200,
      .gas_after = 140,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "0A",
      .stack_before = {4747, 1},
      .stack_after = {1},
      .gas_before = 200,
      .gas_after = 90,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, EXP_OutOfGas_Static) {
  RunBasicTest({
      .code = "0A",
      .stack_before = {2, 40000},
      .stack_after = {2, 40000},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, EXP_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "0A",
      .stack_before = {2, 40000},
      .stack_after = {},
      .gas_before = 12,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, EXP_StackError) {
  RunBasicTest({
      .code = "0A",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 200,
      .gas_after = 200,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SIGNEXTEND
TEST(InterpreterTests, SIGNEXTEND) {
  RunBasicTest({
      .code = "0B",
      .stack_before = {0xFF, 0},
      .stack_after = {kUint256Max},
      .gas_before = 10,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "0B",
      .stack_before = {0x7F, 0},
      .stack_after = {0x7F},
      .gas_before = 10,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "0B",
      .stack_before = {0xFF7F, 0},
      .stack_after = {0x7F},
      .gas_before = 10,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "0B",
      .stack_before = {0xFF7F, 1},
      .stack_after = {kUint256Max - 0x80},
      .gas_before = 10,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SIGNEXTEND_OutOfGas) {
  RunBasicTest({
      .code = "0B",
      .stack_before = {0xFF, 0},
      .stack_after = {0xFF, 0},
      .gas_before = 4,
      .gas_after = 4,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SIGNEXTEND_StackError) {
  RunBasicTest({
      .code = "0B",
      .stack_before = {0xFF},
      .stack_after = {0xFF},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// LT
TEST(InterpreterTests, LT) {
  RunBasicTest({
      .code = "10",
      .stack_before = {10, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "10",
      .stack_before = {9, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "10",
      .stack_before = {10, 9},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, LT_OutOfGas) {
  RunBasicTest({
      .code = "10",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, LT_StackError) {
  RunBasicTest({
      .code = "10",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// GT
TEST(InterpreterTests, GT) {
  RunBasicTest({
      .code = "11",
      .stack_before = {10, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "11",
      .stack_before = {9, 10},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "11",
      .stack_before = {10, 9},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, GT_OutOfGas) {
  RunBasicTest({
      .code = "11",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, GT_StackError) {
  RunBasicTest({
      .code = "11",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SLT
TEST(InterpreterTests, SLT) {
  RunBasicTest({
      .code = "12",
      .stack_before = {10, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "12",
      .stack_before = {9, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "12",
      .stack_before = {10, 9},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "12",
      .stack_before = {0, kUint256Max},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SLT_OutOfGas) {
  RunBasicTest({
      .code = "12",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SLT_StackError) {
  RunBasicTest({
      .code = "12",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SGT
TEST(InterpreterTests, SGT) {
  RunBasicTest({
      .code = "13",
      .stack_before = {10, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "13",
      .stack_before = {9, 10},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "13",
      .stack_before = {10, 9},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "13",
      .stack_before = {kUint256Max, 0},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SGT_OutOfGas) {
  RunBasicTest({
      .code = "13",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SGT_StackError) {
  RunBasicTest({
      .code = "13",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// EQ
TEST(InterpreterTests, EQ) {
  RunBasicTest({
      .code = "14",
      .stack_before = {10, 10},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "14",
      .stack_before = {9, 10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, EQ_OutOfGas) {
  RunBasicTest({
      .code = "14",
      .stack_before = {1, 6},
      .stack_after = {1, 6},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, EQ_StackError) {
  RunBasicTest({
      .code = "14",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// ISZERO
TEST(InterpreterTests, ISZERO) {
  RunBasicTest({
      .code = "15",
      .stack_before = {10},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "15",
      .stack_before = {0},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, ISZERO_OutOfGas) {
  RunBasicTest({
      .code = "15",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, ISZERO_StackError) {
  RunBasicTest({
      .code = "15",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// AND
TEST(InterpreterTests, AND) {
  RunBasicTest({
      .code = "16",
      .stack_before = {0xF, 0xF},
      .stack_after = {0xF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "16",
      .stack_before = {0, 0xFF},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, AND_OutOfGas) {
  RunBasicTest({
      .code = "16",
      .stack_before = {0xF, 0xF},
      .stack_after = {0xF, 0xF},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, AND_StackError) {
  RunBasicTest({
      .code = "16",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// OR
TEST(InterpreterTests, OR) {
  RunBasicTest({
      .code = "17",
      .stack_before = {0xF, 0xF0},
      .stack_after = {0xFF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "17",
      .stack_before = {0xFF, 0xFF},
      .stack_after = {0xFF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, OR_OutOfGas) {
  RunBasicTest({
      .code = "17",
      .stack_before = {0xF, 0xF0},
      .stack_after = {0xF, 0xF0},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, OR_StackError) {
  RunBasicTest({
      .code = "17",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// XOR
TEST(InterpreterTests, XOR) {
  RunBasicTest({
      .code = "18",
      .stack_before = {0xF, 0xF0},
      .stack_after = {0xFF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "18",
      .stack_before = {0xFF, 0xFF},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, XOR_OutOfGas) {
  RunBasicTest({
      .code = "18",
      .stack_before = {0xF, 0xF0},
      .stack_after = {0xF, 0xF0},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, XOR_StackError) {
  RunBasicTest({
      .code = "18",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// NOT
TEST(InterpreterTests, NOT) {
  RunBasicTest({
      .code = "19",
      .stack_before = {0},
      .stack_after = {kUint256Max},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "19",
      .stack_before = {0xFF},
      .stack_after = {kUint256Max - 0xFF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, NOT_OutOfGas) {
  RunBasicTest({
      .code = "19",
      .stack_before = {0},
      .stack_after = {0},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, NOT_StackError) {
  RunBasicTest({
      .code = "19",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// BYTE
TEST(InterpreterTests, BYTE) {
  RunBasicTest({
      .code = "1A",
      .stack_before = {0xFF, 31},
      .stack_after = {0xFF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "1A",
      .stack_before = {0xFF00, 30},
      .stack_after = {0xFF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, BYTE_OutOfRange) {
  RunBasicTest({
      .code = "1A",
      .stack_before = {kUint256Max, 32},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, BYTE_OutOfGas) {
  RunBasicTest({
      .code = "1A",
      .stack_before = {0xFF, 31},
      .stack_after = {0xFF, 31},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, BYTE_StackError) {
  RunBasicTest({
      .code = "1A",
      .stack_before = {0xFF},
      .stack_after = {0xFF},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SHL
TEST(InterpreterTests, SHL) {
  RunBasicTest({
      .code = "1B",
      .stack_before = {1, 1},
      .stack_after = {2},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "1B",
      .stack_before = {uint256_t{0xFF} << 248, 4},
      .stack_after = {uint256_t{0xF} << 252},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "1B",
      .stack_before = {7, 256},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SHL_OutOfGas) {
  RunBasicTest({
      .code = "1B",
      .stack_before = {1, 1},
      .stack_after = {1, 1},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SHL_StackError) {
  RunBasicTest({
      .code = "1B",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SHR
TEST(InterpreterTests, SHR) {
  RunBasicTest({
      .code = "1C",
      .stack_before = {2, 1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "1C",
      .stack_before = {0xFF, 4},
      .stack_after = {0xF},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "1C",
      .stack_before = {kUint256Max, 256},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SHR_OutOfGas) {
  RunBasicTest({
      .code = "1C",
      .stack_before = {2, 1},
      .stack_after = {2, 1},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SHR_StackError) {
  RunBasicTest({
      .code = "1C",
      .stack_before = {2},
      .stack_after = {2},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SAR
TEST(InterpreterTests, SAR) {
  RunBasicTest({
      .code = "1D",
      .stack_before = {2, 1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "1D",
      .stack_before = {kUint256Max - 0xF, 4},
      .stack_after = {kUint256Max},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SAR_OutOfGas) {
  RunBasicTest({
      .code = "1D",
      .stack_before = {2, 1},
      .stack_after = {2, 1},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SAR_StackError) {
  RunBasicTest({
      .code = "1D",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SHA3
TEST(InterpreterTests, SHA3) {
  RunBasicTest({
      .code = "20",
      .stack_before = {4, 0},
      .stack_after = {uint256_t(0x79A1BC8F0BB2C238, 0x9522D0CF0F73282C, 0x46EF02C2223570DA, 0x29045A592007D0C2)},
      .gas_before = 100,
      .gas_after = 64,
      .expected_state = RunState::kDone,
      .memory_before = {0xFF, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0xFF, 0xFF},
  });
}

TEST(InterpreterTests, SHA3_GrowMemory) {
  RunBasicTest({
      .code = "20",
      .stack_before = {4, 0},
      .stack_after = {uint256_t(0x64633A4ACBD3244C, 0xF7685EBD40E852B1, 0x55364C7B4BBF0BB7, 0xE8E77626586F73B9)},
      .gas_before = 100,
      .gas_after = 61,
      .expected_state = RunState::kDone,
      .memory_after = {0x00, 0x00, 0x00, 0x00},
  });
}

TEST(InterpreterTests, SHA3_OutOfGas_Static) {
  RunBasicTest({
      .code = "20",
      .stack_before = {4, 0},
      .stack_after = {4, 0},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SHA3_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "20",
      .stack_before = {4, 0},
      .stack_after = {},
      .gas_before = 32,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SHA3_StackError) {
  RunBasicTest({
      .code = "20",
      .stack_before = {4},
      .stack_after = {4},
      .gas_before = 100,
      .gas_after = 100,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// ADDRESS
TEST(InterpreterTests, ADDRESS) {
  RunBasicTest({
      .code = "30",
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "30",
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .message{.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, ADDRESS_OutOfGas) {
  RunBasicTest({
      .code = "30",
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// BALANCE
TEST(InterpreterTests, BALANCE) {
  RunBasicTest({
      .code = "31",
      .stack_before = {0x42},
      .stack_after = {0x21},
      .gas_before = 3000,
      .gas_after = 400,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0x42), {.balance = evmc::uint256be(0x21)}}},
  });
}

TEST(InterpreterTests, BALANCE_InvalidAccount) {
  RunBasicTest({
      .code = "31",
      .stack_before = {0x42},
      .stack_after = {0},
      .gas_before = 3000,
      .gas_after = 400,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, BALANCE_OutOfGas) {
  RunBasicTest({
      .code = "31",
      .stack_before = {0x42},
      .stack_after = {},
      .gas_before = 100,
      .gas_after = 100,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, BALANCE_StackError) {
  RunBasicTest({
      .code = "31",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 3000,
      .gas_after = 3000,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// ORIGIN
TEST(InterpreterTests, ORIGIN) {
  RunBasicTest({
      .code = "32",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "32",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.tx_origin = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, ORIGIN_OutOfGas) {
  RunBasicTest({
      .code = "32",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// CALLER
TEST(InterpreterTests, CALLER) {
  RunBasicTest({
      .code = "33",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "33",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .message{.sender = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, CALLER_OutOfGas) {
  RunBasicTest({
      .code = "33",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// CALLVALUE
TEST(InterpreterTests, CALLVALUE) {
  RunBasicTest({
      .code = "34",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "34",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 7,
      .gas_after = 5,
      .expected_state = RunState::kDone,
      .message{.value = evmc::uint256be(0x42)},
  });
}

TEST(InterpreterTests, CALLVALUE_OutOfGas) {
  RunBasicTest({
      .code = "34",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, DISABLED_CALLVALUE_StackOverflow) {}

///////////////////////////////////////////////////////////
// CALLDATALOAD
TEST(InterpreterTests, CALLDATALOAD) {
  RunBasicTest({
      .code = "35",
      .stack_before = {0},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });

  std::array<uint8_t, 32> input_data{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,  //
                                     0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,  //
                                     0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,  //
                                     0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37};
  RunBasicTest({
      .code = "35",
      .stack_before = {0},
      .stack_after = {uint256_t(0x3031323334353637, 0x2021222324252627, 0x1011121314151617, 0x0001020304050607)},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });

  RunBasicTest({
      .code = "35",
      .stack_before = {30},
      .stack_after = {uint256_t(0, 0, 0, 0x3637000000000000)},
      .gas_before = 7,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTests, CALLDATALOAD_OutOfGas) {
  RunBasicTest({
      .code = "35",
      .stack_before = {0},
      .stack_after = {0},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, CALLDATALOAD_StackError) {
  RunBasicTest({
      .code = "35",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 7,
      .gas_after = 7,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// CALLDATASIZE
TEST(InterpreterTests, CALLDATASIZE) {
  RunBasicTest({
      .code = "36",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });

  std::array<uint8_t, 3> input_data{};
  RunBasicTest({
      .code = "36",
      .stack_before = {},
      .stack_after = {3},
      .gas_before = 7,
      .gas_after = 5,
      .expected_state = RunState::kDone,
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTests, CALLDATASIZE_OutOfGas) {
  RunBasicTest({
      .code = "36",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, DISABLED_CALLDATASIZE_StackOverflow) {}

///////////////////////////////////////////////////////////
// CALLDATACOPY
TEST(InterpreterTests, CALLDATACOPY) {
  std::array<uint8_t, 4> input_data{0xA0, 0xA1, 0xA2, 0xA3};
  RunBasicTest({
      .code = "37",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 1,
      .expected_state = RunState::kDone,
      .memory_after = {0, 0, 0xA1, 0xA2, 0xA3},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTests, CALLDATACOPY_RetainMemory) {
  std::array<uint8_t, 4> input_data{0xA0, 0xA1, 0xA2, 0xA3};
  RunBasicTest({
      .code = "37",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .memory_before = {0xFF, 0xFF, 0, 0, 0, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0xA1, 0xA2, 0xA3, 0xFF, 0xFF, 0xFF},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTests, CALLDATACOPY_OutOfBounds) {
  std::array<uint8_t, 2> input_data{0xA0, 0xA1};
  RunBasicTest({
      .code = "37",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 1,
      .expected_state = RunState::kDone,
      .memory_after = {0, 0, 0xA1, 0, 0},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTests, CALLDATACOPY_OutOfGas_Static) {
  std::array<uint8_t, 2> input_data{0xA0, 0xA1};
  RunBasicTest({
      .code = "37",
      .stack_before = {3, 1, 2},
      .stack_after = {3, 1, 2},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTests, CALLDATACOPY_OutOfGas_Dynamic) {
  std::array<uint8_t, 2> input_data{0xA0, 0xA1};
  RunBasicTest({
      .code = "37",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 8,
      .gas_after = 5,
      .expected_state = RunState::kErrorGas,
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTests, CALLDATACOPY_StackError) {
  RunBasicTest({
      .code = "37",
      .stack_before = {3, 1},
      .stack_after = {3, 1},
      .gas_before = 100,
      .gas_after = 100,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// CODESIZE
TEST(InterpreterTests, CODESIZE) {
  RunBasicTest({
      .code = "3450345038",
      .stack_before = {},
      .stack_after = {5 + 1 /* for trailing STOP */},
      .gas_before = 100,
      .gas_after = 100 - 8 - 2,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, CODESIZE_OutOfGas) {
  RunBasicTest({
      .code = "3450345038",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 9,
      .gas_after = 9 - 8,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// CODECOPY
TEST(InterpreterTests, CODECOPY) {
  RunBasicTest({
      .code = "3450345039",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 100,
      .gas_after = 100 - 8 - 9,
      .expected_state = RunState::kDone,
      .memory_after = {0, 0, 0x50, 0x34, 0x50},
  });
}

TEST(InterpreterTests, CODECOPY_RetainMemory) {
  RunBasicTest({
      .code = "3450345039",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 100,
      .gas_after = 100 - 8 - 6,
      .expected_state = RunState::kDone,
      .memory_before = {0xFF, 0xFF, 0, 0, 0, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0x50, 0x34, 0x50, 0xFF, 0xFF, 0xFF},
  });
}

TEST(InterpreterTests, CODECOPY_OutOfGas_Static) {
  RunBasicTest({
      .code = "3450345039",
      .stack_before = {3, 1, 2},
      .stack_after = {3, 1, 2},
      .gas_before = 9,
      .gas_after = 9 - 8,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, CODECOPY_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "3450345039",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 13,
      .gas_after = 13 - 8 - 3,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, CODECOPY_StackError) {
  RunBasicTest({
      .code = "3450345039",
      .stack_before = {3, 1},
      .stack_after = {3, 1},
      .gas_before = 100,
      .gas_after = 92,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// GASPRICE
TEST(InterpreterTests, GASPRICE) {
  RunBasicTest({
      .code = "3A",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 7,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "3A",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 7,
      .gas_after = 5,
      .expected_state = RunState::kDone,
      .tx_context{.tx_gas_price = evmc::uint256be(0x42)},
  });
}

TEST(InterpreterTests, GASPRICE_OutOfGas) {
  RunBasicTest({
      .code = "3A",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, DISABLED_GASPRICE_StackOverflow) {}

///////////////////////////////////////////////////////////
// EXTCODESIZE
TEST(InterpreterTests, EXTCODESIZE) {
  RunBasicTest({
      .code = "3B",
      .stack_before = {0x42},
      .stack_after = {0},
      .gas_before = 3000,
      .gas_after = 400,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "3B",
      .stack_before = {0x42},
      .stack_after = {5 + 1 /* for trailing STOP */},
      .gas_before = 3000,
      .gas_after = 400,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0x42), {.code = ByteCodeStringToBinary("3450345039")}}},
  });
}

TEST(InterpreterTests, EXTCODESIZE_OutOfGas) {
  RunBasicTest({
      .code = "3B",
      .stack_before = {0x42},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorGas,
      .accounts{{evmc::address(0x42), {.code = ByteCodeStringToBinary("3450345039")}}},
  });
}

///////////////////////////////////////////////////////////
// EXTCODECOPY
TEST(InterpreterTests, EXTCODECOPY) {
  RunBasicTest({
      .code = "3C",
      .stack_before = {3, 1, 2, 0x42},
      .stack_after = {},
      .gas_before = 3000,
      .gas_after = 3000 - 2600 - 6,
      .expected_state = RunState::kDone,
      .memory_after = {0, 0, 0x50, 0x34, 0x50},
      .accounts{{evmc::address(0x42), {.code = ByteCodeStringToBinary("3450345039")}}},
  });
}

TEST(InterpreterTests, EXTCODECOPY_RetainMemory) {
  RunBasicTest({
      .code = "3C",
      .stack_before = {3, 1, 2, 0x42},
      .stack_after = {},
      .gas_before = 3000,
      .gas_after = 3000 - 2600 - 3,
      .expected_state = RunState::kDone,
      .memory_before = {0xFF, 0xFF, 0, 0, 0, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0x50, 0x34, 0x50, 0xFF, 0xFF, 0xFF},
      .accounts{{evmc::address(0x42), {.code = ByteCodeStringToBinary("3450345039")}}},
  });
}

TEST(InterpreterTests, EXTCODECOPY_OutOfGas) {
  RunBasicTest({
      .code = "3C",
      .stack_before = {3, 1, 2, 0x42},
      .stack_after = {},
      .gas_before = 2000,
      .gas_after = 2000,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, EXTCODECOPY_StackError) {
  RunBasicTest({
      .code = "3C",
      .stack_before = {3, 1, 2},
      .stack_after = {3, 1, 2},
      .gas_before = 3000,
      .gas_after = 3000,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// RETURNDATASIZE
TEST(InterpreterTests, RETURNDATASIZE) {
  RunBasicTest({
      .code = "3D",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "3D",
      .stack_before = {},
      .stack_after = {3},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .return_data = {0x01, 0x02, 0x03},
  });
}

TEST(InterpreterTests, RETURNDATASIZE_OutOfGas) {
  RunBasicTest({
      .code = "3D",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// RETURNDATACOPY
TEST(InterpreterTests, RETURNDATACOPY) {
  RunBasicTest({
      .code = "3E",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
      .memory_after{0x0A, 0x0B, 0x00, 0x00, 0x00, 0x0F},
  });

  RunBasicTest({
      .code = "3E",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
      .memory_after{0x0A, 0x0B, 0x02, 0x03, 0x04, 0x0F},
      .return_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTests, RETURNDATACOPY_Grow) {
  RunBasicTest({
      .code = "3E",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 1,
      .expected_state = RunState::kDone,
      .memory_after{0x00, 0x00, 0x02, 0x03, 0x04},
      .return_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTests, RETURNDATACOPY_OutOfGas_Static) {
  RunBasicTest({
      .code = "3E",
      .stack_before = {3, 1, 2},
      .stack_after = {3, 1, 2},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
      .return_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTests, RETURNDATACOPY_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "3E",
      .stack_before = {3, 1, 2},
      .stack_after = {},
      .gas_before = 8,
      .gas_after = 5,
      .expected_state = RunState::kErrorGas,
      .return_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTests, RETURNDATACOPY_StackError) {
  RunBasicTest({
      .code = "3E",
      .stack_before = {3, 1},
      .stack_after = {3, 1},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
      .return_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

///////////////////////////////////////////////////////////
// EXTCODEHASH
TEST(InterpreterTests, DISABLED_EXTCODEHASH) {
  RunBasicTest({
      .code = "3F",
      .stack_before = {0x42},
      // Words may be swapped here in the test, double check later!
      .stack_after = {uint256_t(0xc5d2460186f7233c, 0x927e7db2dcc703c0, 0xe500b653ca82273b, 0x7bfad8045d85a470)},
      .gas_before = 3000,
      .gas_after = 400,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0x42), {.code = ByteCodeStringToBinary("FFFFFFFF")}}},
  });
}

TEST(InterpreterTests, EXTCODEHASH_OutOfGas) {
  RunBasicTest({
      .code = "3F",
      .stack_before = {0x42},
      .stack_after = {},
      .gas_before = 100,
      .gas_after = 100,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, EXTCODEHASH_StackError) {
  RunBasicTest({
      .code = "3F",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 3000,
      .gas_after = 3000,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// BLOCKHASH
TEST(InterpreterTests, BLOCKHASH) {
  RunBasicTest({
      .code = "40",
      .stack_before = {7},
      .stack_after = {0},  // DummyHost always returns 0
      .gas_before = 40,
      .gas_after = 20,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, BLOCKHASH_OutOfGas) {
  RunBasicTest({
      .code = "40",
      .stack_before = {0},
      .stack_after = {0},
      .gas_before = 19,
      .gas_after = 19,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, BLOCKHASH_StackError) {
  RunBasicTest({
      .code = "40",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 40,
      .gas_after = 40,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// COINBASE
TEST(InterpreterTests, COINBASE) {
  RunBasicTest({
      .code = "41",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "41",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.block_coinbase = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, COINBASE_OutOfGas) {
  RunBasicTest({
      .code = "41",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// TIMESTAMP
TEST(InterpreterTests, TIMESTAMP) {
  RunBasicTest({
      .code = "42",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "42",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.block_timestamp = 0x42},
  });
}

TEST(InterpreterTests, TIMESTAMP_OutOfGas) {
  RunBasicTest({
      .code = "42",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// NUMBER
TEST(InterpreterTests, NUMBER) {
  RunBasicTest({
      .code = "43",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "43",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.block_number = 0x42},
  });
}

TEST(InterpreterTests, NUMBER_OutOfGas) {
  RunBasicTest({
      .code = "43",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// DIFFICULTY / PREVRANDAO
TEST(InterpreterTests, DIFFICULTY) {
  RunBasicTest({
      .code = "44",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "44",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.block_prev_randao = evmc::uint256be(0x42)},
  });
}

TEST(InterpreterTests, DIFFICULTY_OutOfGas) {
  RunBasicTest({
      .code = "44",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// GASLIMIT
TEST(InterpreterTests, GASLIMIT) {
  RunBasicTest({
      .code = "45",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "45",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.block_gas_limit = 0x42},
  });
}

TEST(InterpreterTests, GASLIMIT_OutOfGas) {
  RunBasicTest({
      .code = "45",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// CHAINID
TEST(InterpreterTests, CHAINID) {
  RunBasicTest({
      .code = "46",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "46",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.chain_id = evmc::uint256be(0x42)},
  });
}

TEST(InterpreterTests, CHAINID_OutOfGas) {
  RunBasicTest({
      .code = "46",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// SELFBALANCE
TEST(InterpreterTests, SELFBALANCE) {
  RunBasicTest({
      .code = "47",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 5,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "47",
      .stack_before = {},
      .stack_after = {1042},
      .gas_before = 10,
      .gas_after = 5,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0x42), {.balance = evmc::uint256be(1042)}}},
      .message{.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, SELFBALANCE_OutOfGas) {
  RunBasicTest({
      .code = "47",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 4,
      .gas_after = 4,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// BASEFEE
TEST(InterpreterTests, BASEFEE) {
  RunBasicTest({
      .code = "48",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });
  RunBasicTest({
      .code = "48",
      .stack_before = {},
      .stack_after = {0x42},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .tx_context{.block_base_fee = evmc::uint256be(0x42)},
  });
}

TEST(InterpreterTests, BASEFEE_OutOfGas) {
  RunBasicTest({
      .code = "48",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// POP
TEST(InterpreterTests, POP) {
  RunBasicTest({
      .code = "50",
      .stack_before = {3},
      .stack_after = {},
      .gas_before = 5,
      .gas_after = 3,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, POP_OutOfGas) {
  RunBasicTest({
      .code = "50",
      .stack_before = {3},
      .stack_after = {3},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, POP_StackError) {
  RunBasicTest({
      .code = "50",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 5,
      .gas_after = 5,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// MLOAD
TEST(InterpreterTests, MLOAD) {
  RunBasicTest({
      .code = "51",
      .stack_before = {0},
      .stack_after = {0xFF},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
      .memory_before{
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0xFF,  //
      },
      .memory_after{
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0xFF,  //
      },
  });

  RunBasicTest({
      .code = "51",
      .stack_before = {2},
      .stack_after = {0xFF},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
      .memory_before{
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0xFF,
      },
      .memory_after{
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0xFF,
      },
  });
}

TEST(InterpreterTests, MLOAD_Grow) {
  RunBasicTest({
      .code = "51",
      .stack_before = {32},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .memory_before{
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0xFF,  //
      },
      .memory_after{
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0xFF,  //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
      },
  });

  RunBasicTest({
      .code = "51",
      .stack_before = {1},
      .stack_after = {0xFF00},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .memory_before{
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0xFF,  //
      },
      .memory_after{
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0xFF,  //
          0,
      },
  });
}

TEST(InterpreterTests, MLOAD_RetainExisting) {
  RunBasicTest({
      .code = "51",
      .stack_before = {0},
      .stack_after = {0xFF},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
      .memory_before{
          0,    0, 0, 0, 0, 0, 0, 0,     //
          0,    0, 0, 0, 0, 0, 0, 0,     //
          0,    0, 0, 0, 0, 0, 0, 0,     //
          0,    0, 0, 0, 0, 0, 0, 0xFF,  //
          0xFF, 0, 0, 0, 0, 0, 0, 0,     //
          0xFF, 0, 0, 0, 0, 0, 0, 0,     //
      },
      .memory_after{
          0,    0, 0, 0, 0, 0, 0, 0,     //
          0,    0, 0, 0, 0, 0, 0, 0,     //
          0,    0, 0, 0, 0, 0, 0, 0,     //
          0,    0, 0, 0, 0, 0, 0, 0xFF,  //
          0xFF, 0, 0, 0, 0, 0, 0, 0,     //
          0xFF, 0, 0, 0, 0, 0, 0, 0,     //
      },
  });
}

TEST(InterpreterTests, MLOAD_OutOfGas_Static) {
  RunBasicTest({
      .code = "51",
      .stack_before = {1},
      .stack_after = {1},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
      .memory_before{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },
  });
}

TEST(InterpreterTests, MLOAD_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "51",
      .stack_before = {1},
      .stack_after = {},
      .gas_before = 5,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
      .memory_before{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },
  });
}

TEST(InterpreterTests, MLOAD_StackError) {
  RunBasicTest({
      .code = "51",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 100,
      .gas_after = 100,
      .expected_state = RunState::kErrorStack,
      .memory_before{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },
  });
}

///////////////////////////////////////////////////////////
// MSTORE
TEST(InterpreterTests, MSTORE) {
  RunBasicTest({
      .code = "52",
      .stack_before = {0xFF, 0},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
      .memory_before{
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0,     //
          0, 0, 0, 0, 0, 0, 0, 0xFF,  //
      },
  });
}

TEST(InterpreterTests, MSTORE_Grow) {
  RunBasicTest({
      .code = "52",
      .stack_before = {0xFF, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 1,
      .expected_state = RunState::kDone,
      .memory_after{
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0xFF,
      },
  });

  RunBasicTest({
      .code = "52",
      .stack_before = {0xFF, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .memory_before{
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0xFF,
      },
  });
}

TEST(InterpreterTests, MSTORE_RetainExisting) {
  RunBasicTest({
      .code = "52",
      .stack_before = {0xFF, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
      .memory_before{
          0,    0, 0xFF, 0, 0, 0, 0, 0,  //
          0,    0, 0,    0, 0, 0, 0, 0,  //
          0,    0, 0,    0, 0, 0, 0, 0,  //
          0,    0, 0,    0, 0, 0, 0, 0,  //
          0xFF, 0, 0,    0, 0, 0, 0, 0,  //
          0xFF, 0, 0,    0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0,    0,    0, 0, 0, 0, 0, 0,  //
          0,    0,    0, 0, 0, 0, 0, 0,  //
          0,    0,    0, 0, 0, 0, 0, 0,  //
          0,    0,    0, 0, 0, 0, 0, 0,  //
          0,    0xFF, 0, 0, 0, 0, 0, 0,  //
          0xFF, 0,    0, 0, 0, 0, 0, 0,  //
      },
  });
}

TEST(InterpreterTests, MSTORE_OutOfGas_Static) {
  RunBasicTest({
      .code = "52",
      .stack_before = {0xFF, 0},
      .stack_after = {0xFF, 0},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, MSTORE_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "52",
      .stack_before = {0xFF, 0},
      .stack_after = {},
      .gas_before = 5,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, MSTORE_StackError) {
  RunBasicTest({
      .code = "52",
      .stack_before = {0xFF},
      .stack_after = {0xFF},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// MSTORE8
TEST(InterpreterTests, MSTORE8) {
  RunBasicTest({
      .code = "53",
      .stack_before = {0xBB, 1},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
      .memory_before = {0xAA, 0, 0xCC, 0xDD, 0, 0, 0, 0},
      .memory_after = {0xAA, 0xBB, 0xCC, 0xDD, 0, 0, 0, 0},
  });
}

TEST(InterpreterTests, MSTORE8_Grow) {
  RunBasicTest({
      .code = "53",
      .stack_before = {0xFF, 32},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
      .memory_before = {0, 0, 0, 0, 0, 0, 0, 0},
      .memory_after{
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0xFF,                       //
      },
  });
}

TEST(InterpreterTests, MSTORE8_RetainExisting) {
  RunBasicTest({
      .code = "53",
      .stack_before = {0xFF, 2},
      .stack_after = {},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
      .memory_before{
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0,    0, 0xFF, 0, 0, 0, 0, 0,  //
          0,    0, 0,    0, 0, 0, 0, 0,  //
          0,    0, 0,    0, 0, 0, 0, 0,  //
          0,    0, 0,    0, 0, 0, 0, 0,  //
          0xFF, 0, 0,    0, 0, 0, 0, 0,  //
          0xFF, 0, 0,    0, 0, 0, 0, 0,  //
      },
  });
}

TEST(InterpreterTests, MSTORE8_OutOfGas_Static) {
  RunBasicTest({
      .code = "53",
      .stack_before = {0xFF, 0},
      .stack_after = {0xFF, 0},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, MSTORE8_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "53",
      .stack_before = {0xFF, 0},
      .stack_after = {},
      .gas_before = 5,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, MSTORE8_StackError) {
  RunBasicTest({
      .code = "53",
      .stack_before = {0xFF},
      .stack_after = {0xFF},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// SLOAD
TEST(InterpreterTests, SLOAD) {
  RunBasicTest({
      .code = "54",
      .stack_before = {8965},
      .stack_after = {0xFF},
      .gas_before = 2200,
      .gas_after = 100,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
      .accounts_after{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
  });
  RunBasicTest({
      .code = "54",
      .stack_before = {8964},
      .stack_after = {0},
      .gas_before = 2200,
      .gas_after = 100,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
      .accounts_after{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
  });
}

TEST(InterpreterTests, SLOAD_OutOfGas) {
  RunBasicTest({
      .code = "54",
      .stack_before = {8965},
      .stack_after = {},
      .gas_before = 2000,
      .gas_after = 2000,
      .expected_state = RunState::kErrorGas,
      .accounts{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
      .accounts_after{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
  });
}

TEST(InterpreterTests, SLOAD_StackError) {
  RunBasicTest({
      .code = "54",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 200,
      .gas_after = 200,
      .expected_state = RunState::kErrorStack,
      .accounts{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
      .accounts_after{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
  });
}

///////////////////////////////////////////////////////////
// SSTORE
TEST(InterpreterTests, SSTORE) {
  RunBasicTest({
      .code = "55",
      .stack_before = {0xFF, 8965},
      .stack_after = {},
      .gas_before = 2400,
      .gas_after = 200,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0), {}}},
      .accounts_after{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
  });

  RunBasicTest({
      .code = "55",
      .stack_before = {0xEE, 8965},
      .stack_after = {},
      .gas_before = 2400,
      .gas_after = 200,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xFF)}}}}},
      .accounts_after{{evmc::address(0), {.storage{{evmc::bytes32(8965), evmc::bytes32(0xEE)}}}}},
  });
}

TEST(InterpreterTests, SSTORE_RetainStorage) {
  RunBasicTest({
      .code = "55",
      .stack_before = {0xFF, 8965},
      .stack_after = {},
      .gas_before = 2400,
      .gas_after = 200,
      .expected_state = RunState::kDone,
      .accounts{{evmc::address(0), {.storage{{evmc::bytes32(8964), evmc::bytes32(0xEE)}}}}},
      .accounts_after{{evmc::address(0),
                       {.storage{
                           {evmc::bytes32(8964), evmc::bytes32(0xEE)},
                           {evmc::bytes32(8965), evmc::bytes32(0xFF)},
                       }}}},
  });
}

TEST(InterpreterTests, SSTORE_OutOfGas) {
  RunBasicTest({
      .code = "55",
      .stack_before = {0xFF, 8965},
      .stack_after = {},
      .gas_before = 2150,
      .gas_after = 2150,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SSTORE_StackError) {
  RunBasicTest({
      .code = "55",
      .stack_before = {0xFF},
      .stack_after = {0xFF},
      .gas_before = 3000,
      .gas_after = 3000,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// PC
TEST(InterpreterTests, PC) {
  RunBasicTest({
      .code = "585858",
      .stack_before = {},
      .stack_after = {0, 1, 2},
      .gas_before = 10,
      .gas_after = 4,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, PC_OutOfGas) {
  RunBasicTest({
      .code = "58",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// MSIZE
TEST(InterpreterTests, MSIZE) {
  RunBasicTest({
      .code = "59",
      .stack_before = {},
      .stack_after = {0},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "59",
      .stack_before = {},
      .stack_after = {32},
      .gas_before = 10,
      .gas_after = 8,
      .expected_state = RunState::kDone,
      .memory_before{
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
      },
      .memory_after{
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
          0, 0, 0, 0, 0, 0, 0, 0,  //
      },
  });
}

TEST(InterpreterTests, MSIZE_OutOfGas) {
  RunBasicTest({
      .code = "59",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// GAS
TEST(InterpreterTests, GAS) {
  RunBasicTest({
      .code = "5A",
      .stack_before = {},
      .stack_after = {98},
      .gas_before = 100,
      .gas_after = 98,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, GAS_OutOfGas) {
  RunBasicTest({
      .code = "5A",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

///////////////////////////////////////////////////////////
// PUSH
TEST(InterpreterTests, PUSH) {
  RunBasicTest({
      .code = "63FFFFFFFF",
      .stack_before = {},
      .stack_after = {0xFFFFFFFF},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "73FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFAA",
      .stack_before = {},
      .stack_after = {uint256_t(0xFFFFFFFFFFFFFFAA, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFF)},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, PUSH_OutOfGas) {
  RunBasicTest({
      .code = "63FFFFFFFF",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 2,
      .gas_after = 2,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, PUSH_OutOfBytes) {
  RunBasicTest({
      .code = "63FFFF",  // PUSH4, but only 2 bytes (+ added stop byte)
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorOpcode,
  });
}

TEST(InterpreterTests, DISABLED_PUSH_StackOverflow) {}

///////////////////////////////////////////////////////////
// DUP
TEST(InterpreterTests, DUP) {
  RunBasicTest({
      .code = "83",
      .stack_before = {4, 3, 2, 1},
      .stack_after = {4, 3, 2, 1, 4},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "8E",
      .stack_before = {16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
      .stack_after = {16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 15},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, DUP_OutOfGas) {
  RunBasicTest({
      .code = "83",
      .stack_before = {4, 3, 2, 1},
      .stack_after = {4, 3, 2, 1},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, DUP_StackError) {
  RunBasicTest({
      .code = "83",
      .stack_before = {3, 2, 1},
      .stack_after = {3, 2, 1},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
  });
}

TEST(InterpreterTests, DISABLED_DUP_StackOverflow) {}

///////////////////////////////////////////////////////////
// SWAP
TEST(InterpreterTests, SWAP) {
  RunBasicTest({
      .code = "93",
      .stack_before = {5, 4, 3, 2, 1},
      .stack_after = {1, 4, 3, 2, 5},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
  });

  RunBasicTest({
      .code = "9f",
      .stack_before = {18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
      .stack_after = {18, 1, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 17},
      .gas_before = 10,
      .gas_after = 7,
      .expected_state = RunState::kDone,
  });
}

TEST(InterpreterTests, SWAP_OutOfGas) {
  RunBasicTest({
      .code = "93",
      .stack_before = {5, 4, 3, 2, 1},
      .stack_after = {5, 4, 3, 2, 1},
      .gas_before = 1,
      .gas_after = 1,
      .expected_state = RunState::kErrorGas,
  });
}

TEST(InterpreterTests, SWAP_StackError) {
  RunBasicTest({
      .code = "93",
      .stack_before = {4, 3, 2, 1},
      .stack_after = {4, 3, 2, 1},
      .gas_before = 10,
      .gas_after = 10,
      .expected_state = RunState::kErrorStack,
  });
}

///////////////////////////////////////////////////////////
// LOG
TEST(InterpreterTests, LOG0) {
  RunBasicTest({
      .code = "A0",
      .stack_before = {3, 1},
      .stack_after = {},
      .gas_before = 400,
      .gas_after = 1,
      .expected_state = RunState::kDone,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .accounts_after{{evmc::address(0x42),
                       {.logs{{
                           .data = {0x0B, 0x0C, 0x0D},
                           .topics = {},
                       }}}}},
      .message{.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, LOG3) {
  RunBasicTest({
      .code = "A3",
      .stack_before = {0xF3, 0xF2, 0xF1, 3, 1},
      .stack_after = {},
      .gas_before = 1524,
      .gas_after = 0,
      .expected_state = RunState::kDone,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .accounts_after{{evmc::address(0x42),
                       {.logs{{
                           .data = {0x0B, 0x0C, 0x0D},
                           .topics = {evmc::bytes32(0xF1), evmc::bytes32(0xF2), evmc::bytes32(0xF3)},
                       }}}}},
      .message{.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, LOG0_GrowMemory) {
  RunBasicTest({
      .code = "A0",
      .stack_before = {5, 1},
      .stack_after = {},
      .gas_before = 420,
      .gas_after = 5,
      .expected_state = RunState::kDone,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D, 0x00, 0x00},
      .accounts_after{{evmc::address(0x42),
                       {.logs{{
                           .data = {0x0B, 0x0C, 0x0D, 0x00, 0x00},
                           .topics = {},
                       }}}}},
      .message{.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, LOG0_OutOfGas_Static) {
  RunBasicTest({
      .code = "A0",
      .stack_before = {5, 1},
      .stack_after = {5, 1},
      .gas_before = 350,
      .gas_after = 350,
      .expected_state = RunState::kErrorGas,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D},
      .accounts_after{{evmc::address(0x42), {.logs{}}}},
      .message{.recipient = evmc::address(0x42)},
  });
}
TEST(InterpreterTests, LOG0_OutOfGas_Dynamic) {
  RunBasicTest({
      .code = "A0",
      .stack_before = {5, 1},
      .stack_after = {},
      .gas_before = 400,
      .gas_after = 25,
      .expected_state = RunState::kErrorGas,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D},
      .accounts_after{{evmc::address(0x42), {.logs{}}}},
      .message{.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTests, LOG0_StackError) {
  RunBasicTest({
      .code = "A0",
      .stack_before = {5},
      .stack_after = {5},
      .gas_before = 1000,
      .gas_after = 1000,
      .expected_state = RunState::kErrorStack,
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D},
      .accounts_after{{evmc::address(0x42), {.logs{}}}},
      .message{.recipient = evmc::address(0x42)},
  });
}

///////////////////////////////////////////////////////////
// INVALID
TEST(InterpreterTests, INVALID) {
  RunBasicTest({
      .code = "FE",
      .gas_before = 100,
      .gas_after = 0,
      .expected_state = RunState::kInvalid,
  });
}

///////////////////////////////////////////////////////////
// SELFDESTRUCT
TEST(InterpreterTests, SELFDESTRUCT) {
  RunBasicTest({
      .code = "FF",
      .stack_before = {0x42},
      .stack_after = {},
      .gas_before = 5000,
      .gas_after = 0,
      .expected_state = RunState::kDone,
      .accounts{
          {evmc::address(0x00), {.balance = evmc::uint256be(50)}},
          {evmc::address(0x42), {.balance = evmc::uint256be(0)}},
      },
      .accounts_after{
          {evmc::address(0x00), {.dead = true, .balance = evmc::uint256be(0)}},
          {evmc::address(0x42), {.balance = evmc::uint256be(50)}},
      },
  });
}

TEST(InterpreterTests, SELFDESTRUCT_OutOfGas) {
  RunBasicTest({
      .code = "FF",
      .stack_before = {0x42},
      .stack_after = {0x42},
      .gas_before = 4000,
      .gas_after = 4000,
      .expected_state = RunState::kErrorGas,
      .accounts{
          {evmc::address(0x00), {.balance = evmc::uint256be(50)}},
          {evmc::address(0x42), {.balance = evmc::uint256be(0)}},
      },
      .accounts_after{
          {evmc::address(0x00), {.balance = evmc::uint256be(50)}},
          {evmc::address(0x42), {.balance = evmc::uint256be(0)}},
      },
  });
}

TEST(InterpreterTests, SELFDESTRUCT_StackError) {
  RunBasicTest({
      .code = "FF",
      .stack_before = {},
      .stack_after = {},
      .gas_before = 5000,
      .gas_after = 5000,
      .expected_state = RunState::kErrorStack,
      .accounts{
          {evmc::address(0x00), {.balance = evmc::uint256be(50)}},
          {evmc::address(0x42), {.balance = evmc::uint256be(0)}},
      },
      .accounts_after{
          {evmc::address(0x00), {.balance = evmc::uint256be(50)}},
          {evmc::address(0x42), {.balance = evmc::uint256be(0)}},
      },
  });
}

///////////////////////////////////////////////////////////

TEST(InterpreterTests, EnsureChecks) {
#if !PERFORM_GAS_CHECKS || !PERFORM_STACK_CHECKS
  FAIL();
#endif
}

TEST(InterpreterTests, InvalidJump) {
  RunBasicTest({
      .code = "63"   // 0: PUSH4
              "00"   // 1:
              "00"   // 2:
              "5B"   // 3: jumpdest, but data
              "00"   // 4:
              "60"   // 5: PUSH1
              "03"   // 6: jump target (into data)
              "56",  // 7: jump
      .stack_before = {},
      .stack_after = {23296},
      .gas_before = 5000,
      .gas_after = 4986,
      .expected_state = RunState::kErrorJump,
  });
}

}  // namespace tosca::evmzero

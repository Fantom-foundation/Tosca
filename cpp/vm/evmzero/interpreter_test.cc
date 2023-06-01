#include "vm/evmzero/interpreter.h"

#include <gtest/gtest.h>

#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero {
namespace {

struct InterpreterTestDescription {
  std::vector<uint8_t> code;

  RunState state_after = RunState::kDone;

  uint64_t gas_before = 0;
  uint64_t gas_after = 0;

  Stack stack_before;
  Stack stack_after;
};

void RunInterpreterTest(const InterpreterTestDescription& desc) {
  internal::Context ctx{
      .gas = desc.gas_before,
      .code = desc.code,
      .stack = desc.stack_before,
  };

  // Adding a final STOP byte here so we don't have to add it in every test!
  ctx.code.push_back(op::STOP);

  internal::RunInterpreter(ctx);

  ASSERT_EQ(ctx.state, desc.state_after);

  if (ctx.state == RunState::kDone) {
    EXPECT_EQ(ctx.gas, desc.gas_after);
    EXPECT_EQ(ctx.stack, desc.stack_after);
  }
}

TEST(InterpreterTest, NoStopByte) {
  internal::Context ctx{
      .code = {op::ADD},
      .stack = {1, 2},
  };

  internal::RunInterpreter(ctx);

  EXPECT_EQ(ctx.pc, 1);
  EXPECT_EQ(ctx.state, RunState::kErrorOpcode);
}

///////////////////////////////////////////////////////////
// STOP
TEST(InterpreterTest, STOP) {
  RunInterpreterTest({
      .code = {op::STOP},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 7,
  });
}

///////////////////////////////////////////////////////////
// ADD
TEST(InterpreterTest, ADD) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {1, 6},
      .stack_after = {7},
  });
}

TEST(InterpreterTest, ADD_Overflow) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {1, kUint256Max},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, ADD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, kUint256Max},
  });
}

TEST(InterpreterTest, ADD_StackError) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
  });
}

///////////////////////////////////////////////////////////
// MUL
TEST(InterpreterTest, MUL) {
  RunInterpreterTest({
      .code = {op::MUL},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {10, 0},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::MUL},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {5, 4},
      .stack_after = {20},
  });
}

TEST(InterpreterTest, MUL_Overflow) {
  RunInterpreterTest({
      .code = {op::MUL},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {kUint256Max, 2},
      .stack_after = {kUint256Max - 1},
  });
}

TEST(InterpreterTest, MUL_OutOfGas) {
  RunInterpreterTest({
      .code = {op::MUL},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, MUL_StackError) {
  RunInterpreterTest({
      .code = {op::MUL},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// SUB
TEST(InterpreterTest, SUB) {
  RunInterpreterTest({
      .code = {op::SUB},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::SUB},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {5, 10},
      .stack_after = {5},
  });
}

TEST(InterpreterTest, SUB_Underflow) {
  RunInterpreterTest({
      .code = {op::SUB},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {1, 0},
      .stack_after = {kUint256Max},
  });
}

TEST(InterpreterTest, SUB_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SUB},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, SUB_StackError) {
  RunInterpreterTest({
      .code = {op::SUB},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// DIV
TEST(InterpreterTest, DIV) {
  RunInterpreterTest({
      .code = {op::DIV},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {10, 10},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::DIV},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {2, 1},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, DIV_ByZero) {
  RunInterpreterTest({
      .code = {op::DIV},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {0, 10},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, DIV_OutOfGas) {
  RunInterpreterTest({
      .code = {op::DIV},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, DIV_StackError) {
  RunInterpreterTest({
      .code = {op::DIV},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// SDIV
TEST(InterpreterTest, SDIV) {
  RunInterpreterTest({
      .code = {op::SDIV},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {10, 10},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::SDIV},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {kUint256Max, kUint256Max - 1},
      .stack_after = {2},
  });
}

TEST(InterpreterTest, SDIV_ByZero) {
  RunInterpreterTest({
      .code = {op::SDIV},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {0, 10},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, SDIV_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SDIV},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, SDIV_StackError) {
  RunInterpreterTest({
      .code = {op::SDIV},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// MOD
TEST(InterpreterTest, MOD) {
  RunInterpreterTest({
      .code = {op::MOD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {3, 10},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::MOD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {5, 17},
      .stack_after = {2},
  });
}

TEST(InterpreterTest, MOD_ByZero) {
  RunInterpreterTest({
      .code = {op::MOD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {0, 10},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, MOD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::MOD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, MOD_StackError) {
  RunInterpreterTest({
      .code = {op::MOD},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// SMOD
TEST(InterpreterTest, SMOD) {
  RunInterpreterTest({
      .code = {op::SMOD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {3, 10},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::SMOD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {kUint256Max - 2, kUint256Max - 7},
      .stack_after = {kUint256Max - 1},
  });
}

TEST(InterpreterTest, SMOD_ByZero) {
  RunInterpreterTest({
      .code = {op::SMOD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 2,
      .stack_before = {0, 10},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, SMOD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SMOD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, SMOD_StackError) {
  RunInterpreterTest({
      .code = {op::SMOD},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// ADDMOD
TEST(InterpreterTest, ADDMOD) {
  RunInterpreterTest({
      .code = {op::ADDMOD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 2,
      .stack_before = {8, 10, 10},
      .stack_after = {4},
  });

  RunInterpreterTest({
      .code = {op::ADDMOD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 2,
      .stack_before = {2, 2, kUint256Max},
      .stack_after = {1},
  });
}

TEST(InterpreterTest, ADDMOD_ByZero) {
  RunInterpreterTest({
      .code = {op::ADDMOD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 2,
      .stack_before = {0, 10, 10},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, ADDMOD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::ADDMOD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 10, 10},
  });
}

TEST(InterpreterTest, ADDMOD_StackError) {
  RunInterpreterTest({
      .code = {op::ADDMOD},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {1, 2},
  });
}

///////////////////////////////////////////////////////////
// MULMOD
TEST(InterpreterTest, MULMOD) {
  RunInterpreterTest({
      .code = {op::MULMOD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 2,
      .stack_before = {8, 10, 10},
      .stack_after = {4},
  });

  RunInterpreterTest({
      .code = {op::MULMOD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 2,
      .stack_before = {12, kUint256Max, kUint256Max},
      .stack_after = {9},
  });
}

TEST(InterpreterTest, MULMOD_ByZero) {
  RunInterpreterTest({
      .code = {op::MULMOD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 2,
      .stack_before = {0, 10, 10},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, MULMOD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::MULMOD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {8, 10, 10},
  });
}

TEST(InterpreterTest, MULMOD_StackError) {
  RunInterpreterTest({
      .code = {op::MULMOD},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {1, 2},
  });
}

///////////////////////////////////////////////////////////
// EXP
TEST(InterpreterTest, EXP) {
  RunInterpreterTest({
      .code = {op::EXP},
      .state_after = RunState::kDone,
      .gas_before = 200,
      .gas_after = 140,
      .stack_before = {2, 10},
      .stack_after = {100},
  });

  RunInterpreterTest({
      .code = {op::EXP},
      .state_after = RunState::kDone,
      .gas_before = 200,
      .gas_after = 90,
      .stack_before = {4747, 1},
      .stack_after = {1},
  });
}

TEST(InterpreterTest, EXP_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::EXP},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {2, 40000},
  });
}

TEST(InterpreterTest, EXP_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::EXP},
      .state_after = RunState::kErrorGas,
      .gas_before = 12,
      .stack_before = {2, 40000},
  });
}

TEST(InterpreterTest, EXP_StackError) {
  RunInterpreterTest({
      .code = {op::EXP},
      .state_after = RunState::kErrorStack,
      .gas_before = 200,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// SIGNEXTEND
TEST(InterpreterTest, SIGNEXTEND) {
  RunInterpreterTest({
      .code = {op::SIGNEXTEND},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 5,
      .stack_before = {0xFF, 0},
      .stack_after = {kUint256Max},
  });

  RunInterpreterTest({
      .code = {op::SIGNEXTEND},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 5,
      .stack_before = {0x7F, 0},
      .stack_after = {0x7F},
  });

  RunInterpreterTest({
      .code = {op::SIGNEXTEND},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 5,
      .stack_before = {0xFF7F, 0},
      .stack_after = {0x7F},
  });

  RunInterpreterTest({
      .code = {op::SIGNEXTEND},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 5,
      .stack_before = {0xFF7F, 1},
      .stack_after = {kUint256Max - 0x80},
  });
}

TEST(InterpreterTest, SIGNEXTEND_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SIGNEXTEND},
      .state_after = RunState::kErrorGas,
      .gas_before = 4,
      .stack_before = {0xFF, 0},
  });
}

TEST(InterpreterTest, SIGNEXTEND_StackError) {
  RunInterpreterTest({
      .code = {op::SIGNEXTEND},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// LT
TEST(InterpreterTest, LT) {
  RunInterpreterTest({
      .code = {op::LT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::LT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {9, 10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::LT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 9},
      .stack_after = {1},
  });
}

TEST(InterpreterTest, LT_OutOfGas) {
  RunInterpreterTest({
      .code = {op::LT},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, LT_StackError) {
  RunInterpreterTest({
      .code = {op::LT},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// GT
TEST(InterpreterTest, GT) {
  RunInterpreterTest({
      .code = {op::GT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::GT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {9, 10},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::GT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 9},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, GT_OutOfGas) {
  RunInterpreterTest({
      .code = {op::GT},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, GT_StackError) {
  RunInterpreterTest({
      .code = {op::GT},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// SLT
TEST(InterpreterTest, SLT) {
  RunInterpreterTest({
      .code = {op::SLT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::SLT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {9, 10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::SLT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 9},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::SLT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0, kUint256Max},
      .stack_after = {1},
  });
}

TEST(InterpreterTest, SLT_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SLT},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, SLT_StackError) {
  RunInterpreterTest({
      .code = {op::SLT},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// SGT
TEST(InterpreterTest, SGT) {
  RunInterpreterTest({
      .code = {op::SGT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::SGT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {9, 10},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::SGT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 9},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::SGT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {kUint256Max, 0},
      .stack_after = {1},
  });
}

TEST(InterpreterTest, SGT_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SGT},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, SGT_StackError) {
  RunInterpreterTest({
      .code = {op::SGT},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// EQ
TEST(InterpreterTest, EQ) {
  RunInterpreterTest({
      .code = {op::EQ},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10, 10},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::EQ},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {9, 10},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, EQ_OutOfGas) {
  RunInterpreterTest({
      .code = {op::EQ},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 6},
  });
}

TEST(InterpreterTest, EQ_StackError) {
  RunInterpreterTest({
      .code = {op::EQ},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

///////////////////////////////////////////////////////////
// ISZERO
TEST(InterpreterTest, ISZERO) {
  RunInterpreterTest({
      .code = {op::ISZERO},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {10},
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::ISZERO},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0},
      .stack_after = {1},
  });
}

TEST(InterpreterTest, ISZERO_OutOfGas) {
  RunInterpreterTest({
      .code = {op::ISZERO},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1},
  });
}

TEST(InterpreterTest, ISZERO_StackError) {
  RunInterpreterTest({
      .code = {op::ISZERO},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {},
  });
}

///////////////////////////////////////////////////////////
// AND
TEST(InterpreterTest, AND) {
  RunInterpreterTest({
      .code = {op::AND},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xF, 0xF},
      .stack_after = {0xF},
  });

  RunInterpreterTest({
      .code = {op::AND},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0, 0xFF},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, AND_OutOfGas) {
  RunInterpreterTest({
      .code = {op::AND},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {0xF, 0xF},
  });
}

TEST(InterpreterTest, AND_StackError) {
  RunInterpreterTest({
      .code = {op::AND},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// OR
TEST(InterpreterTest, OR) {
  RunInterpreterTest({
      .code = {op::OR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xF, 0xF0},
      .stack_after = {0xFF},
  });

  RunInterpreterTest({
      .code = {op::OR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xFF, 0xFF},
      .stack_after = {0xFF},
  });
}

TEST(InterpreterTest, OR_OutOfGas) {
  RunInterpreterTest({
      .code = {op::OR},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {0xF, 0xF0},
  });
}

TEST(InterpreterTest, OR_StackError) {
  RunInterpreterTest({
      .code = {op::OR},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// XOR
TEST(InterpreterTest, XOR) {
  RunInterpreterTest({
      .code = {op::XOR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xF, 0xF0},
      .stack_after = {0xFF},
  });

  RunInterpreterTest({
      .code = {op::XOR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xFF, 0xFF},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, XOR_OutOfGas) {
  RunInterpreterTest({
      .code = {op::XOR},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {0xF, 0xF0},
  });
}

TEST(InterpreterTest, XOR_StackError) {
  RunInterpreterTest({
      .code = {op::XOR},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// NOT
TEST(InterpreterTest, NOT) {
  RunInterpreterTest({
      .code = {op::NOT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0},
      .stack_after = {kUint256Max},
  });

  RunInterpreterTest({
      .code = {op::NOT},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xFF},
      .stack_after = {kUint256Max - 0xFF},
  });
}

TEST(InterpreterTest, NOT_OutOfGas) {
  RunInterpreterTest({
      .code = {op::NOT},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {0},
  });
}

TEST(InterpreterTest, NOT_StackError) {
  RunInterpreterTest({
      .code = {op::NOT},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {},
  });
}

///////////////////////////////////////////////////////////
// BYTE
TEST(InterpreterTest, BYTE) {
  RunInterpreterTest({
      .code = {op::BYTE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xFF, 31},
      .stack_after = {0xFF},
  });

  RunInterpreterTest({
      .code = {op::BYTE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xFF00, 30},
      .stack_after = {0xFF},
  });
}

TEST(InterpreterTest, BYTE_OutOfRange) {
  RunInterpreterTest({
      .code = {op::BYTE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {kUint256Max, 32},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, BYTE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::BYTE},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {0xFF, 31},
  });
}

TEST(InterpreterTest, BYTE_StackError) {
  RunInterpreterTest({
      .code = {op::BYTE},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// SHL
TEST(InterpreterTest, SHL) {
  RunInterpreterTest({
      .code = {op::SHL},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {1, 1},
      .stack_after = {2},
  });

  RunInterpreterTest({
      .code = {op::SHL},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {uint256_t{0xFF} << 248, 4},
      .stack_after = {uint256_t{0xF} << 252},
  });

  RunInterpreterTest({
      .code = {op::SHL},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {7, 256},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, SHL_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SHL},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, 1},
  });
}

TEST(InterpreterTest, SHL_StackError) {
  RunInterpreterTest({
      .code = {op::SHL},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// SHR
TEST(InterpreterTest, SHR) {
  RunInterpreterTest({
      .code = {op::SHR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {2, 1},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::SHR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0xFF, 4},
      .stack_after = {0xF},
  });

  RunInterpreterTest({
      .code = {op::SHR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {kUint256Max, 256},
      .stack_after = {0},
  });
}

TEST(InterpreterTest, SHR_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SHR},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {2, 1},
  });
}

TEST(InterpreterTest, SHR_StackError) {
  RunInterpreterTest({
      .code = {op::SHR},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {2},
  });
}

///////////////////////////////////////////////////////////
// SAR
TEST(InterpreterTest, SAR) {
  RunInterpreterTest({
      .code = {op::SAR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {2, 1},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::SAR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {kUint256Max - 0xF, 4},
      .stack_after = {kUint256Max},
  });
}

TEST(InterpreterTest, SAR_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SAR},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .gas_after = 2,
      .stack_before = {2, 1},
      .stack_after = {2, 1},
  });
}

TEST(InterpreterTest, SAR_StackError) {
  RunInterpreterTest({
      .code = {op::SAR},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
      .stack_before = {1},
  });
}

}  // namespace
}  // namespace tosca::evmzero

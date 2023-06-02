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

  Memory memory_before;
  Memory memory_after;

  std::vector<uint8_t> last_call_data;
  std::vector<uint8_t> return_data;
};

void RunInterpreterTest(const InterpreterTestDescription& desc) {
  internal::Context ctx{
      .gas = desc.gas_before,
      .code = desc.code,
      .return_data = desc.last_call_data,
      .memory = desc.memory_before,
      .stack = desc.stack_before,
  };

  // Adding a final STOP byte here so we don't have to add it in every test!
  ctx.code.push_back(op::STOP);

  internal::RunInterpreter(ctx);

  ASSERT_EQ(ctx.state, desc.state_after);

  if (ctx.state == RunState::kDone || ctx.state == RunState::kRevert) {
    EXPECT_EQ(ctx.gas, desc.gas_after);
    EXPECT_EQ(ctx.stack, desc.stack_after);
    EXPECT_EQ(ctx.memory, desc.memory_after);
    if (!desc.return_data.empty()) {
      EXPECT_EQ(ctx.return_data, desc.return_data);
    }
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

///////////////////////////////////////////////////////////
// SHA3
TEST(InterpreterTest, SHA3) {
  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 64,
      .stack_before = {4, 0},
      .stack_after = {uint256_t(0x79A1BC8F0BB2C238, 0x9522D0CF0F73282C, 0x46EF02C2223570DA, 0x29045A592007D0C2)},
      .memory_before = {0xFF, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0xFF, 0xFF},
  });
}

TEST(InterpreterTest, SHA3_GrowMemory) {
  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 61,
      .stack_before = {4, 0},
      .stack_after = {uint256_t(0x64633A4ACBD3244C, 0xF7685EBD40E852B1, 0x55364C7B4BBF0BB7, 0xE8E77626586F73B9)},
      .memory_after = {0x00, 0x00, 0x00, 0x00},
  });
}

TEST(InterpreterTest, SHA3_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kErrorGas,
      .gas_before = 10,
      .stack_before = {4, 0},
  });
}

TEST(InterpreterTest, SHA3_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kErrorGas,
      .gas_before = 32,
      .stack_before = {4, 0},
  });
}

TEST(InterpreterTest, SHA3_StackError) {
  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kErrorStack,
      .gas_before = 100,
      .stack_before = {4},
  });
}

///////////////////////////////////////////////////////////
// CODESIZE
TEST(InterpreterTest, CODESIZE) {
  RunInterpreterTest({
      .code = {op::PUSH1, 0,  //
               op::POP,       //
               op::PUSH1, 0,  //
               op::POP,       //
               op::CODESIZE},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 100 - 10 - 2,
      .stack_after = {7 + 1 /* for trailing STOP byte */},
  });
}

TEST(InterpreterTest, CODESIZE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::PUSH1, 0,  //
               op::POP,       //
               op::PUSH1, 0,  //
               op::POP,       //
               op::CODESIZE},
      .state_after = RunState::kErrorGas,
      .gas_before = 11,
      .stack_before = {},
  });
}

///////////////////////////////////////////////////////////
// CODECOPY
TEST(InterpreterTest, CODECOPY) {
  RunInterpreterTest({
      .code = {op::PUSH1, 23,  //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 100 - 10 - 9,
      .stack_before = {3, 1, 2},
      .memory_after = {0, 0, 23, op::POP, op::PUSH1},
  });
}

TEST(InterpreterTest, CODECOPY_OutOfBytes) {
  RunInterpreterTest({
      .code = {op::PUSH1, 23,  //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 100 - 10 - 9,
      .stack_before = {8, 1, 2},
      .memory_after = {0, 0, 23, op::POP, op::PUSH1, 42, op::POP, op::CODECOPY,  //
                       op::STOP, 0},
  });
}

TEST(InterpreterTest, CODECOPY_OutOfBytes_WriteZero) {
  RunInterpreterTest({
      .code = {op::PUSH1, 0,   //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 84,
      .stack_before = {8, 1, 2},
      .memory_before = {0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,  //
                        0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0, op::POP, op::PUSH1, 42, op::POP, op::CODECOPY,  //
                       op::STOP, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
  });
}

TEST(InterpreterTest, CODECOPY_RetainMemory) {
  RunInterpreterTest({
      .code = {op::PUSH1, 23,  //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 100 - 10 - 6,
      .stack_before = {3, 1, 2},
      .memory_before = {0xFF, 0xFF, 0, 0, 0, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 23, op::POP, op::PUSH1, 0xFF, 0xFF, 0xFF},
  });
}

TEST(InterpreterTest, CODECOPY_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::PUSH1, 0,   //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 11,
      .stack_before = {3, 1, 2},
  });
}

TEST(InterpreterTest, CODECOPY_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::PUSH1, 0,   //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 15,
      .stack_before = {3, 1, 2},
  });
}

TEST(InterpreterTest, CODECOPY_StackError) {
  RunInterpreterTest({
      .code = {op::PUSH1, 0,   //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kErrorStack,
      .gas_before = 100,
      .stack_before = {3, 1},
  });
}

///////////////////////////////////////////////////////////
// RETURNDATASIZE
TEST(InterpreterTest, RETURNDATASIZE) {
  RunInterpreterTest({
      .code = {op::RETURNDATASIZE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::RETURNDATASIZE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {3},
      .last_call_data = {0x01, 0x02, 0x03},
  });
}

TEST(InterpreterTest, RETURNDATASIZE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::RETURNDATASIZE},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
      .gas_after = 1,
  });
}

///////////////////////////////////////////////////////////
// RETURNDATACOPY
TEST(InterpreterTest, RETURNDATACOPY) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {3, 1, 2},
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
      .memory_after{0x0A, 0x0B, 0x00, 0x00, 0x00, 0x0F},
  });

  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {3, 1, 2},
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
      .memory_after{0x0A, 0x0B, 0x02, 0x03, 0x04, 0x0F},
      .last_call_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_OutOfBytes) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {3, 1, 2},
      .memory_before{0x0A, 0x0B, 0x0C},
      .memory_after{0x0A, 0x0B, 0x02, 0x00, 0x00},
      .last_call_data{0x01, 0x02},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_OutOfBytes_WriteZero) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {3, 1, 2},
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
      .memory_after{0x0A, 0x0B, 0x02, 0x00, 0x00, 0x0F},
      .last_call_data{0x01, 0x02},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_Grow) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 1,
      .stack_before = {3, 1, 2},
      .memory_after{0x00, 0x00, 0x02, 0x03, 0x04},
      .last_call_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .gas_after = 2,
      .stack_before = {3, 1, 2},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 8,
      .gas_after = 5,
      .stack_before = {3, 1, 2},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_StackError) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .gas_after = 10,
      .stack_before = {3, 1},
      .stack_after = {3, 1},
  });
}

///////////////////////////////////////////////////////////
// POP
TEST(InterpreterTest, POP) {
  RunInterpreterTest({
      .code = {op::POP},
      .state_after = RunState::kDone,
      .gas_before = 5,
      .gas_after = 3,
      .stack_before = {3},
      .stack_after = {},
  });
}

TEST(InterpreterTest, POP_OutOfGas) {
  RunInterpreterTest({
      .code = {op::POP},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
      .stack_before = {3},
  });
}

TEST(InterpreterTest, POP_StackError) {
  RunInterpreterTest({
      .code = {op::POP},
      .state_after = RunState::kErrorStack,
      .gas_before = 5,
      .stack_before = {},
  });
}

///////////////////////////////////////////////////////////
// MLOAD
TEST(InterpreterTest, MLOAD) {
  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {0},
      .stack_after = {0xFF},
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

  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {2},
      .stack_after = {0xFF},
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

TEST(InterpreterTest, MLOAD_Grow) {
  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {32},
      .stack_after = {0},
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

  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {1},
      .stack_after = {0xFF00},
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

TEST(InterpreterTest, MLOAD_RetainExisting) {
  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {0},
      .stack_after = {0xFF},
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

TEST(InterpreterTest, MLOAD_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1},
      .memory_before{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },

  });
}

TEST(InterpreterTest, MLOAD_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kErrorGas,
      .gas_before = 5,
      .stack_before = {1},
      .memory_before{
          0xFF, 0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
          0,    0, 0, 0, 0, 0, 0, 0,  //
      },
  });
}

TEST(InterpreterTest, MLOAD_StackError) {
  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kErrorStack,
      .gas_before = 100,
  });
}

///////////////////////////////////////////////////////////
// MSTORE
TEST(InterpreterTest, MSTORE) {
  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {0xFF, 0},
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

TEST(InterpreterTest, MSTORE_Grow) {
  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 1,
      .stack_before = {0xFF, 2},
      .memory_after{
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0,    0, 0, 0, 0, 0, 0,  //
          0, 0xFF,
      },
  });

  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {0xFF, 2},
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

TEST(InterpreterTest, MSTORE_RetainExisting) {
  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {0xFF, 2},
      .stack_after = {},
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

TEST(InterpreterTest, MSTORE_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {0xFF, 0},
  });
}

TEST(InterpreterTest, MSTORE_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kErrorGas,
      .gas_before = 5,
      .stack_before = {0xFF, 0},
  });
}

TEST(InterpreterTest, MSTORE_StackError) {
  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// MSTORE8
TEST(InterpreterTest, MSTORE8) {
  RunInterpreterTest({
      .code = {op::MSTORE8},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {0xBB, 1},
      .memory_before = {0xAA, 0, 0xCC, 0xDD, 0, 0, 0, 0},
      .memory_after = {0xAA, 0xBB, 0xCC, 0xDD, 0, 0, 0, 0},
  });
}

TEST(InterpreterTest, MSTORE8_Grow) {
  RunInterpreterTest({
      .code = {op::MSTORE8},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {0xFF, 32},
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

TEST(InterpreterTest, MSTORE8_RetainExisting) {
  RunInterpreterTest({
      .code = {op::MSTORE8},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {0xFF, 2},
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

TEST(InterpreterTest, MSTORE8_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::MSTORE8},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {0xFF, 0},
  });
}

TEST(InterpreterTest, MSTORE8_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::MSTORE8},
      .state_after = RunState::kErrorGas,
      .gas_before = 5,
      .stack_before = {0xFF, 0},
  });
}

TEST(InterpreterTest, MSTORE8_StackError) {
  RunInterpreterTest({
      .code = {op::MSTORE8},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {0xFF},
  });
}

///////////////////////////////////////////////////////////
// JUMP
TEST(InterpreterTest, JUMP) {
  RunInterpreterTest({
      .code = {op::JUMP,       //
               op::PUSH1, 24,  // should be skipped
               op::JUMPDEST,   //
               op::PUSH1, 42},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 4988,
      .stack_before = {3},
      .stack_after = {42},
  });
}

TEST(InterpreterTest, JUMP_Invalid) {
  RunInterpreterTest({
      .code = {op::PUSH4, op::JUMPDEST, op::JUMPDEST, op::JUMPDEST, op::JUMPDEST,  //
               op::PUSH1, 3,                                                       //
               op::JUMP},
      .state_after = RunState::kErrorJump,
      .gas_before = 5000,
  });
}

TEST(InterpreterTest, JUMP_OutOfGas) {
  RunInterpreterTest({
      .code = {op::JUMP},
      .state_after = RunState::kErrorGas,
      .gas_before = 7,
      .stack_before = {0},
  });
}

TEST(InterpreterTest, JUMP_StackError) {
  RunInterpreterTest({
      .code = {op::JUMP},
      .state_after = RunState::kErrorStack,
      .gas_before = 100,
  });
}

///////////////////////////////////////////////////////////
// JUMPI
TEST(InterpreterTest, JUMPI) {
  RunInterpreterTest({
      .code = {op::JUMPI,      //
               op::PUSH1, 24,  //
               op::JUMPDEST,   //
               op::PUSH1, 42},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 4983,
      .stack_before = {0, 3},
      .stack_after = {24, 42},
  });

  RunInterpreterTest({
      .code = {op::JUMPI,      //
               op::PUSH1, 24,  //
               op::JUMPDEST,   //
               op::PUSH1, 42},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 4986,
      .stack_before = {1, 3},
      .stack_after = {42},
  });
}

TEST(InterpreterTest, JUMPI_Invalid) {
  RunInterpreterTest({
      .code = {op::PUSH4, op::JUMPDEST, op::JUMPDEST, op::JUMPDEST, op::JUMPDEST,  //
               op::PUSH1, 3,                                                       //
               op::PUSH1, 1,                                                       //
               op::JUMPI},
      .state_after = RunState::kErrorJump,
      .gas_before = 5000,
  });
}

TEST(InterpreterTest, JUMPI_OutOfGas) {
  RunInterpreterTest({
      .code = {op::JUMPI,      //
               op::PUSH1, 24,  //
               op::JUMPDEST,   //
               op::PUSH1, 42},
      .state_after = RunState::kErrorGas,
      .gas_before = 9,
      .stack_before = {0, 3},
  });
}

TEST(InterpreterTest, JUMPI_StackError) {
  RunInterpreterTest({
      .code = {op::JUMPI,      //
               op::PUSH1, 24,  //
               op::JUMPDEST,   //
               op::PUSH1, 42},
      .state_after = RunState::kErrorStack,
      .gas_before = 100,
      .stack_before = {0},
  });
}

///////////////////////////////////////////////////////////
// PC
TEST(InterpreterTest, PC) {
  RunInterpreterTest({
      .code = {op::PC, op::PC, op::PC},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_after = {0, 1, 2},
  });
}

TEST(InterpreterTest, PC_OutOfGas) {
  RunInterpreterTest({
      .code = {op::PC},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

///////////////////////////////////////////////////////////
// MSIZE
TEST(InterpreterTest, MSIZE) {
  RunInterpreterTest({
      .code = {op::MSIZE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::MSIZE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {32},
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

TEST(InterpreterTest, MSIZE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::MSIZE},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

///////////////////////////////////////////////////////////
// GAS
TEST(InterpreterTest, GAS) {
  RunInterpreterTest({
      .code = {op::GAS},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 98,
      .stack_after = {98},
  });
}

TEST(InterpreterTest, GAS_OutOfGas) {
  RunInterpreterTest({
      .code = {op::GAS},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

///////////////////////////////////////////////////////////
// JUMPDEST
TEST(InterpreterTest, JUMPDEST) {
  RunInterpreterTest({
      .code = {op::JUMPDEST},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 9,
  });
}

TEST(InterpreterTest, JUMPDEST_OutOfGas) {
  RunInterpreterTest({
      .code = {op::JUMPDEST},
      .state_after = RunState::kErrorGas,
      .gas_before = 0,
  });
}

///////////////////////////////////////////////////////////
// PUSH
TEST(InterpreterTest, PUSH) {
  RunInterpreterTest({
      .code = {op::PUSH4, 0xFF, 0xFF, 0xFF, 0xFF},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {},
      .stack_after = {0xFFFFFFFF},
  });

  RunInterpreterTest({
      .code = {op::PUSH20, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
               0xFF,       0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xAA},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {},
      .stack_after = {uint256_t(0xFFFFFFFFFFFFFFAA, 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFF)},
  });
}

TEST(InterpreterTest, PUSH_OutOfGas) {
  RunInterpreterTest({
      .code = {op::PUSH4, 0xFF, 0xFF, 0xFF, 0xFF},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
  });
}

TEST(InterpreterTest, PUSH_OutOfBytes) {
  RunInterpreterTest({
      .code = {op::PUSH4, 0xFF, 0xFF, /* 0 byte added for test */},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
  });
}

TEST(InterpreterTest, DISABLED_PUSH_StackOverflow) {}

///////////////////////////////////////////////////////////
// DUP
TEST(InterpreterTest, DUP) {
  RunInterpreterTest({
      .code = {op::DUP4},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {4, 3, 2, 1},
      .stack_after = {4, 3, 2, 1, 4},
  });

  RunInterpreterTest({
      .code = {op::DUP15},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
      .stack_after = {16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 15},
  });
}

TEST(InterpreterTest, DUP_OutOfGas) {
  RunInterpreterTest({
      .code = {op::DUP4},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
      .stack_before = {4, 3, 2, 1},
  });
}

TEST(InterpreterTest, DUP_StackError) {
  RunInterpreterTest({
      .code = {op::DUP4},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {3, 2, 1},
  });
}

TEST(InterpreterTest, DISABLED_DUP_StackOverflow) {}

///////////////////////////////////////////////////////////
// SWAP
TEST(InterpreterTest, SWAP) {
  RunInterpreterTest({
      .code = {op::SWAP4},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {5, 4, 3, 2, 1},
      .stack_after = {1, 4, 3, 2, 5},
  });

  RunInterpreterTest({
      .code = {op::SWAP16},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
      .stack_after = {18, 1, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 17},
  });
}

TEST(InterpreterTest, SWAP_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SWAP4},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
      .stack_before = {5, 4, 3, 2, 1},
  });
}

TEST(InterpreterTest, SWAP_StackError) {
  RunInterpreterTest({
      .code = {op::SWAP4},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {4, 3, 2, 1},
  });
}

///////////////////////////////////////////////////////////
// RETURN
TEST(InterpreterTest, RETURN) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 10,
      .stack_before = {2, 1},
      .memory_before = {0xAA, 0xBB, 0xCC},
      .memory_after = {0xAA, 0xBB, 0xCC},
      .return_data = {0xBB, 0xCC},
  });
}

TEST(InterpreterTest, RETURN_GrowMemory) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {3, 1},
      .memory_after = {0, 0, 0, 0},
      .return_data = {0, 0, 0},
  });
}

TEST(InterpreterTest, RETURN_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {3, 1},
  });
}

TEST(InterpreterTest, RETURN_StackError) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {3},
  });
}

///////////////////////////////////////////////////////////
// REVERT
TEST(InterpreterTest, REVERT) {
  RunInterpreterTest({
      .code = {op::REVERT},
      .state_after = RunState::kRevert,
      .gas_before = 10,
      .gas_after = 10,
      .stack_before = {2, 1},
      .memory_before = {0xAA, 0xBB, 0xCC},
      .memory_after = {0xAA, 0xBB, 0xCC},
      .return_data = {0xBB, 0xCC},
  });
}

TEST(InterpreterTest, REVERT_GrowMemory) {
  RunInterpreterTest({
      .code = {op::REVERT},
      .state_after = RunState::kRevert,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {3, 1},
      .memory_after = {0, 0, 0, 0},
      .return_data = {0, 0, 0},
  });
}

TEST(InterpreterTest, REVERT_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::REVERT},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {3, 1},
  });
}

TEST(InterpreterTest, REVERT_StackError) {
  RunInterpreterTest({
      .code = {op::REVERT},
      .state_after = RunState::kErrorStack,
      .gas_before = 10,
      .stack_before = {3},
  });
}

}  // namespace
}  // namespace tosca::evmzero

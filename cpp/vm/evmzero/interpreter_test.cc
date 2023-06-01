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

  Stack stack_before = {};
  Stack stack_after = {};
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
TEST(InterpreterTests, ADD) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {1, 6},
      .stack_after = {7},
  });
}

TEST(InterpreterTests, ADD_Overflow) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {1, kUint256Max},
      .stack_after = {0},
  });
}

TEST(InterpreterTests, ADD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {1, kUint256Max},
  });
}

TEST(InterpreterTests, ADD_StackError) {
  RunInterpreterTest({
      .code = {op::ADD},
      .state_after = RunState::kErrorStack,
      .gas_before = 7,
  });
}

}  // namespace
}  // namespace tosca::evmzero

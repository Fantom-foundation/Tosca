#include "vm/evmzero/interpreter.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

#include <evmc/evmc.hpp>

#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero {
namespace {

using ::testing::_;
using ::testing::AllOf;
using ::testing::Args;
using ::testing::DoAll;
using ::testing::ElementsAre;
using ::testing::Return;
using ::testing::SetArrayArgument;

class MockHost : public evmc::Host {
 public:
  MOCK_METHOD(bool, account_exists, (const evmc::address& addr), (const, noexcept, override));
  MOCK_METHOD(evmc::bytes32, get_storage, (const evmc::address& addr, const evmc::bytes32& key),
              (const, noexcept, override));
  MOCK_METHOD(evmc_storage_status, set_storage,
              (const evmc::address& addr, const evmc::bytes32& key, const evmc::bytes32& value), (noexcept, override));
  MOCK_METHOD(evmc::uint256be, get_balance, (const evmc::address& addr), (const, noexcept, override));
  MOCK_METHOD(size_t, get_code_size, (const evmc::address& addr), (const, noexcept, override));
  MOCK_METHOD(evmc::bytes32, get_code_hash, (const evmc::address& addr), (const, noexcept, override));
  MOCK_METHOD(size_t, copy_code,
              (const evmc::address& addr, size_t code_offset, uint8_t* buffer_data, size_t buffer_size),
              (const, noexcept, override));
  MOCK_METHOD(bool, selfdestruct, (const evmc::address& addr, const evmc::address& beneficiary), (noexcept, override));
  MOCK_METHOD(evmc::Result, call, (const evmc_message& msg), (noexcept, override));
  MOCK_METHOD(evmc_tx_context, get_tx_context, (), (const, noexcept, override));
  MOCK_METHOD(evmc::bytes32, get_block_hash, (int64_t black_number), (const, noexcept, override));
  MOCK_METHOD(void, emit_log,
              (const evmc::address& addr, const uint8_t* data, size_t data_size, const evmc::bytes32 topics[],
               size_t num_topics),
              (noexcept, override));
  MOCK_METHOD(evmc_access_status, access_account, (const evmc::address& addr), (noexcept));
  MOCK_METHOD(evmc_access_status, access_storage, (const evmc::address& addr, const evmc::bytes32& key), (noexcept));
};

struct InterpreterTestDescription {
  std::vector<uint8_t> code;

  RunState state_after = RunState::kDone;
  bool is_static_call = false;

  int64_t gas_before = 0;
  int64_t gas_after = 0;
  int64_t gas_refund_before = 0;
  int64_t gas_refund_after = 0;

  Stack stack_before;
  Stack stack_after;

  Memory memory_before;
  Memory memory_after;

  std::vector<uint8_t> last_call_data;
  std::vector<uint8_t> return_data;

  evmc_message message{};
  evmc::HostInterface* host = nullptr;

  evmc_revision revision = EVMC_ISTANBUL;
};

void RunInterpreterTest(const InterpreterTestDescription& desc) {
  internal::Context ctx{
      .is_static_call = desc.is_static_call,
      .gas = desc.gas_before,
      .gas_refunds = desc.gas_refund_before,
      .code = desc.code,
      .return_data = desc.last_call_data,
      .memory = desc.memory_before,
      .stack = desc.stack_before,
      .message = &desc.message,
      .host = desc.host,
      .revision = desc.revision,
  };

  internal::RunInterpreter<false>(ctx);

  ASSERT_EQ(ctx.state, desc.state_after);

  if (IsSuccess(ctx.state)) {
    EXPECT_EQ(ctx.gas, desc.gas_after);
    EXPECT_EQ(ctx.gas_refunds, desc.gas_refund_after);
    EXPECT_EQ(ctx.stack, desc.stack_after);
    EXPECT_EQ(ctx.memory, desc.memory_after);
    EXPECT_EQ(ctx.return_data, desc.return_data);
  }
}

///////////////////////////////////////////////////////////
// MemoryExpansionCost
TEST(InterpreterTest, MemoryExpansionCost) {
  internal::Context ctx;

  const auto [cost, offset, size] = ctx.MemoryExpansionCost(128, 42);
  EXPECT_EQ(cost, 18);
  EXPECT_EQ(offset, 128);
  EXPECT_EQ(size, 42);
}

TEST(InterpreterTest, MemoryExpansionCost_NoGrowNeeded) {
  internal::Context ctx;
  ctx.memory.Grow(128, 42);

  const auto [cost, offset, size] = ctx.MemoryExpansionCost(128, 42);
  EXPECT_EQ(cost, 0);
  EXPECT_EQ(offset, 128);
  EXPECT_EQ(size, 42);
}

TEST(InterpreterTest, MemoryExpansionCost_ZeroSize) {
  internal::Context ctx;

  const auto [cost, offset, size] = ctx.MemoryExpansionCost(128, 0);
  EXPECT_EQ(cost, 0);
  EXPECT_EQ(offset, 128);
  EXPECT_EQ(size, 0);
}

TEST(InterpreterTest, MemoryExpansionCost_Overflow) {
  internal::Context ctx;

  const auto [cost, offset, size] = ctx.MemoryExpansionCost(1ul << 63, 1ul << 63);
  EXPECT_EQ(cost, internal::kMaxGas);
  EXPECT_EQ(offset, 1ul << 63);
  EXPECT_EQ(size, 1ul << 63);
}

TEST(InterpreterTest, MemoryExpansionCost_OversizedMemory) {
  internal::Context ctx;

  {
    const auto [cost, offset, size] = ctx.MemoryExpansionCost(0, uint256_t{1} << 100);
    EXPECT_EQ(cost, internal::kMaxGas);
    EXPECT_EQ(offset, 0);
    EXPECT_EQ(size, 0);
  }

  {
    const auto [cost, offset, size] = ctx.MemoryExpansionCost(uint256_t{1} << 100, 1);
    EXPECT_EQ(cost, internal::kMaxGas);
    EXPECT_EQ(offset, 0);
    EXPECT_EQ(size, 0);
  }
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

TEST(InterpreterTest, STOP_NoReturnData) {
  RunInterpreterTest({
      .code = {op::STOP},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 7,
      .last_call_data = {0xFF},
      .return_data = {},
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .stack_before = {kUint256Max, 100},
      .stack_after = {kUint256Max << 100},
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .stack_before = {kUint256Max, 100},
      .stack_after = {kUint256Max >> 100},
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
      .state_after = RunState::kErrorStackUnderflow,
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

  RunInterpreterTest({
      .code = {op::SAR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {uint256_t{0, 0, 1}, 128},
      .stack_after = {1},
  });

  RunInterpreterTest({
      .code = {op::SAR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {kUint256Max, 100},
      .stack_after = {kUint256Max},
  });

  RunInterpreterTest({
      .code = {op::SAR},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {kUint256Max, 256},
      .stack_after = {0},
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
      .state_after = RunState::kErrorStackUnderflow,
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

TEST(InterpreterTest, SHA3_ZeroSize) {
  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 70,
      .stack_before = {0, 42},
      .stack_after = {uint256_t(0x7BFAD8045D85A470, 0xE500B653CA82273B, 0x927E7DB2DCC703C0, 0xC5D2460186F7233C)},
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 100,
      .stack_before = {4},
  });
}

TEST(InterpreterTest, SHA3_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kErrorGas,
      .gas_before = 1000000,
      .stack_before = {uint256_t{1} << 100, 0},
  });

  RunInterpreterTest({
      .code = {op::SHA3},
      .state_after = RunState::kErrorGas,
      .gas_before = 1000000,
      .stack_before = {1, uint256_t{1} << 100},
  });
}

///////////////////////////////////////////////////////////
// ADDRESS
TEST(InterpreterTest, ADDRESS) {
  RunInterpreterTest({
      .code = {op::ADDRESS},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::ADDRESS},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0x42},
      .message{.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTest, ADDRESS_OutOfGas) {
  RunInterpreterTest({
      .code = {op::ADDRESS},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

///////////////////////////////////////////////////////////
// BALANCE
TEST(InterpreterTest, BALANCE) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42)))  //
      .Times(1)
      .WillOnce(Return(evmc::uint256be(0x21)));

  RunInterpreterTest({
      .code = {op::BALANCE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2300,
      .stack_before = {0x42},
      .stack_after = {0x21},
      .host = &host,
  });
}

TEST(InterpreterTest, BALANCE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::BALANCE},
      .state_after = RunState::kErrorGas,
      .gas_before = 600,
      .stack_before = {0x42},
  });
}

TEST(InterpreterTest, BALANCE_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, get_balance(evmc::address(0x42)))  //
      .Times(1)
      .WillOnce(Return(evmc::uint256be(0x21)));

  RunInterpreterTest({
      .code = {op::BALANCE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 400,
      .stack_before = {0x42},
      .stack_after = {0x21},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, BALANCE_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, get_balance(evmc::address(0x42)))  //
      .Times(1)
      .WillOnce(Return(evmc::uint256be(0x21)));

  RunInterpreterTest({
      .code = {op::BALANCE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2900,
      .stack_before = {0x42},
      .stack_after = {0x21},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, BALANCE_OutOfGas_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));

  RunInterpreterTest({
      .code = {op::BALANCE},
      .state_after = RunState::kErrorGas,
      .gas_before = 2500,
      .stack_before = {0x42},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, BALANCE_OutOfGas_WARM) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));

  RunInterpreterTest({
      .code = {op::BALANCE},
      .state_after = RunState::kErrorGas,
      .gas_before = 90,
      .stack_before = {0x42},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, BALANCE_StackError) {
  RunInterpreterTest({
      .code = {op::BALANCE},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 3000,
  });
}

///////////////////////////////////////////////////////////
// ORIGIN
TEST(InterpreterTest, ORIGIN) {
  evmc_tx_context tx_context{
      .tx_origin = evmc::address(0x42),
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::ORIGIN},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0x42},
      .host = &host,
  });
}

TEST(InterpreterTest, ORIGIN_OutOfGas) {
  RunInterpreterTest({
      .code = {op::ORIGIN},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_ORIGIN_StackOverflow) {}

///////////////////////////////////////////////////////////
// CALLER
TEST(InterpreterTest, CALLER) {
  RunInterpreterTest({
      .code = {op::CALLER},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0},
  });
  RunInterpreterTest({
      .code = {op::CALLER},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0x42},
      .message{.sender = evmc::address(0x42)},
  });
}

TEST(InterpreterTest, CALLER_OutOfGas) {
  RunInterpreterTest({
      .code = {op::CALLER},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

///////////////////////////////////////////////////////////
// CALLVALUE
TEST(InterpreterTest, CALLVALUE) {
  RunInterpreterTest({
      .code = {op::CALLVALUE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 5,
      .stack_after = {0},
  });

  RunInterpreterTest({
      .code = {op::CALLVALUE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 5,
      .stack_after = {0x42},
      .message{.value = evmc::uint256be(0x42)},
  });
}

TEST(InterpreterTest, CALLVALUE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::CALLVALUE},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

///////////////////////////////////////////////////////////
// CALLDATALOAD
TEST(InterpreterTest, CALLDATALOAD) {
  RunInterpreterTest({
      .code = {op::CALLDATALOAD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0},
      .stack_after = {0},
  });

  std::array<uint8_t, 32> input_data{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,  //
                                     0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,  //
                                     0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,  //
                                     0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37};
  RunInterpreterTest({
      .code = {op::CALLDATALOAD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0},
      .stack_after = {uint256_t(0x3031323334353637, 0x2021222324252627, 0x1011121314151617, 0x0001020304050607)},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });

  RunInterpreterTest({
      .code = {op::CALLDATALOAD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {30},
      .stack_after = {uint256_t(0, 0, 0, 0x3637000000000000)},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATALOAD_InputLargerThan32Bytes) {
  std::array<uint8_t, 128> input_data{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,  //
                                      0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,  //
                                      0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,  //
                                      0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37};
  RunInterpreterTest({
      .code = {op::CALLDATALOAD},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 4,
      .stack_before = {0},
      .stack_after = {uint256_t(0x3031323334353637, 0x2021222324252627, 0x1011121314151617, 0x0001020304050607)},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATALOAD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::CALLDATALOAD},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
      .stack_before = {0},
  });
}

TEST(InterpreterTest, CALLDATALOAD_StackError) {
  RunInterpreterTest({
      .code = {op::CALLDATALOAD},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 7,
      .stack_after = {},
  });
}

///////////////////////////////////////////////////////////
// CALLDATASIZE
TEST(InterpreterTest, CALLDATASIZE) {
  RunInterpreterTest({
      .code = {op::CALLDATASIZE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 5,
      .stack_after = {0},
  });

  std::array<uint8_t, 3> input_data{};
  RunInterpreterTest({
      .code = {op::CALLDATASIZE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 5,
      .stack_after = {3},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATASIZE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::CALLDATASIZE},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_CALLDATASIZE_StackOverflow) {}

///////////////////////////////////////////////////////////
// CALLDATACOPY
TEST(InterpreterTest, CALLDATACOPY) {
  std::array<uint8_t, 4> input_data{0xA0, 0xA1, 0xA2, 0xA3};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 1,
      .stack_before = {3, 1, 2},
      .memory_after = {0, 0, 0xA1, 0xA2, 0xA3},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATACOPY_ZeroSize) {
  std::array<uint8_t, 4> input_data{0xA0, 0xA1, 0xA2, 0xA3};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 7,
      .stack_before = {0, 1, 2},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATACOPY_RetainMemory) {
  std::array<uint8_t, 4> input_data{0xA0, 0xA1, 0xA2, 0xA3};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {3, 1, 2},
      .memory_before = {0xFF, 0xFF, 0, 0, 0, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0xA1, 0xA2, 0xA3, 0xFF, 0xFF, 0xFF},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATACOPY_WriteZeros) {
  std::array<uint8_t, 4> input_data{0xA0, 0xA1};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 4,
      .stack_before = {3, 1, 2},
      .memory_before = {0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
      .memory_after = {0xFF, 0xFF, 0xA1, 0, 0, 0xFF, 0xFF, 0xFF},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATACOPY_OutOfBounds) {
  std::array<uint8_t, 2> input_data{0xA0, 0xA1};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 1,
      .stack_before = {3, 1, 2},
      .memory_after = {0, 0, 0xA1, 0, 0},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATACOPY_OutOfGas_Static) {
  std::array<uint8_t, 2> input_data{0xA0, 0xA1};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 2,
      .stack_before = {3, 1, 2},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATACOPY_OutOfGas_Dynamic) {
  std::array<uint8_t, 2> input_data{0xA0, 0xA1};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 8,
      .stack_before = {3, 1, 2},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });
}

TEST(InterpreterTest, CALLDATACOPY_StackError) {
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 100,
      .stack_before = {3, 1},
  });
}

TEST(InterpreterTest, CALLDATACOPY_OversizedMemory) {
  std::array<uint8_t, 4> input_data{0xA0, 0xA1, 0xA2, 0xA3};
  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100, 0, 0},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
  });

  RunInterpreterTest({
      .code = {op::CALLDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {1, 0, uint256_t{1} << 100},
      .message{.input_data = input_data.data(), .input_size = input_data.size()},
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
      .stack_after = {7},
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

TEST(InterpreterTest, CODECOPY_ZeroSize) {
  RunInterpreterTest({
      .code = {op::PUSH1, 23,  //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 100,
      .gas_after = 100 - 10 - 3,
      .stack_before = {0, 1, 2},
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 100,
      .stack_before = {3, 1},
  });
}

TEST(InterpreterTest, CODECOPY_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::PUSH1, 23,  //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100, 0, 0},
  });

  RunInterpreterTest({
      .code = {op::PUSH1, 23,  //
               op::POP,        //
               op::PUSH1, 42,  //
               op::POP,        //
               op::CODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {1, 0, uint256_t{1} << 100},
  });
}

///////////////////////////////////////////////////////////
// GASPRICE
TEST(InterpreterTest, GASPRICE) {
  evmc_tx_context tx_context{
      .tx_gas_price = evmc::uint256be(42),
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::GASPRICE},
      .state_after = RunState::kDone,
      .gas_before = 7,
      .gas_after = 5,
      .stack_after = {42},
      .host = &host,
  });
}

TEST(InterpreterTest, GASPRICE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::GASPRICE},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_GASPRICE_StackOverflow) {}

///////////////////////////////////////////////////////////
// EXTCODESIZE
TEST(InterpreterTest, EXTCODESIZE) {
  MockHost host;
  EXPECT_CALL(host, get_code_size(evmc::address(0x42))).Times(1).WillOnce(Return(16));

  RunInterpreterTest({
      .code = {op::EXTCODESIZE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2300,
      .stack_before = {0x42},
      .stack_after = {16},
      .host = &host,
  });
}

TEST(InterpreterTest, EXTCODESIZE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::EXTCODESIZE},
      .state_after = RunState::kErrorGas,
      .gas_before = 600,
      .stack_before = {0x42},
  });
}

TEST(InterpreterTest, EXTCODESIZE_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, get_code_size(evmc::address(0x42))).Times(1).WillOnce(Return(16));

  RunInterpreterTest({
      .code = {op::EXTCODESIZE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 400,
      .stack_before = {0x42},
      .stack_after = {16},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODESIZE_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, get_code_size(evmc::address(0x42))).Times(1).WillOnce(Return(16));

  RunInterpreterTest({
      .code = {op::EXTCODESIZE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2900,
      .stack_before = {0x42},
      .stack_after = {16},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODESIZE_OutOfGas_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));

  RunInterpreterTest({
      .code = {op::EXTCODESIZE},
      .state_after = RunState::kErrorGas,
      .gas_before = 2500,
      .stack_before = {0x42},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODESIZE_OutOfGas_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));

  RunInterpreterTest({
      .code = {op::EXTCODESIZE},
      .state_after = RunState::kErrorGas,
      .gas_before = 90,
      .stack_before = {0x42},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, DISABLED_EXTCODESIZE_StackOverflow) {}

///////////////////////////////////////////////////////////
// EXTCODECOPY
TEST(InterpreterTest, EXTCODECOPY) {
  const std::vector<uint8_t> code = {op::PUSH4, 0x0A, 0x0B, 0x0C, 0xD};

  MockHost host;
  EXPECT_CALL(host, copy_code(evmc::address(0x42), 1, _, 3))  //
      .Times(1)
      .WillOnce(DoAll(SetArrayArgument<2>(code.data() + 1, code.data() + 1 + 3), Return(3)));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 3000 - 700 - 6,
      .stack_before = {3, 1, 2, 0x42},
      .memory_after = {0, 0, 0x0A, 0x0B, 0x0C},
      .host = &host,
  });
}

TEST(InterpreterTest, EXTCODECOPY_ZeroSize) {
  const std::vector<uint8_t> code = {op::PUSH4, 0x0A, 0x0B, 0x0C, 0xD};

  MockHost host;
  EXPECT_CALL(host, copy_code(evmc::address(0x42), 1, _, 0))  //
      .Times(1)
      .WillOnce(Return(0));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 3000 - 700,
      .stack_before = {0, 1, 2, 0x42},
      .host = &host,
  });
}

TEST(InterpreterTest, EXTCODECOPY_OutOfGas) {
  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 699,
      .stack_before = {3, 1, 2, 0x42},
  });
}

TEST(InterpreterTest, EXTCODECOPY_RetainMemory) {
  const std::vector<uint8_t> code = {op::PUSH4, 0x0A, 0x0B, 0x0C, 0xD};

  MockHost host;
  EXPECT_CALL(host, copy_code(evmc::address(0x42), 1, _, 3))  //
      .Times(1)
      .WillOnce(DoAll(SetArrayArgument<2>(code.data() + 1, code.data() + 1 + 3), Return(3)));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 3000 - 700 - 3,
      .stack_before = {3, 1, 2, 0x42},
      .memory_before = {0, 0, 0, 0, 0, 0xFF},
      .memory_after = {0, 0, 0x0A, 0x0B, 0x0C, 0xFF},
      .host = &host,
  });
}

TEST(InterpreterTest, EXTCODECOPY_WriteZeros) {
  const std::vector<uint8_t> code = {op::PUSH1, 0x0A};

  MockHost host;
  EXPECT_CALL(host, copy_code(evmc::address(0x42), 1, _, 3))  //
      .Times(1)
      .WillOnce(DoAll(SetArrayArgument<2>(code.data() + 1, code.data() + 2), Return(3)));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 3000 - 700 - 3,
      .stack_before = {3, 1, 2, 0x42},
      .memory_before = {0, 0, 0, 0xFF, 0xFF, 0xFF},
      .memory_after = {0, 0, 0x0A, 0, 0, 0xFF},
      .host = &host,
  });
}

TEST(InterpreterTest, EXTCODECOPY_Cold) {
  const std::vector<uint8_t> code = {op::PUSH4, 0x0A, 0x0B, 0x0C, 0xD};

  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, copy_code(evmc::address(0x42), 1, _, 3))  //
      .Times(1)
      .WillOnce(DoAll(SetArrayArgument<2>(code.data() + 1, code.data() + 1 + 3), Return(3)));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 3000 - 2600 - 6,
      .stack_before = {3, 1, 2, 0x42},
      .memory_after = {0, 0, 0x0A, 0x0B, 0x0C},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODECOPY_Warm) {
  const std::vector<uint8_t> code = {op::PUSH4, 0x0A, 0x0B, 0x0C, 0xD};

  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, copy_code(evmc::address(0x42), 1, _, 3))  //
      .Times(1)
      .WillOnce(DoAll(SetArrayArgument<2>(code.data() + 1, code.data() + 1 + 3), Return(3)));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 3000 - 100 - 6,
      .stack_before = {3, 1, 2, 0x42},
      .memory_after = {0, 0, 0x0A, 0x0B, 0x0C},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODECOPY_OutOfGas_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 2500,
      .stack_before = {3, 1, 2, 0x42},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODECOPY_OutOfGas_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 690,
      .stack_before = {3, 1, 2, 0x42},
      .host = &host,
  });
}

TEST(InterpreterTest, EXTCODECOPY_StackError) {
  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 3000,
      .stack_before = {3, 1, 2},
  });
}

TEST(InterpreterTest, EXTCODECOPY_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100, 0, 0, 0x42},
  });

  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {0, 0, uint256_t{1} << 100, 0x42},
  });
}

TEST(InterpreterTest, EXTCODECOPY_OutOfBoundsCodeOffset) {
  RunInterpreterTest({
      .code = {op::EXTCODECOPY},
      .state_after = RunState::kDone,
      .gas_before = 1000,
      .gas_after = 297,
      .stack_before = {4, uint256_t{1} << 100, 0, 0x42},
      .memory_before = {0xAA, 0xBB, 0xCC, 0xDD, 0xEE},
      .memory_after = {0x00, 0x00, 0x00, 0x00, 0xEE},
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
      .memory_after{0x0A, 0x0B, 0x02, 0x03, 0x04, 0x0F},
      .last_call_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_OutOfBounds) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorReturnDataCopyOutOfBounds,
      .gas_before = 10,
      .stack_before = {3, 1, 2},
      .memory_before{0x0A, 0x0B, 0x0C},
      .last_call_data{0x01, 0x02},
  });

  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorReturnDataCopyOutOfBounds,
      .gas_before = 10,
      .stack_before = {0, 1, 2},
      .memory_before{0x0A, 0x0B, 0x0C},
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 10,
      .gas_after = 10,
      .stack_before = {3, 1},
      .stack_after = {3, 1},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100, 0, 0},
      .last_call_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });

  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {0, 0, uint256_t{1} << 100},
      .last_call_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

TEST(InterpreterTest, RETURNDATACOPY_OffsetOverflow) {
  RunInterpreterTest({
      .code = {op::RETURNDATACOPY},
      .state_after = RunState::kErrorReturnDataCopyOutOfBounds,
      .gas_before = 10000000,
      .stack_before = {1, kUint256Max, 0},
      .last_call_data{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
  });
}

///////////////////////////////////////////////////////////
// EXTCODEHASH
TEST(InterpreterTest, EXTCODEHASH) {
  MockHost host;
  EXPECT_CALL(host, get_code_hash(evmc::address(0x42)))  //
      .Times(1)
      .WillOnce(Return(evmc::bytes32(0x0a0b0c0d)));

  RunInterpreterTest({
      .code = {op::EXTCODEHASH},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2300,
      .stack_before = {0x42},
      .stack_after = {0x0a0b0c0d},
      .host = &host,
  });
}

TEST(InterpreterTest, EXTCODEHASH_OutOfGas) {
  RunInterpreterTest({
      .code = {op::EXTCODEHASH},
      .state_after = RunState::kErrorGas,
      .gas_before = 600,
      .stack_before = {0x42},
  });
}

TEST(InterpreterTest, EXTCODEHASH_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, get_code_hash(evmc::address(0x42)))  //
      .Times(1)
      .WillOnce(Return(evmc::bytes32(0x0a0b0c0d)));

  RunInterpreterTest({
      .code = {op::EXTCODEHASH},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 400,
      .stack_before = {0x42},
      .stack_after = {0x0a0b0c0d},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODEHASH_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, get_code_hash(evmc::address(0x42)))  //
      .Times(1)
      .WillOnce(Return(evmc::bytes32(0x0a0b0c0d)));

  RunInterpreterTest({
      .code = {op::EXTCODEHASH},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2900,
      .stack_before = {0x42},
      .stack_after = {0x0a0b0c0d},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODEHASH_OutOfGas_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_COLD));

  RunInterpreterTest({
      .code = {op::EXTCODEHASH},
      .state_after = RunState::kErrorGas,
      .gas_before = 2500,
      .stack_before = {0x42},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODEHASH_OutOfGas_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_account(evmc::address(0x42))).WillRepeatedly(Return(EVMC_ACCESS_WARM));

  RunInterpreterTest({
      .code = {op::EXTCODEHASH},
      .state_after = RunState::kErrorGas,
      .gas_before = 90,
      .stack_before = {0x42},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, EXTCODEHASH_StackError) {
  RunInterpreterTest({
      .code = {op::EXTCODEHASH},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 3000,
  });
}

///////////////////////////////////////////////////////////
// BLOCKHASH
TEST(InterpreterTest, BLOCKHASH) {
  MockHost host;
  EXPECT_CALL(host, get_block_hash(21))  //
      .Times(1)
      .WillOnce(Return(evmc::bytes32(0x0a0b0c0d)));

  RunInterpreterTest({
      .code = {op::BLOCKHASH},
      .state_after = RunState::kDone,
      .gas_before = 40,
      .gas_after = 20,
      .stack_before = {21},
      .stack_after = {0x0a0b0c0d},
      .host = &host,
  });
}

TEST(InterpreterTest, BLOCKHASH_OutOfGas) {
  RunInterpreterTest({
      .code = {op::BLOCKHASH},
      .state_after = RunState::kErrorGas,
      .gas_before = 19,
      .stack_before = {0},
  });
}

TEST(InterpreterTest, BLOCKHASH_StackError) {
  RunInterpreterTest({
      .code = {op::BLOCKHASH},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 40,
  });
}

///////////////////////////////////////////////////////////
// COINBASE
TEST(InterpreterTest, COINBASE) {
  evmc_tx_context tx_context{
      .block_coinbase = evmc::address(0x42),
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::COINBASE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {0x42},
      .host = &host,
  });
}

TEST(InterpreterTest, COINBASE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::COINBASE},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_COINBASE_StackOverflow) {}

///////////////////////////////////////////////////////////
// TIMESTAMP
TEST(InterpreterTest, TIMESTAMP) {
  evmc_tx_context tx_context{
      .block_timestamp = 42,
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::TIMESTAMP},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {42},
      .host = &host,
  });
}

TEST(InterpreterTest, TIMESTAMP_OutOfGas) {
  RunInterpreterTest({
      .code = {op::TIMESTAMP},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_TIMESTAMP_StackOverflow) {}

///////////////////////////////////////////////////////////
// NUMBER
TEST(InterpreterTest, NUMBER) {
  evmc_tx_context tx_context{
      .block_number = 42,
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::NUMBER},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {42},
      .host = &host,
  });
}

TEST(InterpreterTest, NUMBER_OutOfGas) {
  RunInterpreterTest({
      .code = {op::NUMBER},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_NUMBER_StackOverflow) {}

///////////////////////////////////////////////////////////
// DIFFICULTY / PREVRANDAO
TEST(InterpreterTest, DIFFICULTY) {
  evmc_tx_context tx_context{
      .block_prev_randao = evmc::uint256be(42),
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::DIFFICULTY},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {42},
      .host = &host,
  });
}

TEST(InterpreterTest, DIFFICULTY_OutOfGas) {
  RunInterpreterTest({
      .code = {op::DIFFICULTY},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_DIFFICULTY_StackOverflow) {}

///////////////////////////////////////////////////////////
// GASLIMIT
TEST(InterpreterTest, GASLIMIT) {
  evmc_tx_context tx_context{
      .block_gas_limit = 42,
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::GASLIMIT},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {42},
      .host = &host,
  });
}

TEST(InterpreterTest, GASLIMIT_OutOfGas) {
  RunInterpreterTest({
      .code = {op::GASLIMIT},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_GASLIMIT_StackOverflow) {}

///////////////////////////////////////////////////////////
// CHAINID
TEST(InterpreterTest, CHAINID) {
  evmc_tx_context tx_context{
      .chain_id = evmc::uint256be(42),
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::CHAINID},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {42},
      .host = &host,
  });
}

TEST(InterpreterTest, CHAINID_OutOfGas) {
  RunInterpreterTest({
      .code = {op::CHAINID},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
  });
}

TEST(InterpreterTest, DISABLED_CHAINID_StackOverflow) {}

///////////////////////////////////////////////////////////
// SELFBALANCE
TEST(InterpreterTest, SELFBALANCE) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42)))  //
      .Times(1)
      .WillOnce(Return(evmc::uint256be(1042)));

  RunInterpreterTest({
      .code = {op::SELFBALANCE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 5,
      .stack_after = {1042},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SELFBALANCE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SELFBALANCE},
      .state_after = RunState::kErrorGas,
      .gas_before = 4,
  });
}

TEST(InterpreterTest, DISABLED_SELFBALANCE_StackOverflow) {}

///////////////////////////////////////////////////////////
// BASEFEE
TEST(InterpreterTest, BASEFEE) {
  evmc_tx_context tx_context{
      .block_base_fee = evmc::uint256be(42),
  };

  MockHost host;
  EXPECT_CALL(host, get_tx_context()).Times(1).WillOnce(Return(tx_context));

  RunInterpreterTest({
      .code = {op::BASEFEE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {42},
      .host = &host,
      .revision = EVMC_LONDON,
  });
}

TEST(InterpreterTest, BASEFEE_OutOfGas) {
  RunInterpreterTest({
      .code = {op::BASEFEE},
      .state_after = RunState::kErrorGas,
      .gas_before = 1,
      .revision = EVMC_LONDON,
  });
}

TEST(InterpreterTest, DISABLED_BASEFEE_StackOverflow) {}

TEST(InterpreterTest, BASEFEE_PreRevision) {
  RunInterpreterTest({
      .code = {op::BASEFEE},
      .state_after = RunState::kErrorOpcode,
      .gas_before = 10,
      .revision = EVMC_BERLIN,
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 100,
  });
}

TEST(InterpreterTest, MLOAD_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::MLOAD},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100},
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 10,
      .stack_before = {0xFF},
  });
}

TEST(InterpreterTest, MSTORE_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::MSTORE},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {0xFF, uint256_t{1} << 100},
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 10,
      .stack_before = {0xFF},
  });
}

TEST(InterpreterTest, MSTORE8_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::MSTORE8},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {0xBB, uint256_t{1} << 100},
  });
}

///////////////////////////////////////////////////////////
// SLOAD
TEST(InterpreterTest, SLOAD) {
  MockHost host;
  EXPECT_CALL(host, get_storage(evmc::address(0x42), evmc::bytes32(16)))  //
      .Times(1)
      .WillOnce(Return(evmc::bytes32(32)));

  RunInterpreterTest({
      .code = {op::SLOAD},
      .state_after = RunState::kDone,
      .gas_before = 2000,
      .gas_after = 1200,
      .stack_before = {16},
      .stack_after = {32},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SLOAD_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SLOAD},
      .state_after = RunState::kErrorGas,
      .gas_before = 700,
      .stack_before = {16},
      .message = {.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTest, SLOAD_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, get_storage(evmc::address(0x42), evmc::bytes32(16)))  //
      .Times(1)
      .WillOnce(Return(evmc::bytes32(32)));

  RunInterpreterTest({
      .code = {op::SLOAD},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 900,
      .stack_before = {16},
      .stack_after = {32},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SLOAD_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, get_storage(evmc::address(0x42), evmc::bytes32(16)))  //
      .Times(1)
      .WillOnce(Return(evmc::bytes32(32)));

  RunInterpreterTest({
      .code = {op::SLOAD},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2900,
      .stack_before = {16},
      .stack_after = {32},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SLOAD_OutOfGas_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_COLD));

  RunInterpreterTest({
      .code = {op::SLOAD},
      .state_after = RunState::kErrorGas,
      .gas_before = 2000,
      .stack_before = {16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SLOAD_OutOfGas_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_WARM));

  RunInterpreterTest({
      .code = {op::SLOAD},
      .state_after = RunState::kErrorGas,
      .gas_before = 90,
      .stack_before = {16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SLOAD_StackError) {
  RunInterpreterTest({
      .code = {op::SLOAD},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 2200,
  });
}

///////////////////////////////////////////////////////////
// SSTORE
TEST(InterpreterTest, SSTORE) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_ASSIGNED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2200,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_StorageAdded) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_ADDED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 21000,
      .gas_after = 1000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_StorageModified) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_MODIFIED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 6000,
      .gas_after = 1000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_StorageDeleted) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_DELETED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 6000,
      .gas_after = 1000,
      .gas_refund_after = 15000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_OutOfGas) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_MODIFIED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kErrorGas,
      .gas_before = 2700,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_OutOfGas_EIP2200) {
  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kErrorGas,
      .gas_before = 2300,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
  });
}

TEST(InterpreterTest, SSTORE_StackError) {
  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 3000,
      .stack_before = {0xFF},
  });
}

TEST(InterpreterTest, SSTORE_BerlinRevision) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_ASSIGNED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3300,
      .gas_after = 1100,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SSTORE_BerlinRevision_StorageModified) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_MODIFIED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 100,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SSTORE_BerlinRevision_StorageDeleted) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_DELETED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 100,
      .gas_refund_after = 15000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageDeleted) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_DELETED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 6000,
      .gas_after = 1000,
      .gas_refund_after = 15000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageDeletedAdded) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .WillRepeatedly(Return(EVMC_STORAGE_DELETED_ADDED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2200,
      .gas_refund_before = 20000,
      .gas_refund_after = 5000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2200,
      .gas_refund_before = 10000,
      .gas_refund_after = -5000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageModifiedDeleted) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_MODIFIED_DELETED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2200,
      .gas_refund_after = 15000,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageAddedDeleted) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_ADDED_DELETED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2200,
      .gas_refund_after = 19200,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageModifiedRestored) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_MODIFIED_RESTORED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2200,
      .gas_refund_after = 4200,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageModifiedRestored_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_MODIFIED_RESTORED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3300,
      .gas_after = 1100,
      .gas_refund_after = 4900,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageModifiedRestored_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_MODIFIED_RESTORED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2900,
      .gas_refund_after = 2800,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageDeletedRestored) {
  MockHost host;
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_DELETED_RESTORED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2200,
      .gas_refund_after = -10800,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageDeletedRestored_Cold) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_DELETED_RESTORED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3300,
      .gas_after = 1100,
      .gas_refund_after = -10100,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SSTORE_Refund_StorageDeletedRestored_Warm) {
  MockHost host;
  EXPECT_CALL(host, access_storage(evmc::address(0x42), evmc::bytes32(16))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, set_storage(evmc::address(0x42), evmc::bytes32(16), evmc::bytes32(32)))  //
      .Times(1)
      .WillOnce(Return(EVMC_STORAGE_DELETED_RESTORED));

  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kDone,
      .gas_before = 3000,
      .gas_after = 2900,
      .gas_refund_after = -12200,
      .stack_before = {32, 16},
      .message = {.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SSTORE_StaticCallViolation) {
  RunInterpreterTest({
      .code = {op::SSTORE},
      .state_after = RunState::kErrorStaticCall,
      .is_static_call = true,
      .gas_before = 3000,
      .stack_before = {32, 16},
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

TEST(InterpreterTest, JUMP_OutOfCode) {
  RunInterpreterTest({
      .code = {op::JUMP},
      .state_after = RunState::kErrorJump,
      .gas_before = 5000,
      .stack_before = {3},
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
      .state_after = RunState::kErrorStackUnderflow,
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

TEST(InterpreterTest, JUMPI_LargeCondition) {
  RunInterpreterTest({
      .code = {op::JUMPI,      //
               op::PUSH1, 24,  //
               op::JUMPDEST,   //
               op::PUSH1, 42},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 4986,
      .stack_before = {uint256_t(1) << 80, 3},
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
      .state_after = RunState::kErrorStackUnderflow,
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

  // Automatically expands to multiples of 32.
  RunInterpreterTest({
      .code = {op::MSIZE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {32},
      .memory_before{0, 0, 0, 0, 0, 0, 0, 0},
      .memory_after{0, 0, 0, 0, 0, 0, 0, 0},
  });

  // Automatically expands to multiples of 32.
  RunInterpreterTest({
      .code = {op::MSIZE},
      .state_after = RunState::kDone,
      .gas_before = 10,
      .gas_after = 8,
      .stack_after = {64},
      .memory_before{
          0, 0, 0, 0, 0, 0, 0, 0,  //
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
      .state_after = RunState::kErrorStackUnderflow,
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 10,
      .stack_before = {4, 3, 2, 1},
  });
}

///////////////////////////////////////////////////////////
// LOG
TEST(InterpreterTest, LOG0) {
  MockHost host;
  EXPECT_CALL(host, emit_log(evmc::address(0x42), _, _, _, 0))  //
      .With(Args<1, 2>(ElementsAre(0x0B, 0x0C, 0x0D)))
      .Times(1);

  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kDone,
      .gas_before = 400,
      .gas_after = 1,
      .stack_before = {3, 1},
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, LOG3) {
  using evmc::bytes32;

  MockHost host;
  EXPECT_CALL(host, emit_log(evmc::address(0x42), _, _, _, _))
      .With(AllOf(Args<1, 2>(ElementsAre(0x0B, 0x0C, 0x0D)),
                  Args<3, 4>(ElementsAre(bytes32{0xF1}, bytes32{0xF2}, bytes32{0xF3}))))
      .Times(1);

  RunInterpreterTest({
      .code = {op::LOG3},
      .state_after = RunState::kDone,
      .gas_before = 1524,
      .gas_after = 0,
      .stack_before = {0xF3, 0xF2, 0xF1, 3, 1},
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, LOG0_ZeroSize) {
  MockHost host;
  EXPECT_CALL(host, emit_log(evmc::address(0x42), _, _, _, 0))  //
      .With(Args<1, 2>(ElementsAre()))
      .Times(1);

  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kDone,
      .gas_before = 420,
      .gas_after = 45,
      .stack_before = {0, 1},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, LOG0_GrowMemory) {
  MockHost host;
  EXPECT_CALL(host, emit_log(evmc::address(0x42), _, _, _, 0))  //
      .With(Args<1, 2>(ElementsAre(0x0B, 0x0C, 0x0D, 0, 0)))
      .Times(1);

  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kDone,
      .gas_before = 420,
      .gas_after = 5,
      .stack_before = {5, 1},
      .memory_before{0x0A, 0x0B, 0x0C, 0x0D},
      .memory_after{0x0A, 0x0B, 0x0C, 0x0D, 0, 0},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, LOG0_OutOfGas_Static) {
  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kErrorGas,
      .gas_before = 350,
      .stack_before = {5, 1},
  });
}

TEST(InterpreterTest, LOG0_OutOfGas_Dynamic) {
  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kErrorGas,
      .gas_before = 400,
      .stack_before = {5, 1},
  });
}

TEST(InterpreterTest, LOG0_StackError) {
  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 1000,
      .stack_before = {5},
  });
}

TEST(InterpreterTest, LOG0_StaticCallViolation) {
  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kErrorStaticCall,
      .is_static_call = true,
      .gas_before = 400,
      .stack_before = {3, 1},
  });
}

TEST(InterpreterTest, LOG0_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100, 0},
  });

  RunInterpreterTest({
      .code = {op::LOG0},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {1, uint256_t{1} << 100},
  });
}

///////////////////////////////////////////////////////////
// RETURN
TEST(InterpreterTest, RETURN) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kReturn,
      .gas_before = 10,
      .gas_after = 10,
      .stack_before = {2, 1},
      .memory_before = {0xAA, 0xBB, 0xCC},
      .memory_after = {0xAA, 0xBB, 0xCC},
      .return_data = {0xBB, 0xCC},
  });
}

TEST(InterpreterTest, RETURN_ZeroSize) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kReturn,
      .gas_before = 10,
      .gas_after = 10,
      .stack_before = {0, 1},
  });
}

TEST(InterpreterTest, RETURN_GrowMemory) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kReturn,
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 10,
      .stack_before = {3},
  });
}

TEST(InterpreterTest, RETURN_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100, 0},
  });

  RunInterpreterTest({
      .code = {op::RETURN},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {1, uint256_t{1} << 100},
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
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 10,
      .stack_before = {3},
  });
}

TEST(InterpreterTest, REVERT_OversizedMemory) {
  RunInterpreterTest({
      .code = {op::REVERT},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {uint256_t{1} << 100, 0},
  });

  RunInterpreterTest({
      .code = {op::REVERT},
      .state_after = RunState::kErrorGas,
      .gas_before = 10000000,
      .stack_before = {1, uint256_t{1} << 100},
  });
}

///////////////////////////////////////////////////////////
// INVALID
TEST(InterpreterTest, INVALID) {
  RunInterpreterTest({
      .code = {op::INVALID},
      .state_after = RunState::kInvalid,
      .gas_before = 10,
  });
}

///////////////////////////////////////////////////////////
// SELFDESTRUCT
TEST(InterpreterTest, SELFDESTRUCT) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42))).WillRepeatedly(Return(evmc::uint256be(1)));
  EXPECT_CALL(host, account_exists(evmc::address(0x43))).WillRepeatedly(Return(true));
  EXPECT_CALL(host, selfdestruct(evmc::address(0x42), evmc::address(0x43)))  //
      .Times(1)
      .WillOnce(Return(true));

  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 0,
      .gas_refund_after = 24000,
      .stack_before = {0x43},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_AccountNotExisting) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42))).WillRepeatedly(Return(evmc::uint256be(1)));
  EXPECT_CALL(host, account_exists(evmc::address(0x43))).WillRepeatedly(Return(false));
  EXPECT_CALL(host, selfdestruct(evmc::address(0x42), evmc::address(0x43)))  //
      .Times(1)
      .WillOnce(Return(true));

  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kDone,
      .gas_before = 30000,
      .gas_after = 0,
      .gas_refund_after = 24000,
      .stack_before = {0x43},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_AccountNotExisting_ButNoValueSent) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42))).WillRepeatedly(Return(evmc::uint256be(0)));
  EXPECT_CALL(host, account_exists(evmc::address(0x43))).WillRepeatedly(Return(false));
  EXPECT_CALL(host, selfdestruct(evmc::address(0x42), evmc::address(0x43)))  //
      .Times(1)
      .WillOnce(Return(true));

  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 0,
      .gas_refund_after = 24000,
      .stack_before = {0x43},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_NoRefund) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42))).WillRepeatedly(Return(evmc::uint256be(1)));
  EXPECT_CALL(host, account_exists(evmc::address(0x43))).WillRepeatedly(Return(true));
  EXPECT_CALL(host, selfdestruct(evmc::address(0x42), evmc::address(0x43)))  //
      .Times(1)
      .WillOnce(Return(false));

  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 0,
      .stack_before = {0x43},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_BerlinRevision_Cold) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42))).WillRepeatedly(Return(evmc::uint256be(1)));
  EXPECT_CALL(host, account_exists(evmc::address(0x43))).WillRepeatedly(Return(true));
  EXPECT_CALL(host, access_account(evmc::address(0x43))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, selfdestruct(evmc::address(0x42), evmc::address(0x43)))  //
      .Times(1)
      .WillOnce(Return(true));

  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kDone,
      .gas_before = 7600,
      .gas_after = 0,
      .gas_refund_after = 24000,
      .stack_before = {0x43},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_BerlinRevision_Warm) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42))).WillRepeatedly(Return(evmc::uint256be(1)));
  EXPECT_CALL(host, account_exists(evmc::address(0x43))).WillRepeatedly(Return(true));
  EXPECT_CALL(host, access_account(evmc::address(0x43))).WillRepeatedly(Return(EVMC_ACCESS_WARM));
  EXPECT_CALL(host, selfdestruct(evmc::address(0x42), evmc::address(0x43)))  //
      .Times(1)
      .WillOnce(Return(true));

  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kDone,
      .gas_before = 5000,
      .gas_after = 0,
      .gas_refund_after = 24000,
      .stack_before = {0x43},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_BERLIN,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_LondonRevision_NoRefund) {
  MockHost host;
  EXPECT_CALL(host, get_balance(evmc::address(0x42))).WillRepeatedly(Return(evmc::uint256be(1)));
  EXPECT_CALL(host, account_exists(evmc::address(0x43))).WillRepeatedly(Return(true));
  EXPECT_CALL(host, access_account(evmc::address(0x43))).WillRepeatedly(Return(EVMC_ACCESS_COLD));
  EXPECT_CALL(host, selfdestruct(evmc::address(0x42), evmc::address(0x43)))  //
      .Times(1)
      .WillOnce(Return(true));

  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kDone,
      .gas_before = 7600,
      .gas_after = 0,
      .stack_before = {0x43},
      .message{.recipient = evmc::address(0x42)},
      .host = &host,
      .revision = EVMC_LONDON,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_OutOfGas) {
  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kErrorGas,
      .gas_before = 4000,
      .stack_before = {0x43},
  });
}

TEST(InterpreterTest, SELFDESTRUCT_StackError) {
  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kErrorStackUnderflow,
      .gas_before = 5000,
  });
}

TEST(InterpreterTest, SELFDESTRUCT_StaticCallViolation) {
  RunInterpreterTest({
      .code = {op::SELFDESTRUCT},
      .state_after = RunState::kErrorStaticCall,
      .is_static_call = true,
      .gas_before = 5000,
      .stack_before = {0x43},
  });
}

}  // namespace
}  // namespace tosca::evmzero

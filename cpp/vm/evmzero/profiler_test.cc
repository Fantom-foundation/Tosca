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

#include "vm/evmzero/profiler.h"

#include <algorithm>
#include <functional>
#include <map>
#include <set>
#include <thread>

#include <gtest/gtest.h>

namespace tosca::evmzero {
namespace {

using namespace std::chrono_literals;

using NumCallsExpectation = std::function<void(std::uint64_t)>;
using NumCallsExpectations = std::map<op::OpCode, NumCallsExpectation>;
using TotalTimeExpectation = std::function<void(std::chrono::nanoseconds)>;
using TotalTimeExpectations = std::map<op::OpCode, TotalTimeExpectation>;
using TotalTicksExpectation = std::function<void(std::uint64_t)>;
using TotalTicksExpectations = std::map<op::OpCode, TotalTicksExpectation>;
using RelativeTimeExpectation = std::function<void(std::uint64_t, std::uint64_t)>;
using RelativeTimeExpectations = std::map<std::pair<op::OpCode, op::OpCode>, RelativeTimeExpectation>;
template <ProfilerMode Mode = ProfilerMode::kFull>
using InterpreterExpectation = std::function<void(typename Profile<Mode>::Stats)>;

template <ProfilerMode Mode = ProfilerMode::kFull>
void FillProfile(Profiler<Mode>& profiler, int call_depth = 0) {
  auto msg = evmc_message{.depth = call_depth};
  const auto args = InterpreterArgs{.message = &msg};
  const auto ctx = internal::Context{};

  profiler.PreRun(args);
  for (std::size_t i = 0; i < op::kNumUsedAndUnusedOpCodes; ++i) {
    const auto opcode = static_cast<op::OpCode>(i);
    if (op::IsUsedOpCode(opcode) && !op::IsCallOpCode(opcode)) {
      profiler.PreInstruction(opcode, ctx);
      profiler.PostInstruction(opcode, ctx);
    }
  }
  profiler.PostRun(args);
}

template <typename T>
auto ExpectAllMarkers(std::function<void(T)> expectation) {
  std::map<op::OpCode, std::function<void(T)>> expectations;
  for (std::size_t i = 0; i < op::kNumUsedAndUnusedOpCodes; ++i) {
    const auto opcode = static_cast<op::OpCode>(i);
    if (op::IsUsedOpCode(opcode) && !op::IsCallOpCode(opcode)) {
      expectations[opcode] = expectation;
    }
  }
  return expectations;
}
auto ExpectAllNumCalls(std::function<void(std::uint64_t)> expectation) { return ExpectAllMarkers(expectation); }
auto ExpectAllTotalTimes(std::function<void(std::chrono::nanoseconds)> expectation) {
  return ExpectAllMarkers(expectation);
}
auto ExpectAllTotalTicks(std::function<void(std::uint64_t)> expectation) { return ExpectAllMarkers(expectation); }

template <typename T>
auto ExpectAllMarkersEmpty() {
  return ExpectAllMarkers<T>([](T value) { EXPECT_EQ(value, T(0)); });
}
auto ExpectAllNumCallsEmpty() { return ExpectAllMarkersEmpty<std::uint64_t>(); }
auto ExpectAllTotalTimesEmpty() { return ExpectAllMarkersEmpty<std::chrono::nanoseconds>(); }
auto ExpectAllTotalTicksEmpty() { return ExpectAllMarkersEmpty<std::uint64_t>(); }

template <ProfilerMode Mode = ProfilerMode::kFull>
auto ExpectInterpreterEmpty() {
  return [](typename Profile<Mode>::Stats stats) {
    EXPECT_EQ(stats.num_calls, 0);
    EXPECT_EQ(stats.total_ticks, 0);
    EXPECT_EQ(stats.total_time, 0ns);
  };
}

template <ProfilerMode Mode = ProfilerMode::kFull>
auto ExpectInterpreterCalled(std::uint64_t num_calls) {
  return [num_calls](typename Profile<Mode>::Stats stats) {
    EXPECT_EQ(stats.num_calls, num_calls);
    EXPECT_GT(stats.total_ticks, 0);
    EXPECT_GT(stats.total_time, 0ns);
  };
}

template <ProfilerMode Mode = ProfilerMode::kFull>
struct ProfilerTestDescription {
  using TestFunction = std::function<void(Profiler<Mode>&)>;

  Profile<Mode> initial_profile;
  TestFunction test;
  NumCallsExpectations num_calls_before;
  NumCallsExpectations num_calls_after;
  TotalTimeExpectations total_time_before;
  TotalTimeExpectations total_time_after;
  TotalTicksExpectations total_ticks_before;
  TotalTicksExpectations total_ticks_after;
  RelativeTimeExpectations relative_time_before;
  RelativeTimeExpectations relative_time_after;
  InterpreterExpectation<Mode> interpreter_before;
  InterpreterExpectation<Mode> interpreter_after;
};

template <ProfilerMode Mode = ProfilerMode::kFull>
void RunProfilerTest(const ProfilerTestDescription<Mode>& desc) {
  auto profiler = Profiler<Mode>(desc.initial_profile);

  // Check preconditions.
  const auto profile_before = profiler.Collect();
  auto untouched_call_opcodes = std::set<op::OpCode>{};
  for (const auto& [op, expect] : desc.num_calls_before) {
    untouched_call_opcodes.insert(op);
    expect(profile_before.GetInstructionStats(op).num_calls);
  }
  auto untouched_time_opcodes = std::set<op::OpCode>{};
  for (const auto& [op, expect] : desc.total_time_before) {
    untouched_time_opcodes.insert(op);
    expect(profile_before.GetInstructionStats(op).total_time);
  }
  auto untouched_total_ticks_opcodes = std::set<op::OpCode>{};
  for (const auto& [op, expect] : desc.total_ticks_before) {
    untouched_total_ticks_opcodes.insert(op);
    expect(profile_before.GetInstructionStats(op).total_ticks);
  }
  for (const auto& [ops, expect] : desc.relative_time_before) {
    expect(profile_before.GetInstructionStats(ops.first).total_ticks,
           profile_before.GetInstructionStats(ops.second).total_ticks);
  }
  desc.interpreter_before(profile_before.GetInterpreterStats());

  // Execute test function.
  if (desc.test) {
    desc.test(profiler);
  }

  // Check postconditions.
  const auto profile_after = profiler.Collect();

  for (const auto& [op, expect] : desc.num_calls_after) {
    untouched_call_opcodes.erase(op);
    expect(profile_after.GetInstructionStats(op).num_calls);
  }
  for (const auto& [op, expect] : desc.total_time_after) {
    untouched_time_opcodes.erase(op);
    expect(profile_after.GetInstructionStats(op).total_time);
  }
  for (const auto& [op, expect] : desc.total_ticks_after) {
    untouched_total_ticks_opcodes.erase(op);
    expect(profile_after.GetInstructionStats(op).total_ticks);
  }
  for (const auto& [ops, expect] : desc.relative_time_after) {
    expect(profile_after.GetInstructionStats(ops.first).total_ticks,
           profile_after.GetInstructionStats(ops.second).total_ticks);
  }
  desc.interpreter_after(profile_after.GetInterpreterStats());

  // Check that all opcodes that don't have explicit postconditions still satisfy the preconditions.
  for (const auto& op : untouched_call_opcodes) {
    const auto& expect = desc.num_calls_before.at(op);
    expect(profile_after.GetInstructionStats(op).num_calls);
  }
  for (const auto& op : untouched_time_opcodes) {
    const auto& expect = desc.total_time_before.at(op);
    expect(profile_after.GetInstructionStats(op).total_time);
  }
  for (const auto& op : untouched_total_ticks_opcodes) {
    const auto& expect = desc.total_ticks_before.at(op);
    expect(profile_after.GetInstructionStats(op).total_ticks);
  }
}

TEST(ProfilerTest, Empty) {
  RunProfilerTest({
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, Collects) {
  RunProfilerTest({
      .test = [](auto& profiler) { FillProfile(profiler); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterCalled(1),
  });
}

TEST(ProfilerTest, ConstructFromProfile) {
  auto profiler = Profiler<ProfilerMode::kFull>{};
  FillProfile(profiler);

  RunProfilerTest({
      .initial_profile = profiler.Collect(),
      .num_calls_before = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .total_time_before = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .interpreter_before = ExpectInterpreterCalled(1),
      .interpreter_after = ExpectInterpreterCalled(1),
  });
}

TEST(ProfilerTest, ResetClears) {
  auto profiler = Profiler<ProfilerMode::kFull>{};
  FillProfile(profiler);

  RunProfilerTest({
      .initial_profile = profiler.Collect(),
      .test = [](auto& profiler) { profiler.Reset(); },
      .num_calls_before = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
      .interpreter_before = ExpectInterpreterCalled(1),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, MergeMerges) {
  auto init_profiler = Profiler<ProfilerMode::kFull>{};
  FillProfile(init_profiler);

  RunProfilerTest({
      .initial_profile = init_profiler.Collect(),
      .test = [&init_profiler](auto& profiler) { profiler.Merge(init_profiler.Collect()); },
      .num_calls_before = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 2); }),
      .total_time_before = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .interpreter_before = ExpectInterpreterCalled(1),
      .interpreter_after = ExpectInterpreterCalled(2),
  });
}

TEST(ProfilerTest, NestedInterpreterRemainsEmpty) {
  RunProfilerTest({
      .test = [](auto& profiler) { FillProfile(profiler, 1); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, SingleMarker) {
  RunProfilerTest({
      .test =
          [](auto& profiler) {
            const auto ctx = internal::Context{};
            profiler.PreInstruction(op::OpCode::ADD, ctx);
            profiler.PostInstruction(op::OpCode::ADD, ctx);
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = {{op::OpCode::ADD, [](auto value) { EXPECT_EQ(value, 1); }}},
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = {{op::OpCode::ADD, [](auto value) { EXPECT_GT(value, 0ns); }}},
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = {{op::OpCode::ADD, [](auto value) { EXPECT_GT(value, 0); }}},
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, UnmatchedStartMarker) {
  RunProfilerTest({
      .test = [](auto& profiler) { profiler.PreInstruction(op::OpCode::MUL, internal::Context{}); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, UnmatchedEndMarker) {
  RunProfilerTest({
      .test = [](auto& profiler) { profiler.PostInstruction(op::OpCode::INVALID, internal::Context{}); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = {{op::OpCode::INVALID, [](auto value) { EXPECT_EQ(value, 1); }}},
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = {{op::OpCode::INVALID, [](auto value) { EXPECT_GT(value, 0ns); }}},
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = {{op::OpCode::INVALID, [](auto value) { EXPECT_GT(value, 0); }}},
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, MultipleMarkers) {
  RunProfilerTest({
      .test =
          [](auto& profiler) {
            const auto ctx = internal::Context{};
            profiler.PreInstruction(op::OpCode::ADD, ctx);
            profiler.PostInstruction(op::OpCode::ADD, ctx);
            profiler.PreInstruction(op::OpCode::PUSH1, ctx);
            profiler.PostInstruction(op::OpCode::PUSH1, ctx);
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {op::OpCode::ADD, [](auto value) { EXPECT_EQ(value, 1); }},
              {op::OpCode::PUSH1, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {op::OpCode::ADD, [](auto value) { EXPECT_GT(value, 0ns); }},
              {op::OpCode::PUSH1, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {op::OpCode::ADD, [](auto value) { EXPECT_GT(value, 0); }},
              {op::OpCode::PUSH1, [](auto value) { EXPECT_GT(value, 0); }},
          },
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, NestedMarkers) {
  RunProfilerTest({
      .test =
          [](auto& profiler) {
            const auto ctx = internal::Context{};
            profiler.PreInstruction(op::OpCode::SHA3, ctx);
            profiler.PreInstruction(op::OpCode::SUB, ctx);
            profiler.PostInstruction(op::OpCode::SUB, ctx);
            profiler.PostInstruction(op::OpCode::SHA3, ctx);
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {op::OpCode::SHA3, [](auto value) { EXPECT_EQ(value, 1); }},
              {op::OpCode::SUB, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {op::OpCode::SHA3, [](auto value) { EXPECT_GT(value, 0ns); }},
              {op::OpCode::SUB, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {op::OpCode::SHA3, [](auto value) { EXPECT_GT(value, 0); }},
              {op::OpCode::SUB, [](auto value) { EXPECT_GT(value, 0); }},
          },
      .relative_time_before = {{{op::OpCode::SHA3, op::OpCode::SUB},
                                [](auto first, auto second) {
                                  EXPECT_EQ(first, 0);
                                  EXPECT_EQ(second, 0);
                                }}},
      .relative_time_after = {{{op::OpCode::SHA3, op::OpCode::SUB},
                               [](auto first, auto second) { EXPECT_LE(second, first); }}},
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerTest, WallClockAccurate) {
  std::chrono::nanoseconds elapsed_time;
  RunProfilerTest({
      .test =
          [&elapsed_time](auto& profiler) {
            const auto ctx = internal::Context{};
            const auto start = std::chrono::high_resolution_clock::now();
            profiler.PreInstruction(op::OpCode::SHA3, ctx);
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            profiler.PostInstruction(op::OpCode::SHA3, ctx);
            const auto end = std::chrono::high_resolution_clock::now();
            elapsed_time = std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = {{op::OpCode::SHA3, [](auto value) { EXPECT_EQ(value, 1); }}},
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = {{op::OpCode::SHA3,
                            [&elapsed_time](auto value) {
                              const auto value_count = static_cast<double>(value.count());
                              const auto elapsed_time_count = static_cast<double>(elapsed_time.count());
                              constexpr auto max_error =
                                  std::chrono::duration_cast<std::chrono::nanoseconds>(5us).count();
                              EXPECT_NEAR(value_count, elapsed_time_count, max_error);
                            }}},
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = {{op::OpCode::SHA3, [](auto value) { EXPECT_GT(value, 0); }}},
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterEmpty(),
  });
}

TEST(ProfilerExternalTest, InternalMarkersIgnored) {
  RunProfilerTest<ProfilerMode::kExternal>({
      .test =
          [](auto& profiler) {
            const auto ctx = internal::Context{};
            profiler.PreInstruction(op::OpCode::ADD, ctx);
            profiler.PostInstruction(op::OpCode::ADD, ctx);
            profiler.PreInstruction(op::OpCode::SLOAD, ctx);
            profiler.PostInstruction(op::OpCode::SLOAD, ctx);
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {op::OpCode::ADD, [](auto value) { EXPECT_EQ(value, 0); }},
              {op::OpCode::SLOAD, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {op::OpCode::ADD, [](auto value) { EXPECT_EQ(value, 0ns); }},
              {op::OpCode::SLOAD, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {op::OpCode::ADD, [](auto value) { EXPECT_EQ(value, 0); }},
              {op::OpCode::SLOAD, [](auto value) { EXPECT_GT(value, 0); }},
          },
      .interpreter_before = ExpectInterpreterEmpty<ProfilerMode::kExternal>(),
      .interpreter_after = ExpectInterpreterEmpty<ProfilerMode::kExternal>(),
  });
}

TEST(ProfilerTest, RecursiveCalls) {
  RunProfilerTest({
      .test =
          [](auto& profiler) {
            const auto call_depth = 0;
            const auto msg = evmc_message{.depth = call_depth};
            const auto args = InterpreterArgs{.message = &msg};
            const auto ctx = internal::Context{.message = &msg};

            profiler.PreRun(args);
            profiler.PreInstruction(op::CALL, ctx);

            const auto call_msg = evmc_message{.depth = msg.depth + 1};
            const auto call_args = InterpreterArgs{.message = &call_msg};
            const auto call_ctx = internal::Context{.message = &call_msg};
            profiler.PreRun(call_args);
            profiler.PreInstruction(op::ADD, call_ctx);
            profiler.PostInstruction(op::ADD, call_ctx);
            profiler.PostRun(call_args);

            profiler.PostInstruction(op::CALL, ctx);
            profiler.PostRun(args);
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {op::OpCode::CALL, [](auto value) { EXPECT_EQ(value, 1); }},
              {op::OpCode::ADD, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {op::OpCode::CALL, [](auto value) { EXPECT_GT(value, 0ns); }},
              {op::OpCode::ADD, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {op::OpCode::CALL, [](auto value) { EXPECT_GT(value, 0); }},
              {op::OpCode::ADD, [](auto value) { EXPECT_GT(value, 0); }},
          },
      .relative_time_before = {{{op::OpCode::CALL, op::OpCode::ADD},
                                [](auto first, auto second) {
                                  EXPECT_EQ(first, 0);
                                  EXPECT_EQ(second, 0);
                                }}},
      .relative_time_after = {{{op::OpCode::CALL, op::OpCode::ADD},
                               [](auto first, auto second) { EXPECT_LE(second, first); }}},
      .interpreter_before = ExpectInterpreterEmpty(),
      .interpreter_after = ExpectInterpreterCalled(1),
  });
}

}  // namespace
}  // namespace tosca::evmzero

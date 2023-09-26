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
using NumCallsExpectations = std::map<Marker, NumCallsExpectation>;
using TotalTimeExpectation = std::function<void(std::chrono::nanoseconds)>;
using TotalTimeExpectations = std::map<Marker, TotalTimeExpectation>;
using TotalTicksExpectation = std::function<void(std::uint64_t)>;
using TotalTicksExpectations = std::map<Marker, TotalTicksExpectation>;
using RelativeTimeExpectation = std::function<void(std::uint64_t, std::uint64_t)>;
using RelativeTimeExpectations = std::map<std::pair<Marker, Marker>, RelativeTimeExpectation>;

template <bool EnabledProfiler>
void FillProfile(Profiler<EnabledProfiler>& profiler) {
#define EVMZERO_PROFILER_MARKER(name)                  \
  if constexpr (Marker::name != Marker::NUM_MARKERS) { \
    profiler.template Start<Marker::name>();           \
    profiler.template End<Marker::name>();             \
  }
#include "profiler_markers.inc"
}

template <bool EnabledProfiler>
void FillProfileScoped(Profiler<EnabledProfiler>& profiler) {
#define EVMZERO_PROFILER_MARKER(name)                        \
  if constexpr (Marker::name != Marker::NUM_MARKERS) {       \
    const auto _ = profiler.template Scoped<Marker::name>(); \
  }
#include "profiler_markers.inc"
}

template <typename T>
auto ExpectAllMarkers(std::function<void(T)> expectation) {
  std::map<Marker, std::function<void(T)>> expectations;
  for (std::size_t i = 0; i < Profile::kNumMarkers; ++i) {
    expectations[static_cast<Marker>(i)] = expectation;
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

template <bool EnabledProfiler>
struct ProfilerTestDescription {
  using TestFunction = std::function<void(Profiler<EnabledProfiler>&)>;

  Profile initial_profile;
  TestFunction test;
  NumCallsExpectations num_calls_before;
  NumCallsExpectations num_calls_after;
  TotalTimeExpectations total_time_before;
  TotalTimeExpectations total_time_after;
  TotalTicksExpectations total_ticks_before;
  TotalTicksExpectations total_ticks_after;
  RelativeTimeExpectations relative_time_before;
  RelativeTimeExpectations relative_time_after;
};

template <bool EnabledProfiler>
void RunProfilerTest(const ProfilerTestDescription<EnabledProfiler>& desc) {
  auto profiler = Profiler<EnabledProfiler>(desc.initial_profile);

  // Check preconditions.
  const auto profile_before = profiler.Collect();
  auto untouched_call_markers = std::set<Marker>{};
  for (const auto& [marker, expect] : desc.num_calls_before) {
    untouched_call_markers.insert(marker);
    expect(profile_before.GetNumCalls(marker));
  }
  auto untouched_time_markers = std::set<Marker>{};
  for (const auto& [marker, expect] : desc.total_time_before) {
    untouched_time_markers.insert(marker);
    expect(profile_before.GetTotalTime(marker));
  }
  auto untouched_total_ticks_markers = std::set<Marker>{};
  for (const auto& [marker, expect] : desc.total_ticks_before) {
    untouched_total_ticks_markers.insert(marker);
    expect(profile_before.GetTotalTicks(marker));
  }
  for (const auto& [markers, expect] : desc.relative_time_before) {
    expect(profile_before.GetTotalTicks(markers.first), profile_before.GetTotalTicks(markers.second));
  }

  // Execute test function.
  if (desc.test) {
    desc.test(profiler);
  }

  // Check postconditions.
  const auto profile_after = profiler.Collect();

  for (const auto& [marker, expect] : desc.num_calls_after) {
    untouched_call_markers.erase(marker);
    expect(profile_after.GetNumCalls(marker));
  }
  for (const auto& [marker, expect] : desc.total_time_after) {
    untouched_time_markers.erase(marker);
    expect(profile_after.GetTotalTime(marker));
  }
  for (const auto& [marker, expect] : desc.total_ticks_after) {
    untouched_total_ticks_markers.erase(marker);
    expect(profile_after.GetTotalTicks(marker));
  }
  for (const auto& [markers, expect] : desc.relative_time_after) {
    expect(profile_after.GetTotalTicks(markers.first), profile_after.GetTotalTicks(markers.second));
  }

  // Check that all markers that don't have explicit postconditions still satisfy the preconditions.
  for (const auto& marker : untouched_call_markers) {
    const auto& expect = desc.num_calls_before.at(marker);
    expect(profile_after.GetNumCalls(marker));
  }
  for (const auto& marker : untouched_time_markers) {
    const auto& expect = desc.total_time_before.at(marker);
    expect(profile_after.GetTotalTime(marker));
  }
  for (const auto& marker : untouched_total_ticks_markers) {
    const auto& expect = desc.total_ticks_before.at(marker);
    expect(profile_after.GetTotalTicks(marker));
  }
}

TEST(EnabledProfilerTest, Empty) {
  RunProfilerTest<true>({
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(DisabledProfilerTest, Empty) {
  RunProfilerTest<false>({
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, Collects) {
  RunProfilerTest<true>({
      .test = FillProfile<true>,
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
  });
}

TEST(DisabledProfilerTest, DoesNotCollect) {
  RunProfilerTest<false>({
      .test = FillProfile<false>,
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, ScopedCollects) {
  RunProfilerTest<true>({
      .test = FillProfileScoped<true>,
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
  });
}

TEST(DisabledProfilerTest, ScopedDoesNotCollect) {
  RunProfilerTest<false>({
      .test = FillProfileScoped<false>,
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, ConstructFromProfile) {
  auto profiler = Profiler<true>{};
  FillProfile(profiler);

  RunProfilerTest<true>({
      .initial_profile = profiler.Collect(),
      .num_calls_before = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .total_time_before = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
  });
}

TEST(DisabledProfilerTest, ConstructFromProfile) {
  auto profiler = Profiler<true>{};
  FillProfile(profiler);

  RunProfilerTest<false>({
      .initial_profile = profiler.Collect(),
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, ResetClears) {
  auto profiler = Profiler<true>{};
  FillProfile(profiler);

  RunProfilerTest<true>({
      .initial_profile = profiler.Collect(),
      .test = [](auto& profiler) { profiler.Reset(); },
      .num_calls_before = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(DisabledProfilerTest, ResetRemainsEmpty) {
  auto profiler = Profiler<true>{};
  FillProfile(profiler);

  RunProfilerTest<false>({
      .initial_profile = profiler.Collect(),
      .test = [](auto& profiler) { profiler.Reset(); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, MergeMerges) {
  auto init_profiler = Profiler<true>{};
  FillProfile(init_profiler);

  RunProfilerTest<true>({
      .initial_profile = init_profiler.Collect(),
      .test = [&init_profiler](auto& profiler) { profiler.Merge(init_profiler.Collect()); },
      .num_calls_before = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 1); }),
      .num_calls_after = ExpectAllNumCalls([](auto value) { EXPECT_EQ(value, 2); }),
      .total_time_before = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_time_after = ExpectAllTotalTimes([](auto value) { EXPECT_GT(value, 0ns); }),
      .total_ticks_before = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
      .total_ticks_after = ExpectAllTotalTicks([](auto value) { EXPECT_GT(value, 0); }),
  });
}

TEST(DisabledProfilerTest, MergeDoesNotMerge) {
  auto init_profiler = Profiler<true>{};
  FillProfile(init_profiler);

  RunProfilerTest<false>({
      .initial_profile = init_profiler.Collect(),
      .test = [&init_profiler](auto& profiler) { profiler.Merge(init_profiler.Collect()); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, SingleMarker) {
  RunProfilerTest<true>({
      .test =
          [](auto& profiler) {
            profiler.template Start<Marker::ADD>();
            profiler.template End<Marker::ADD>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = {{Marker::ADD, [](auto value) { EXPECT_EQ(value, 1); }}},
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = {{Marker::ADD, [](auto value) { EXPECT_GT(value, 0ns); }}},
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = {{Marker::ADD, [](auto value) { EXPECT_GT(value, 0); }}},
  });
}

TEST(DisabledProfilerTest, SingleMarker) {
  RunProfilerTest<false>({
      .test =
          [](auto& profiler) {
            profiler.template Start<Marker::ADD>();
            profiler.template End<Marker::ADD>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, SingleMarkerScoped) {
  RunProfilerTest<true>({
      .test = [](auto& profiler) { auto _ = profiler.template Scoped<Marker::AND>(); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = {{Marker::AND, [](auto value) { EXPECT_EQ(value, 1); }}},
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = {{Marker::AND, [](auto value) { EXPECT_GT(value, 0ns); }}},
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = {{Marker::AND, [](auto value) { EXPECT_GT(value, 0); }}},
  });
}

TEST(DisabledProfilerTest, SingleMarkerScoped) {
  RunProfilerTest<false>({
      .test = [](auto& profiler) { auto _ = profiler.template Scoped<Marker::AND>(); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, UnmatchedStartMarker) {
  RunProfilerTest<true>({
      .test = [](auto& profiler) { profiler.template Start<Marker::MUL>(); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(DisabledProfilerTest, UnmatchedStartMarker) {
  RunProfilerTest<false>({
      .test = [](auto& profiler) { profiler.template Start<Marker::MUL>(); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, UnmatchedEndMarker) {
  RunProfilerTest<true>({
      .test = [](auto& profiler) { profiler.template End<Marker::INVALID>(); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = {{Marker::INVALID, [](auto value) { EXPECT_EQ(value, 1); }}},
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = {{Marker::INVALID, [](auto value) { EXPECT_GT(value, 0ns); }}},
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = {{Marker::INVALID, [](auto value) { EXPECT_GT(value, 0); }}},
  });
}

TEST(DisabledProfilerTest, UnmatchedEndMarker) {
  RunProfilerTest<false>({
      .test = [](auto& profiler) { profiler.template End<Marker::INVALID>(); },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, MultipleMarkers) {
  RunProfilerTest<true>({
      .test =
          [](auto& profiler) {
            profiler.template Start<Marker::ADD>();
            profiler.template End<Marker::ADD>();
            profiler.template Start<Marker::PUSH1>();
            profiler.template End<Marker::PUSH1>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {Marker::ADD, [](auto value) { EXPECT_EQ(value, 1); }},
              {Marker::PUSH1, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {Marker::ADD, [](auto value) { EXPECT_GT(value, 0ns); }},
              {Marker::PUSH1, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {Marker::ADD, [](auto value) { EXPECT_GT(value, 0); }},
              {Marker::PUSH1, [](auto value) { EXPECT_GT(value, 0); }},
          },
  });
}

TEST(DisabledProfilerTest, MultipleMarkers) {
  RunProfilerTest<false>({
      .test =
          [](auto& profiler) {
            profiler.template Start<Marker::ADD>();
            profiler.template End<Marker::ADD>();
            profiler.template Start<Marker::PUSH1>();
            profiler.template End<Marker::PUSH1>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, MultipleMarkersScoped) {
  RunProfilerTest<true>({
      .test =
          [](auto& profiler) {
            auto first = profiler.template Scoped<Marker::CREATE2>();
            auto second = profiler.template Scoped<Marker::POP>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {Marker::CREATE2, [](auto value) { EXPECT_EQ(value, 1); }},
              {Marker::POP, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {Marker::CREATE2, [](auto value) { EXPECT_GT(value, 0ns); }},
              {Marker::POP, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {Marker::CREATE2, [](auto value) { EXPECT_GT(value, 0); }},
              {Marker::POP, [](auto value) { EXPECT_GT(value, 0); }},
          },
      .relative_time_before = {{{Marker::CREATE2, Marker::POP},
                                [](auto first, auto second) {
                                  EXPECT_EQ(first, 0);
                                  EXPECT_EQ(second, 0);
                                }}},
      .relative_time_after = {{{Marker::CREATE2, Marker::POP},
                               [](auto first, auto second) { EXPECT_LE(second, first); }}},
  });
}

TEST(DisabledProfilerTest, MultipleMarkersScoped) {
  RunProfilerTest<false>({
      .test =
          [](auto& profiler) {
            auto first = profiler.template Scoped<Marker::CREATE2>();
            auto second = profiler.template Scoped<Marker::POP>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, NestedMarkers) {
  RunProfilerTest<true>({
      .test =
          [](auto& profiler) {
            profiler.template Start<Marker::CALL>();
            profiler.template Start<Marker::SUB>();
            profiler.template End<Marker::SUB>();
            profiler.template End<Marker::CALL>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {Marker::CALL, [](auto value) { EXPECT_EQ(value, 1); }},
              {Marker::SUB, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {Marker::CALL, [](auto value) { EXPECT_GT(value, 0ns); }},
              {Marker::SUB, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {Marker::CALL, [](auto value) { EXPECT_GT(value, 0); }},
              {Marker::SUB, [](auto value) { EXPECT_GT(value, 0); }},
          },
      .relative_time_before = {{{Marker::CALL, Marker::SUB},
                                [](auto first, auto second) {
                                  EXPECT_EQ(first, 0);
                                  EXPECT_EQ(second, 0);
                                }}},
      .relative_time_after = {{{Marker::CALL, Marker::SUB}, [](auto first, auto second) { EXPECT_LE(second, first); }}},
  });
}

TEST(DisabledProfilerTest, NestedMarkers) {
  RunProfilerTest<false>({
      .test =
          [](auto& profiler) {
            profiler.template Start<Marker::CALL>();
            profiler.template Start<Marker::SUB>();
            profiler.template End<Marker::SUB>();
            profiler.template End<Marker::CALL>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, NestedMarkersScoped) {
  RunProfilerTest<true>({
      .test =
          [](auto& profiler) {
            auto _ = profiler.template Scoped<Marker::CREATE>();
            { auto _ = profiler.template Scoped<Marker::SUB>(); }
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after =
          {
              {Marker::CREATE, [](auto value) { EXPECT_EQ(value, 1); }},
              {Marker::SUB, [](auto value) { EXPECT_EQ(value, 1); }},
          },
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after =
          {
              {Marker::CREATE, [](auto value) { EXPECT_GT(value, 0ns); }},
              {Marker::SUB, [](auto value) { EXPECT_GT(value, 0ns); }},
          },
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after =
          {
              {Marker::CREATE, [](auto value) { EXPECT_GT(value, 0); }},
              {Marker::SUB, [](auto value) { EXPECT_GT(value, 0); }},
          },
      .relative_time_before = {{{Marker::CREATE, Marker::SUB},
                                [](auto first, auto second) {
                                  EXPECT_EQ(first, 0);
                                  EXPECT_EQ(second, 0);
                                }}},
      .relative_time_after = {{{Marker::CREATE, Marker::SUB},
                               [](auto first, auto second) { EXPECT_LE(second, first); }}},
  });
}

TEST(DisabledProfilerTest, NestedMarkersScoped) {
  RunProfilerTest<false>({
      .test =
          [](auto& profiler) {
            auto _ = profiler.template Scoped<Marker::CREATE>();
            { auto _ = profiler.template Scoped<Marker::SUB>(); }
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

TEST(EnabledProfilerTest, WallClockAccurate) {
  std::chrono::nanoseconds elapsed_time;
  RunProfilerTest<true>({
      .test =
          [&elapsed_time](auto& profiler) {
            const auto start = std::chrono::high_resolution_clock::now();
            profiler.template Start<Marker::SHA3>();
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            profiler.template End<Marker::SHA3>();
            const auto end = std::chrono::high_resolution_clock::now();
            elapsed_time = std::chrono::duration_cast<std::chrono::nanoseconds>(end - start);
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = {{Marker::SHA3, [](auto value) { EXPECT_EQ(value, 1); }}},
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = {{Marker::SHA3,
                            [&elapsed_time](auto value) {
                              const auto value_count = static_cast<double>(value.count());
                              const auto elapsed_time_count = static_cast<double>(elapsed_time.count());
                              constexpr auto max_error =
                                  std::chrono::duration_cast<std::chrono::nanoseconds>(5us).count();
                              EXPECT_NEAR(value_count, elapsed_time_count, max_error);
                            }}},
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = {{Marker::SHA3, [](auto value) { EXPECT_GT(value, 0); }}},
  });
}

TEST(DisabledProfilerTest, WallClockAccurate) {
  RunProfilerTest<false>({
      .test =
          [](auto& profiler) {
            profiler.template Start<Marker::SHA3>();
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            profiler.template End<Marker::SHA3>();
          },
      .num_calls_before = ExpectAllNumCallsEmpty(),
      .num_calls_after = ExpectAllNumCallsEmpty(),
      .total_time_before = ExpectAllTotalTimesEmpty(),
      .total_time_after = ExpectAllTotalTimesEmpty(),
      .total_ticks_before = ExpectAllTotalTicksEmpty(),
      .total_ticks_after = ExpectAllTotalTicksEmpty(),
  });
}

}  // namespace
}  // namespace tosca::evmzero

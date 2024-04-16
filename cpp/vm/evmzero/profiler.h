//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of 
// this software will be governed by the GNU Lesser General Public Licence v3.
//

#pragma once

#include <array>
#include <chrono>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <type_traits>
#include <utility>

#if defined(__x86_64__)
#include <x86intrin.h>
#endif

#include "build_info.h"
#include "common/assert.h"
#include "common/macros.h"
#include "interpreter.h"
#include "opcodes.h"

#if EVMZERO_TRACY_ENABLED
#include <tracy/Tracy.hpp>
#define EVMZERO_PROFILE_ZONE() ZoneScoped
#define EVMZERO_PROFILE_ZONE_N(name) ZoneScopedN(name)
#define EVMZERO_PROFILE_ALLOC(ptr, size) TracyAlloc(ptr, size)
#define EVMZERO_PROFILE_FREE(ptr) TracyFree(ptr)
#else
#define EVMZERO_PROFILE_ZONE()
#define EVMZERO_PROFILE_ZONE_N(name)
#define EVMZERO_PROFILE_ALLOC(ptr, size)
#define EVMZERO_PROFILE_FREE(ptr)
#endif

namespace tosca::evmzero {

namespace internal {

// Get the current processor time-stamp as reported by the rdtscp x86 instruction.
// The time-stamp is monotonic and gets incremented in a fixed interval regardless of processor frequency.
// It is also serializing meaning all previous instructions must have completed before the time-stamp is read.
// See: https://www.felixcloutier.com/x86/rdtscp
// Falls back to standard library high resolution clock on other architectures.
TOSCA_FORCE_INLINE inline std::uint64_t Now() noexcept {
#if defined(__x86_64__)
  unsigned int _;
  return __rdtscp(&_);
#else
  return static_cast<std::uint64_t>(std::chrono::high_resolution_clock::now().time_since_epoch().count());
#endif
}

// Helper class to convert between processor-time and wall-clock time.
// Converting from processor-time to wall-clock works differently for different CPUs.
// See: https://stackoverflow.com/questions/42189976/calculate-system-time-using-rdtsc
// In order to not have to maintain different implementations for different CPUs this class
// measures the same period of time in both processor-time as well as wall-clock time,
// and uses this as a reference to convert from one to the other.
class TimeConverter {
  struct TimeReference {
    std::uint64_t time_stamp;
    std::chrono::nanoseconds wall_clock;
  };

 public:
  void MarkEnd() noexcept { end_ = {Now(), std::chrono::high_resolution_clock::now().time_since_epoch()}; }

  std::chrono::nanoseconds Convert(std::uint64_t ticks) const noexcept {
    const auto conversion_factor = GetConversionFactor();

    const auto converted_time = static_cast<double>(ticks) * conversion_factor;
    const auto rounded_time = std::chrono::nanoseconds(static_cast<std::uint64_t>(std::round(converted_time)));
    return rounded_time;
  }

 private:
  TimeReference start_ = {Now(), std::chrono::high_resolution_clock::now().time_since_epoch()};
  TimeReference end_;

  double GetConversionFactor() const noexcept {
    const auto time_difference_ticks = end_.time_stamp - start_.time_stamp;
    const auto time_difference_wall_clock = end_.wall_clock.count() - start_.wall_clock.count();

    const auto conversion_factor =
        static_cast<double>(time_difference_wall_clock) / static_cast<double>(time_difference_ticks);
    return conversion_factor;
  }
};

}  // namespace internal

// Enum type used to select the mode of the profiler and its associated profile.
enum class ProfilerMode {
  kFull,
  kExternal,
};

// This type represents collected profiling data. Specialized with the profiler mode to make merging of incompatible
// profiles a compilte-time error.
template <ProfilerMode Mode>
class Profile {
 public:
  struct Stats {
    const std::uint64_t num_calls;
    const std::uint64_t total_ticks;
    const std::chrono::nanoseconds total_time;
  };

  // Print the contained profiling data to stdout or to wherever the env var "EVMZERO_PROFILE_FILE" points to.
  void Dump() const {
    auto out_file = std::ofstream();
    const auto* const profile_file = std::getenv("EVMZERO_PROFILE_FILE");
    if (profile_file) {
      out_file.open(profile_file, std::ios::out | std::ios::app);
    }

    // profiling format: <opcode>, <calls>, <total-time-ticks>, <total-time-nanoseconds>\n
    std::ostream& out = out_file.is_open() ? out_file : std::cout;
    out << "Compiler: " << internal::kCompilerId << " " << internal::kCompilerVersion << "\n";
    out << "Build type: " << internal::kBuildType << "\n";
    out << "Compile definitions: " << internal::kCompileDefinitions << "\n";
    out << "Compile options: " << internal::kCompileOptions << "\n";
    out << "Assertions: " << internal::kAssertions << "\n";
    out << "ASAN: " << internal::kAsan << "\n";
    out << "Mimalloc: " << internal::kMimalloc << "\n";
    out << "Tracy: " << internal::kTracy << "\n";
    out << "opcode,calls,ticks,duration[ns]\n";
    const auto interpreter_stats = GetInterpreterStats();
    out << "INTERPRETER,"                        //
        << interpreter_stats.num_calls << ","    //
        << interpreter_stats.total_ticks << ","  //
        << interpreter_stats.total_time.count() << "\n";
    for (std::size_t i = 0; i < op::kNumUsedAndUnusedOpCodes; ++i) {
      const auto opcode = static_cast<op::OpCode>(i);
      if ((Mode == ProfilerMode::kFull && op::IsUsedOpCode(opcode)) ||
          (Mode == ProfilerMode::kExternal && (op::IsExternalOpCode(opcode) || op::IsCallOpCode(opcode)))) {
        const auto opcode_stats = GetInstructionStats(opcode);
        out << ToString(opcode) << ", "          //
            << opcode_stats.num_calls << ", "    //
            << opcode_stats.total_ticks << ", "  //
            << opcode_stats.total_time.count() << "\n";
      }
    }
    out << std::flush;
  }

  // Merge the contained profile with another profile.
  inline void Merge(const Profile& other) noexcept {
    for (std::size_t i = 0; i < op::kNumUsedAndUnusedOpCodes; ++i) {
      calls_[i] += other.calls_[i];
      total_ticks_[i] += other.total_ticks_[i];
    }
    interpreter_stats_.calls += other.interpreter_stats_.calls;
    interpreter_stats_.total_ticks += other.interpreter_stats_.total_ticks;
  }

  // Reset the contained profiling data.
  inline void Reset() noexcept {
    calls_ = {};
    total_ticks_ = {};
    interpreter_stats_ = {};
  }

  // Get collected profiling statistics for the given opcode.
  inline Stats GetInstructionStats(const op::OpCode opcode) const noexcept {
    return Stats{
        .num_calls = calls_[static_cast<std::size_t>(opcode)],
        .total_ticks = total_ticks_[static_cast<std::size_t>(opcode)],
        .total_time = time_converter_.Convert(total_ticks_[static_cast<std::size_t>(opcode)]),
    };
  }

  // Get collected profiling statistics for the interpreter.
  inline Stats GetInterpreterStats() const noexcept {
    return Stats{
        .num_calls = interpreter_stats_.calls,
        .total_ticks = interpreter_stats_.total_ticks,
        .total_time = time_converter_.Convert(interpreter_stats_.total_ticks),
    };
  }

 private:
  using Data = std::array<std::uint64_t, op::kNumUsedAndUnusedOpCodes>;

  template <ProfilerMode>
  friend class Profiler;

  struct InterpreterStats {
    std::uint64_t calls;
    std::uint64_t total_ticks;
  };

  internal::TimeConverter time_converter_;
  Data calls_ = {};
  Data total_ticks_ = {};
  InterpreterStats interpreter_stats_ = {};

  // Mark the end of a measurement to have a reference with which to convert processor-time to wall-clock time.
  void MarkEnd() noexcept { time_converter_.MarkEnd(); }
};

// This type allows the collection of profiling data through the observer interface.
template <ProfilerMode Mode>
class Profiler {
 public:
  Profiler() noexcept = default;
  Profiler(const Profiler&) = delete;
  Profiler(Profiler&&) = delete;

  Profiler& operator=(const Profiler&) = delete;
  Profiler& operator=(Profiler&&) = delete;

  static constexpr bool uses_context = false;

  // Construct the profiler from an already existing profile.
  explicit Profiler(const Profile<Mode>& profile) noexcept : profile_(profile) {}

  // Start measurement for the given opcode. Must be followed by a PostInstruction call for the same opcode to finish
  // the measurement.
  TOSCA_FORCE_INLINE void PreInstruction(op::OpCode opcode, const internal::Context& ctx) {
    if constexpr (Mode == ProfilerMode::kExternal) {
      if (!op::IsExternalOpCode(opcode) && !op::IsCallOpCode(opcode)) {
        return;
      }
    }

    if (op::IsCallOpCode(opcode)) {
      const auto depth_idx = static_cast<std::size_t>(ctx.message->depth);
      TOSCA_ASSERT(depth_idx < call_data_.size());
      TOSCA_ASSERT(depth_idx + 1 < call_data_.size());
      call_data_[depth_idx].call_start_ticks = internal::Now();
      call_data_[depth_idx + 1].interpreter_start_ticks = kTicksNotMeasured;
    } else {
      const auto opcode_idx = static_cast<std::size_t>(opcode);
      start_ticks_[opcode_idx] = internal::Now();
    }
  }

  // End measurement for the given opcode. Must have been preceded by a PreInstruction call of the same opcode.
  TOSCA_FORCE_INLINE void PostInstruction(op::OpCode opcode, const internal::Context& ctx) {
    if constexpr (Mode == ProfilerMode::kExternal) {
      if (!op::IsExternalOpCode(opcode) && !op::IsCallOpCode(opcode)) {
        return;
      }
    }

    const auto opcode_idx = static_cast<std::size_t>(opcode);
    const auto end_ticks = internal::Now();
    auto total_ticks = std::uint64_t{};

    if (op::IsCallOpCode(opcode)) {
      const auto depth_idx = static_cast<std::size_t>(ctx.message->depth);
      TOSCA_ASSERT(depth_idx < call_data_.size());
      TOSCA_ASSERT(depth_idx + 1 < call_data_.size());
      const auto& call = call_data_[depth_idx];
      const auto& interpreter = call_data_[depth_idx + 1];
      const auto interpreter_ticks = interpreter.interpreter_end_ticks - interpreter.interpreter_start_ticks;
      const auto call_ticks = end_ticks - call.call_start_ticks;

      // Only access data from recursive call if it succeeded (could have failed due to out of gas, max recursion
      // reached, etc).
      if (interpreter.interpreter_start_ticks != kTicksNotMeasured) {
        TOSCA_ASSERT(call.call_start_ticks < interpreter.interpreter_start_ticks);
        TOSCA_ASSERT(interpreter.interpreter_end_ticks < end_ticks);
        TOSCA_ASSERT(call_ticks > interpreter_ticks);
        total_ticks = call_ticks - interpreter_ticks;
      } else {
        total_ticks = call_ticks;
      }
    } else {
      total_ticks = end_ticks - start_ticks_[opcode_idx];
    }

    ++profile_.calls_[opcode_idx];
    profile_.total_ticks_[opcode_idx] += total_ticks;
  }

  // Start measurement for the interpreter time, only measures time for call depth 0 (i.e. outermost call).
  inline void PreRun(const InterpreterArgs& args) {
    const auto depth_idx = static_cast<std::size_t>(args.message->depth);
    TOSCA_ASSERT(depth_idx < call_data_.size());
    call_data_[depth_idx].interpreter_start_ticks = internal::Now();
  }

  // End measurement for the interpreter time, only measures time for call depth 0 (i.e. outermost call).
  inline void PostRun(const InterpreterArgs& args) {
    const auto depth_idx = static_cast<std::size_t>(args.message->depth);
    TOSCA_ASSERT(depth_idx < call_data_.size());
    call_data_[depth_idx].interpreter_end_ticks = internal::Now();

    if (depth_idx == 0) {
      const auto interpreter_ticks =
          call_data_[depth_idx].interpreter_end_ticks - call_data_[depth_idx].interpreter_start_ticks;
      ++profile_.interpreter_stats_.calls;
      profile_.interpreter_stats_.total_ticks += interpreter_ticks;
    }
  }

  // Adds the counter and execution times of the provided profile to the profile currently recorded by this profiler.
  inline void Merge(const Profile<Mode>& profile) noexcept { profile_.Merge(profile); }

  // Reset collected data.
  inline void Reset() noexcept {
    profile_.Reset();
    start_ticks_ = {};
    call_data_ = {};
  }

  // Get a reference to the data collected so far.
  // Note: Create a copy to retain a snapshot of the current state.
  // Note: This function must not be called when there are ongoing measurements (i.e. Start with no End).
  inline const Profile<Mode>& Collect() noexcept {
    profile_.MarkEnd();
    return profile_;
  }

 private:
  struct CallData {
    std::uint64_t interpreter_start_ticks;
    std::uint64_t interpreter_end_ticks;
    std::uint64_t call_start_ticks;
  };

  static constexpr std::uint64_t kTicksNotMeasured = static_cast<std::uint64_t>(-1);

  Profile<Mode> profile_ = {};
  typename Profile<Mode>::Data start_ticks_ = {};
  // Call depth range is from 0 to kMaxCallDepth inclusive, so requires kMaxCallDepth + 1 entries.
  // One more entry is provided, because an additional call instruction at the call depth limit
  // can be issued, but will not be executed by the interpreter.
  std::array<CallData, internal::kMaxCallDepth + 2> call_data_ = {};
};

static_assert(!std::is_convertible_v<Profile<ProfilerMode::kFull>, Profile<ProfilerMode::kExternal>>);
static_assert(!std::is_convertible_v<Profiler<ProfilerMode::kFull>, Profiler<ProfilerMode::kExternal>>);

}  // namespace tosca::evmzero

#pragma once

#include <array>
#include <chrono>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <utility>

#if defined(__x86_64__)
#include <x86intrin.h>
#endif

#include "build_info.h"
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

// This type represents collected profiling data.
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
      if (op::IsUsedOpCode(opcode)) {
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
class Profiler {
 public:
  Profiler() noexcept = default;
  Profiler(const Profiler&) = delete;
  Profiler(Profiler&&) = delete;

  Profiler& operator=(const Profiler&) = delete;
  Profiler& operator=(Profiler&&) = delete;

  // Construct the profiler from an already existing profile.
  explicit Profiler(const Profile& profile) noexcept : profile_(profile) {}

  // Start measurement for the given opcode. Must be followed by a PostInstruction call for the same opcode to finish
  // the measurement.
  TOSCA_FORCE_INLINE void PreInstruction(op::OpCode opcode, const internal::Context&) {
    if (!op::IsCallOpCode(opcode)) {
      const auto opcode_idx = static_cast<std::size_t>(opcode);
      start_time_[opcode_idx] = internal::Now();
    }
  }

  // End measurement for the given opcode. Must have been preceded by a PreInstruction call of the same opcode.
  TOSCA_FORCE_INLINE void PostInstruction(op::OpCode opcode, const internal::Context&) {
    if (!op::IsCallOpCode(opcode)) {
      const auto opcode_idx = static_cast<std::size_t>(opcode);
      ++profile_.calls_[opcode_idx];
      profile_.total_ticks_[opcode_idx] += internal::Now() - start_time_[opcode_idx];
    }
  }

  // Start measurement for the interpreter time, only measures time for call depth 0 (i.e. outermost call).
  inline void PreRun(const InterpreterArgs& args) {
    if (args.message->depth == 0) {
      interpreter_start_time_ = internal::Now();
    }
  }

  // End measurement for the interpreter time, only measures time for call depth 0 (i.e. outermost call).
  inline void PostRun(const InterpreterArgs& args) {
    if (args.message->depth == 0) {
      ++profile_.interpreter_stats_.calls;
      profile_.interpreter_stats_.total_ticks += internal::Now() - interpreter_start_time_;
    }
  }

  // Adds the counter and execution times of the provided profile to the profile currently recorded by this profiler.
  inline void Merge(const Profile& profile) noexcept { profile_.Merge(profile); }

  // Reset collected data.
  inline void Reset() noexcept {
    profile_.Reset();
    start_time_ = {};
    interpreter_start_time_ = {};
  }

  // Get a reference to the data collected so far.
  // Note: Create a copy to retain a snapshot of the current state.
  // Note: This function must not be called when there are ongoing measurements (i.e. Start with no End).
  inline const Profile& Collect() noexcept {
    profile_.MarkEnd();
    return profile_;
  }

 private:
  Profile profile_;
  Profile::Data start_time_ = {};
  std::uint64_t interpreter_start_time_ = {};
};

}  // namespace tosca::evmzero

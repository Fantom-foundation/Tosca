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
#include "profiler_markers.h"

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
  static constexpr auto kNumMarkers = static_cast<std::size_t>(Marker::NUM_MARKERS);

  // Print the contained profiling data to stdout or to wherever the env var "EVMZERO_PROFILE_FILE" points to.
  void Dump() const {
    auto out_file = std::ofstream();
    const auto* const profile_file = std::getenv("EVMZERO_PROFILE_FILE");
    if (profile_file) {
      out_file.open(profile_file, std::ios::out | std::ios::app);
    }

    // profiling format: <marker>, <calls>, <total-time-ticks>, <total-time-nanoseconds>\n
    std::ostream& out = out_file.is_open() ? out_file : std::cout;
    out << "Compiler: " << internal::kCompilerId << " " << internal::kCompilerVersion << "\n";
    out << "Build type: " << internal::kBuildType << "\n";
    out << "Compile definitions: " << internal::kCompileDefinitions << "\n";
    out << "Compile options: " << internal::kCompileOptions << "\n";
    out << "Assertions: " << internal::kAssertions << "\n";
    out << "ASAN: " << internal::kAsan << "\n";
    out << "Mimalloc: " << internal::kMimalloc << "\n";
    out << "Tracy: " << internal::kTracy << "\n";
    out << "marker,calls,ticks,duration[ns]\n";
    for (std::size_t i = 0; i < kNumMarkers; ++i) {
      const auto marker = static_cast<Marker>(i);
      out << ToString(marker) << ", "       //
          << GetNumCalls(marker) << ", "    //
          << GetTotalTicks(marker) << ", "  //
          << GetTotalTime(marker).count() << "\n";
    }
    out << std::flush;
  }

  // Merge the contained profile with another profile.
  inline void Merge(const Profile& other) noexcept {
    for (std::size_t i = 0; i < kNumMarkers; ++i) {
      calls_[i] += other.calls_[i];
      total_time_[i] += other.total_time_[i];
    }
  }

  // Reset the contained profiling data.
  inline void Reset() noexcept {
    calls_ = {};
    total_time_ = {};
  }

  // Get the number of times the specified marker was called.
  inline std::uint64_t GetNumCalls(const Marker marker) const noexcept {
    return calls_[static_cast<std::size_t>(marker)];
  }

  // Get the cumulative time spent in the specified marker in processor-time.
  inline std::uint64_t GetTotalTicks(const Marker marker) const noexcept {
    return total_time_[static_cast<std::size_t>(marker)];
  }

  // Get the cumulative time spent in the specified marker in nanoseconds.
  inline std::chrono::nanoseconds GetTotalTime(const Marker marker) const noexcept {
    const auto time_ticks = total_time_[static_cast<std::size_t>(marker)];
    return time_converter_.Convert(time_ticks);
  }

 private:
  using Data = std::array<std::uint64_t, kNumMarkers>;

  template <bool ProfilingEnabled>
  friend class Profiler;

  internal::TimeConverter time_converter_;
  Data calls_ = {};
  Data total_time_ = {};

  // Mark the end of a measurement to have a reference with which to convert processor-time to wall-clock time.
  void MarkEnd() noexcept { time_converter_.MarkEnd(); }
};

// This type allows the collection of profiling data through start/end or scoped markers.
template <bool ProfilingEnabled>
class Profiler {
 public:
  Profiler() noexcept = default;
  Profiler(const Profiler&) = delete;
  Profiler(Profiler&&) = delete;

  Profiler& operator=(const Profiler&) = delete;
  Profiler& operator=(Profiler&&) = delete;

  // Construct the profiler from an already existing profile.
  explicit Profiler(const Profile& profile) noexcept : profile_(profile) {}

  // Start measurement for the given marker. Must be followed by an end for the same marker to finish the measurement.
  template <Marker M>
  TOSCA_FORCE_INLINE void Start() noexcept {
    constexpr auto marker_idx = static_cast<std::size_t>(M);
    start_time_[marker_idx] = internal::Now();
  }

  // End measurement for the given marker. Must have been preceded by a start of the same marker.
  template <Marker M>
  TOSCA_FORCE_INLINE void End() noexcept {
    constexpr auto marker_idx = static_cast<std::size_t>(M);
    ++profile_.calls_[marker_idx];
    profile_.total_time_[marker_idx] += internal::Now() - start_time_[marker_idx];
  }

  // RAII style wrapper for Start/End that automatically calls End once the returned type goes out of scope.
  template <Marker M>
  [[nodiscard]] inline auto Scoped() noexcept {
    Start<M>();
    return DeferredEnd<M>(*this);
  }

  // Adds the counter and execution times of the provided profile to the profile currently recorded by this profiler.
  inline void Merge(const Profile& profile) noexcept { profile_.Merge(profile); }

  // Reset collected data.
  inline void Reset() noexcept {
    profile_.Reset();
    start_time_ = {};
  }

  // Get a reference to the data collected so far.
  // Note: Create a copy to retain a snapshot of the current state.
  // Note: This function must not be called when there are ongoing measurements (i.e. Start with no End).
  inline const Profile& Collect() noexcept {
    profile_.MarkEnd();
    return profile_;
  }

 private:
  // RAII helper that automatically calls End when it goes out of scope.
  template <Marker M>
  class DeferredEnd {
   public:
    DeferredEnd(Profiler& profiler) noexcept : profiler_(profiler) {}
    ~DeferredEnd() { profiler_.template End<M>(); }

    DeferredEnd(const DeferredEnd&) = delete;
    DeferredEnd(DeferredEnd&&) = delete;

   private:
    Profiler& profiler_;
  };

  Profile profile_;
  Profile::Data start_time_ = {};
};

// Stub specialization for disabled profiler that does nothing.
template <>
class Profiler<false> {
 public:
  Profiler() noexcept = default;
  Profiler(const Profiler&) = delete;
  Profiler(Profiler&&) = delete;

  Profiler& operator=(const Profiler&) = delete;
  Profiler& operator=(Profiler&&) = delete;

  explicit Profiler(const Profile&) noexcept {}

  template <Marker M>
  inline void Start() noexcept {}
  template <Marker M>
  inline void End() noexcept {}
  template <Marker M>
  [[nodiscard]] inline auto Scoped() noexcept {
    return DeferredEnd{};
  }
  inline void Merge(const Profile&) noexcept {}
  inline void Reset() noexcept {}
  inline Profile Collect() noexcept { return Profile{}; }

 private:
  struct DeferredEnd {
    ~DeferredEnd() {}
  };
};

}  // namespace tosca::evmzero

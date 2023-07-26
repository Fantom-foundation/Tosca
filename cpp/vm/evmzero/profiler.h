#pragma once

#include <array>
#include <cstddef>
#include <cstdint>
#include <iostream>

#if defined(__x86_64__)
#include <x86intrin.h>
#else
#include <chrono>
#endif

#include "common/macros.h"
#include "profiler_markers.h"

namespace tosca::evmzero {

template <bool ProfilingEnabled>
class Profiler {
 public:
  template <Markers Marker>
  TOSCA_FORCE_INLINE void Start() {
    if constexpr (ProfilingEnabled) {
      constexpr auto marker_idx = static_cast<std::size_t>(Marker);
      start_time_[marker_idx] = GetTime();
    }
  }

  template <Markers Marker>
  TOSCA_FORCE_INLINE void End() {
    if constexpr (ProfilingEnabled) {
      constexpr auto marker_idx = static_cast<std::size_t>(Marker);
      ++calls_[marker_idx];
      total_time_[marker_idx] += GetTime() - start_time_[marker_idx];
    }
  }

  template <Markers Marker>
  inline auto Scoped() {
    Start<Marker>();
    return DeferredEnd<Marker>(*this);
  }

  void Dump() const {
    if constexpr (ProfilingEnabled) {
      // profiling format: <marker>, <calls>, <total-time>\n
      for (std::size_t i = 0; i < static_cast<std::size_t>(Markers::NUM_MARKERS); ++i) {
        if (calls_[i]) {
          std::cout << ToString(static_cast<Markers>(i)) << ", "  //
                    << calls_[i] << ", "                          //
                    << total_time_[i];
          std::cout << "\n" << std::flush;
        }
      }
    }
  }

  void Reset() {
    calls_ = {};
    start_time_ = {};
    total_time_ = {};
  }

  inline void Merge(const Profiler& other) {
    if constexpr (ProfilingEnabled) {
      for (std::size_t i = 0; i < static_cast<std::size_t>(Markers::NUM_MARKERS); ++i) {
        calls_[i] += other.calls_[i];
        total_time_[i] += other.total_time_[i];
      }
    }
  }

 private:
  template <Markers Marker>
  class DeferredEnd {
   public:
    DeferredEnd(Profiler& profiler) : profiler_(profiler) {}
    ~DeferredEnd() { profiler_.template End<Marker>(); }

   private:
    Profiler& profiler_;
  };

  std::array<std::uint64_t, static_cast<std::size_t>(Markers::NUM_MARKERS)> calls_ = {};
  std::array<std::uint64_t, static_cast<std::size_t>(Markers::NUM_MARKERS)> start_time_ = {};
  std::array<std::uint64_t, static_cast<std::size_t>(Markers::NUM_MARKERS)> total_time_ = {};

  TOSCA_FORCE_INLINE std::uint64_t GetTime() const {
#if defined(__x86_64__)
    unsigned int _;
    return __rdtscp(&_);
#else
    return static_cast<std::uint64_t>(std::chrono::high_resolution_clock::now().time_since_epoch().count());
#endif
  }
};

}  // namespace tosca::evmzero

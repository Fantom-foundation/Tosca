#pragma once

#include <array>
#include <cstddef>
#include <cstdint>
#include <iostream>

#include <x86intrin.h>

#include "profiler_markers.h"

namespace tosca::evmzero {

template <bool ProfilingEnabled>
class Profiler {
 public:
  template <Markers Marker>
  inline void Start() {
    if constexpr (ProfilingEnabled) {
      constexpr auto marker_idx = static_cast<std::size_t>(Marker);
      ++calls_[marker_idx];
      start_time_[marker_idx] = GetTime();
    }
  }

  template <Markers Marker>
  inline void End() {
    if constexpr (ProfilingEnabled) {
      constexpr auto marker_idx = static_cast<std::size_t>(Marker);
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
                    << total_time_[i] << ", ";
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

  inline std::uint64_t GetTime() const { return __rdtsc(); }
};

}  // namespace tosca::evmzero

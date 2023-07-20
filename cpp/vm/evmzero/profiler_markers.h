#pragma once

#include <cstddef>

namespace tosca::evmzero {

enum class Marker : std::size_t {
#define EVMZERO_PROFILER_MARKER(name) name,
#include "profiler_markers.inc"
};

inline constexpr const char* ToString(const Marker marker) {
  switch (marker) {
#define EVMZERO_PROFILER_MARKER(name) \
  case Marker::name:                  \
    return #name;
#include "profiler_markers.inc"
  }
  return "UNKNOWN";
}

}  // namespace tosca::evmzero

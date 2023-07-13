#pragma once

#include <cstddef>

namespace tosca::evmzero {

enum class Markers : std::size_t {
#define EVMZERO_PROFILER_MARKER(name) name,
#include "profiler_markers.inc"
};

constexpr const char* ToString(Markers marker) {
  switch (marker) {
#define EVMZERO_PROFILER_MARKER(name) \
  case Markers::name:                 \
    return #name;
#include "profiler_markers.inc"
  }
  return "UNKNOWN";
}

}  // namespace tosca::evmzero

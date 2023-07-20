#ifndef EVMZERO_PROFILER_MARKER
#define EVMZERO_PROFILER_MARKER(name)
#endif

// X-macro to construct profiler markers.

EVMZERO_PROFILER_MARKER(TOTAL)
EVMZERO_PROFILER_MARKER(INTERPRETER)

// Create a marker from every opcode
#define EVMZERO_OPCODE(name, value) EVMZERO_PROFILER_MARKER(name)
#include "opcodes.inc"

// Do not add any markers after this, as this represents the number of markers by relying on the fact
// that the last entry in an enum will numerically represent the number of enum entries.
EVMZERO_PROFILER_MARKER(NUM_MARKERS)

#undef EVMZERO_PROFILER_MARKER

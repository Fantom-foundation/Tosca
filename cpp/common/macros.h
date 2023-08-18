#pragma once

#include <cstdlib>

#ifdef __clang__
#define TOSCA_DEBUG_BREAK() __builtin_debugtrap()
#elif __GNUC__
#define TOSCA_DEBUG_BREAK() __builtin_trap()
#elif _MSC_VER
#define TOSCA_DEBUG_BREAK() __debugbreak()
#else
#define TOSCA_DEBUG_BREAK() ::std::abort()
#endif

#define TOSCA_CHECK_OVERFLOW_ADD(a, b, result) __builtin_add_overflow(a, b, result)

#define TOSCA_FORCE_INLINE __attribute__((always_inline))

#define TOSCA_STRINGIFY_(str) #str
#define TOSCA_STRINGIFY(str) TOSCA_STRINGIFY_(str)

#ifdef __clang__
#define TOSCA_COMPILER "clang " __clang_version__
#elif __GNUC__
#define TOSCA_COMPILER \
  "gcc " TOSCA_STRINGIFY(__GNUC__) "." TOSCA_STRINGIFY(__GNUC_MINOR__) "." TOSCA_STRINGIFY(__GNUC_PATCHLEVEL__)
#elif _MSC_VER
#define TOSCA_COMPILER "msvc " TOSCA_STRINGIFY(_MSC_VER)
#else
#define TOSCA_COMPILER "unknown"
#endif

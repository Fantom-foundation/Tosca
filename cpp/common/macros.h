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

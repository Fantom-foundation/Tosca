#pragma once

#include <cstdio>

#include "common/macros.h"

//#if TOSCA_ASSERT_ENABLED
//#define TOSCA_ASSERT(condition)                                                                \
//  do {                                                                                         \
//    if (!(condition)) [[unlikely]] {                                                           \
//      ::std::fprintf(stderr, "%s:%d: Assertion failed: %s\n", __FILE__, __LINE__, #condition); \
//      TOSCA_DEBUG_BREAK();                                                                     \
//    }                                                                                          \
//  } while (0)
//#else
#define TOSCA_ASSERT(condition)
//#endif

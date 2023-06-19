#pragma once

#include <cstdio>
#include <fstream>

#include "common/macros.h"

#if TOSCA_ASSERT_ENABLED
#define TOSCA_ASSERT(condition)                                                                   \
  do {                                                                                            \
    if (!(condition)) [[unlikely]] {                                                              \
      ::std::fprintf(stdout, "%s:%d: Assertion failed: %s\n", __FILE__, __LINE__, #condition);    \
      ::std::fflush(stdout);                                                                      \
      ::std::fprintf(stderr, "%s:%d: Assertion failed: %s\n", __FILE__, __LINE__, #condition);    \
      ::std::fflush(stderr);                                                                      \
      ::std::ofstream("assert.txt") << __FILE__ << ":" << __LINE__ << ": " << #condition << "\n"; \
    }                                                                                             \
  } while (0)
#else
#define TOSCA_ASSERT(condition)
#endif

//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of 
// this software will be governed by the GNU Lesser General Public Licence v3.
//

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
#define TOSCA_CHECK_OVERFLOW_MUL(a, b, result) __builtin_mul_overflow(a, b, result)

#define TOSCA_FORCE_INLINE __attribute__((always_inline))

#define TOSCA_STRINGIFY_(str) #str
#define TOSCA_STRINGIFY(str) TOSCA_STRINGIFY_(str)

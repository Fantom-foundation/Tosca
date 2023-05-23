#pragma once

#ifdef _MSC_VER
#define TOSCA_FORCEINLINE __forceinline
#else
#define TOSCA_FORCEINLINE __attribute__((always_inline))
#endif

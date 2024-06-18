// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#if EVMZERO_MIMALLOC_ENABLED

#include <mimalloc.h>

#include "profiler.h"

// replaceable allocation functions

[[nodiscard]] void* operator new(size_t size) {
  void* ptr = mi_new(size);
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

[[nodiscard]] void* operator new[](size_t size) {
  void* ptr = mi_new(size);
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

[[nodiscard]] void* operator new(size_t size, std::align_val_t al) {
  void* ptr = mi_new_aligned(size, static_cast<size_t>(al));
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

[[nodiscard]] void* operator new[](size_t size, std::align_val_t al) {
  void* ptr = mi_new_aligned(size, static_cast<size_t>(al));
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

// replaceable non-throwing allocation functions

[[nodiscard]] void* operator new(size_t size, const std::nothrow_t&) noexcept {
  void* ptr = mi_new_nothrow(size);
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

[[nodiscard]] void* operator new[](size_t size, const std::nothrow_t&) noexcept {
  void* ptr = mi_new_nothrow(size);
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

[[nodiscard]] void* operator new(size_t size, std::align_val_t al, const std::nothrow_t&) noexcept {
  void* ptr = mi_new_aligned_nothrow(size, static_cast<size_t>(al));
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

[[nodiscard]] void* operator new[](size_t size, std::align_val_t al, const std::nothrow_t&) noexcept {
  void* ptr = mi_new_aligned_nothrow(size, static_cast<size_t>(al));
  EVMZERO_PROFILE_ALLOC(ptr, size);
  return ptr;
}

// replaceable usual deallocation functions

void operator delete(void* ptr) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free(ptr);
}

void operator delete[](void* ptr) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free(ptr);
}

void operator delete(void* ptr, std::align_val_t al) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free_aligned(ptr, static_cast<size_t>(al));
}

void operator delete[](void* ptr, std::align_val_t al) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free_aligned(ptr, static_cast<size_t>(al));
}

void operator delete(void* ptr, size_t) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free(ptr);
}

void operator delete[](void* ptr, size_t) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free(ptr);
}

void operator delete(void* ptr, size_t, std::align_val_t al) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free_aligned(ptr, static_cast<size_t>(al));
}

void operator delete[](void* ptr, size_t, std::align_val_t al) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free_aligned(ptr, static_cast<size_t>(al));
}

// replaceable placement deallocation functions

void operator delete(void* ptr, const std::nothrow_t&) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free(ptr);
}

void operator delete[](void* ptr, const std::nothrow_t&) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free(ptr);
}

void operator delete(void* ptr, std::align_val_t al, const std::nothrow_t&) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free_aligned(ptr, static_cast<size_t>(al));
}

void operator delete[](void* ptr, std::align_val_t al, const std::nothrow_t&) noexcept {
  EVMZERO_PROFILE_FREE(ptr);
  mi_free_aligned(ptr, static_cast<size_t>(al));
}

#endif

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

#include <array>
#include <cstdint>
#include <functional>
#include <span>

namespace tosca {

template <typename T>
constexpr void CombineHash(size_t& seed, const T& v) {
  std::hash<T> hasher;
  seed ^= hasher(v) + 0x9e3779b9 + (seed << 6) + (seed >> 2);
}

template <typename It>
constexpr void HashRange(size_t& seed, It first, It last) {
  for (; first != last; ++first) {
    CombineHash(seed, *first);
  }
}

template <size_t N>
struct HashBytes {
  constexpr size_t operator()(std::span<const uint8_t, N> key) const noexcept {
    // Hash initial bytes as multi-byte values.
    using Word = uint32_t;  // HashCombine is tuned for 32-bit words and does not work well with 64-bit values.
    constexpr size_t kWordSize = sizeof(Word);
    constexpr size_t kNumFullWords = N / kWordSize;
    size_t seed = 0;
    if constexpr (kNumFullWords > 0) {
      const Word* begin = reinterpret_cast<const Word*>(key.data());
      HashRange(seed, begin, begin + kNumFullWords);
    }

    // Hash the rest as bytes.
    if constexpr (N % kWordSize != 0) {
      HashRange(seed, key.begin() + kNumFullWords * kWordSize, key.end());
    }
    return seed;
  }
};

}  // namespace tosca

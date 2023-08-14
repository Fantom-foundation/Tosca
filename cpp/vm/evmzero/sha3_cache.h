#pragma once

#include <algorithm>
#include <array>
#include <iostream>
#include <span>

#include "common/hash_utils.h"
#include "common/lru_cache.h"
#include "common/macros.h"
#include "ethash/keccak.hpp"
#include "vm/evmzero/uint256.h"

namespace tosca::evmzero {

template <size_t N>
struct HashBytes {
  constexpr size_t operator()(std::span<const uint8_t, N> key) const noexcept {
    // Hash initial bytes as multi-byte values.
    using Word = uint32_t;  // HashCombine is tuned for 32-bit words.
    constexpr size_t kWordSize = sizeof(Word);
    constexpr size_t kNumFullWords = N / kWordSize;
    size_t seed = 0;
    if constexpr (kNumFullWords > 0) {
      const Word* begin = reinterpret_cast<const Word*>(key.data());
      tosca::HashRange(seed, begin, begin + kNumFullWords);
    }

    // Hash the rest as bytes.
    if constexpr (N % kWordSize != 0) {
      tosca::HashRange(seed, key.begin() + kNumFullWords * kWordSize, key.end());
    }
    return seed;
  }
};

// This component calculates and caches keccak256 hashes. The cache is composed
// of multiple buckets varying in key size, each of which has a fixed maximum
// capcity. A least-recently-used strategy is employed.
class Sha3Cache {
 public:
  TOSCA_FORCE_INLINE uint256_t Hash(std::span<const uint8_t> key_view) noexcept {
    auto calculate_hash = [&key_view, this]() {  //
      // misses_++;
      return ToUint256(ethash::keccak256(key_view.data(), key_view.size()));
    };

    /*
        if (accesses_ % 100000 == 0) {
          auto hits = accesses_ - misses_;
          auto rate = float(hits)/float(accesses_);
          std::cout << "Sha3Cache - hits: " << hits << " of " << accesses_ << " (" << rate << ")\n";
        }

        accesses_++;
    */
    if (key_view.size() == 32) {
      Bytes<32> key;
      std::copy_n(key_view.data(), 32, key.begin());
      return cache_32_.GetOrInsert(key, calculate_hash);
    }

    if (key_view.size() == 64) {
      Bytes<64> key;
      std::copy_n(key_view.data(), 64, key.begin());
      return cache_64_.GetOrInsert(key, calculate_hash);
    }

    return calculate_hash();
  }

 private:
  template <size_t N>
  using Bytes = std::array<uint8_t, N>;

  // N=1024 => ~84% hit rate
  // N=4096 => ~85% hit rate
  // N=2^16 => ~88% hit rate
  // N=2^20 => ~89% hit rate

  LruCache<Bytes<32>, uint256_t, 1024, HashBytes<32>> cache_32_;
  LruCache<Bytes<64>, uint256_t, 1024, HashBytes<64>> cache_64_;

  /*
    uint64_t accesses_;
    uint64_t misses_;
    */
};

}  // namespace tosca::evmzero

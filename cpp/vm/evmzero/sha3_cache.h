#pragma once

#include <algorithm>
#include <array>
#include <span>

#include "common/hash_utils.h"
#include "common/lru_cache.h"
#include "common/macros.h"
#include "ethash/keccak.hpp"
#include "vm/evmzero/uint256.h"

namespace tosca::evmzero {

// This component calculates and caches keccak256 hashes. The cache is composed
// of multiple buckets varying in key size, each of which has a fixed maximum
// capcity. A least-recently-used strategy is employed.
class Sha3Cache {
 public:
  TOSCA_FORCE_INLINE uint256_t Hash(std::span<const uint8_t> key_view) noexcept {
    auto calculate_hash = [&key_view]() {  //
      return ToUint256(ethash::keccak256(key_view.data(), key_view.size()));
    };

    if (key_view.size() == 32) {
      Bytes<32> key;
      std::copy_n(key_view.data(), 32, key.begin());
      return cache_32_.GetOrInsert(key, calculate_hash);
    }

    else if (key_view.size() == 64) {
      Bytes<64> key;
      std::copy_n(key_view.data(), 64, key.begin());
      return cache_64_.GetOrInsert(key, calculate_hash);
    }

    else {
      return calculate_hash();
    }
  }

 private:
  template <size_t N>
  using Bytes = std::array<uint8_t, N>;

  template <size_t N>
  struct HashBytes {
    constexpr size_t operator()(std::span<const uint8_t, N> key) const noexcept {
      size_t seed = 0;
      tosca::HashRange(seed, key.begin(), key.end());
      return seed;
    }
  };

  // N=1024 => ~84% hit rate
  // N=4096 => ~85% hit rate
  // N=2^16 => ~88% hit rate
  // N=2^20 => ~89% hit rate



  LruCache<Bytes<32>, uint256_t, 1024, HashBytes<32>> cache_32_;
  LruCache<Bytes<64>, uint256_t, 1024, HashBytes<64>> cache_64_;
};

}  // namespace tosca::evmzero

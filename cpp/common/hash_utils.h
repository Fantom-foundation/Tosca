#pragma once

#include <array>
#include <functional>

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

}  // namespace tosca

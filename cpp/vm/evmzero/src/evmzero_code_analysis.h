#pragma once

#include <algorithm>
#include <cinttypes>
#include <cstdint>
#include <cstdio>
#include <map>
#include <tuple>

#include "evmzero.h"

namespace tosca::evmzero {

template <typename A, typename B>
std::pair<B, A> FlipPair(const std::pair<A, B>& p) {
  return std::pair<B, A>(p.second, p.first);
}

template <typename A, typename B>
std::multimap<B, A> FlipMap(const std::map<A, B>& src) {
  std::multimap<B, A> dst;
  std::transform(src.begin(), src.end(), std::inserter(dst, dst.begin()), FlipPair<A, B>);
  return dst;
}

inline void AnalyzeFrequencies(std::vector<uint8_t> code) {
  std::map<std::pair<uint8_t, uint8_t>, uint64_t> pair_frequencies;
  std::map<std::tuple<uint8_t, uint8_t, uint8_t>, uint64_t> triple_frequencies;
  std::map<std::tuple<uint8_t, uint8_t, uint8_t, uint8_t>, uint64_t> quad_frequencies;

  for (std::size_t i = 0; i < code.size(); ++i) {
    if (i < code.size() - 1) {
      pair_frequencies[{code[i], code[i + 1]}]++;
    }
    if (i < code.size() - 2) {
      triple_frequencies[{code[i], code[i + 1], code[i + 2]}]++;
    }
    if (i < code.size() - 3) {
      quad_frequencies[{code[i], code[i + 1], code[i + 2], code[i + 3]}]++;
    }
  }

  static constexpr int kPrintTopN = 5;

  {
    const auto pairs = FlipMap(pair_frequencies);
    auto cur = pairs.rbegin();
    for (int i = 0; i < kPrintTopN; ++i) {
      printf("%02hhX %02hhX: %" PRIu64 " (%" PRIu64 "%%)\n", cur->second.first, cur->second.second, cur->first,
             cur->first * 100 / code.size());
      cur++;
    }
  }

  {
    const auto triples = FlipMap(triple_frequencies);
    auto cur = triples.rbegin();
    for (int i = 0; i < kPrintTopN; ++i) {
      printf("%02hhX %02hhX %02hhX: %" PRIu64 " (%" PRIu64 "%%)\n", std::get<0>(cur->second), std::get<1>(cur->second),
             std::get<2>(cur->second), cur->first, cur->first * 100 / code.size());
      cur++;
    }
  }

  {
    const auto quads = FlipMap(quad_frequencies);
    auto cur = quads.rbegin();
    for (int i = 0; i < kPrintTopN; ++i) {
      printf("%02hhX %02hhX %02hhX %02hhX: %" PRIu64 " (%" PRIu64 "%%)\n", std::get<0>(cur->second),
             std::get<1>(cur->second), std::get<2>(cur->second), std::get<3>(cur->second), cur->first,
             cur->first * 100 / code.size());
      cur++;
    }
  }
}

}  // namespace tosca::evmzero

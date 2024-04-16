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

#include "common/hash_utils.h"

#include <array>
#include <cstdint>
#include <span>

#include "benchmark/benchmark.h"

namespace tosca {
namespace {

// To run benchmarks, use the following commands:
//
//    cmake -Bbuild -DCMAKE_BUILD_TYPE=Release -DTOSCA_ASAN=OFF
//    cmake --build build --parallel --target hash_utils_benchmark
//    ./build/common/hash_utils_benchmark
//
// To get a CPU profile, use
//
//    CPUPROFILE=profile.dat ./build/common/hash_utils_benchmark
//
// and
//
//    go tool pprof --http="localhost:8001" profile.dat
//
// for an interactive visualization of the profile.

// Updates values in the given byte span such that a cycle of 1024 items is
// created when calling this function in sequence. In particular, starting
// with an all-zero span, after 1024 calls the same span is reproduced.
void Inc(std::span<uint8_t> data) {
  auto low = data.size() / 3;
  auto high = 2 * low;
  if (data[low]++ == 0) {
    data[high] = static_cast<uint8_t>((data[high] + 1) % 4);
  }
}

template <size_t N>
struct HashBytesByteWise {
  constexpr size_t operator()(std::span<const uint8_t, N> key) const noexcept {
    size_t seed = 0;
    HashRange(seed, key.begin(), key.end());
    return seed;
  }
};

// Benchmark the performance of hashing of input data.
void BM_HashBytesByteWise(benchmark::State& state) {
  std::array<uint8_t, 64> data{};
  for (auto _ : state) {
    Inc(data);
    auto hash = HashBytesByteWise<64>()(data);
    benchmark::DoNotOptimize(hash);
  }
}

BENCHMARK(BM_HashBytesByteWise);

// Benchmark the performance of hashing of input data.
void BM_HashBytesWordWise(benchmark::State& state) {
  std::array<uint8_t, 64> data{};
  for (auto _ : state) {
    Inc(data);
    auto hash = HashBytes<64>()(data);
    benchmark::DoNotOptimize(hash);
  }
}

BENCHMARK(BM_HashBytesWordWise);

}  // namespace
}  // namespace tosca

#include "vm/evmzero/sha3_cache.h"

#include <array>
#include <cstdint>
#include <span>

#include "benchmark/benchmark.h"
#include "ethash/keccak.hpp"

namespace tosca::evmzero {
namespace {

// To run benchmarks, use the following commands:
//
//    cmake -Bbuild -DCMAKE_BUILD_TYPE=Release -DTOSCA_ASAN=OFF
//    cmake --build build --parallel --target sha3_cache_benchmark
//    ./build/vm/evmzero/sha3_cache_benchmark
//
// To get a CPU profile, use
//
//    CPUPROFILE=profile.dat ./build/vm/evmzero/sha3_cache_benchmark
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

// Benchmark the performance of SHA3 hashing.
void BM_Sha3Hash(benchmark::State& state) {
  std::vector<uint8_t> data(static_cast<size_t>(state.range(0)));
  for (auto _ : state) {
    Inc(data);
    auto hash = ethash::keccak256(data.data(), data.size());
    benchmark::DoNotOptimize(hash);
  }
}

BENCHMARK(BM_Sha3Hash)->Arg(32)->Arg(48)->Arg(64);

// Benchmark the performance of SHA3 hashing when enabling caching.
void BM_Sha3HashCached(benchmark::State& state) {
  std::vector<uint8_t> data(static_cast<size_t>(state.range(0)));
  Sha3Cache cache;
  for (auto _ : state) {
    Inc(data);
    auto hash = cache.Hash(data);
    benchmark::DoNotOptimize(hash);
  }
}

BENCHMARK(BM_Sha3HashCached)->Arg(32)->Arg(48)->Arg(64);

// Benchmark the performance of hashing of input data.
void BM_DataHashingOld(benchmark::State& state) {
  std::array<uint8_t, 64> data{};
  for (auto _ : state) {
    Inc(data);
    auto hash = HashBytesOld<64>()(data);
    benchmark::DoNotOptimize(hash);
  }
}

BENCHMARK(BM_DataHashingOld);

// Benchmark the performance of hashing of input data.
void BM_DataHashing(benchmark::State& state) {
  std::array<uint8_t, 64> data{};
  for (auto _ : state) {
    Inc(data);
    auto hash = HashBytes<64>()(data);
    benchmark::DoNotOptimize(hash);
  }
}

BENCHMARK(BM_DataHashing);

}  // namespace
}  // namespace tosca::evmzero
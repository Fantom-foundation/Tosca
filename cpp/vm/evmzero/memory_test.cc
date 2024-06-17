// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#include "vm/evmzero/memory.h"

#include <gtest/gtest.h>

namespace tosca::evmzero {
namespace {

TEST(MemoryTest, DefaultInit) {
  Memory memory;
  EXPECT_EQ(memory.GetSize(), 0);
}

TEST(MemoryTest, InitializerList) {
  Memory memory = {1, 2, 3};
  ASSERT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 1);
  EXPECT_EQ(memory[1], 2);
  EXPECT_EQ(memory[2], 3);
  for (uint64_t i = 3; i < memory.GetSize(); ++i) {
    EXPECT_EQ(memory[i], 0);
  }
}

TEST(MemoryTest, InitializerSpan) {
  const auto elements = std::array<uint8_t, 3>{1, 2, 3};
  Memory memory(elements);
  ASSERT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 1);
  EXPECT_EQ(memory[1], 2);
  EXPECT_EQ(memory[2], 3);
  for (uint64_t i = 3; i < memory.GetSize(); ++i) {
    EXPECT_EQ(memory[i], 0);
  }
}

TEST(MemoryTest, ReadFrom) {
  Memory memory;

  std::vector<uint8_t> buffer = {1, 2, 3};
  memory.ReadFrom(buffer, 1);

  ASSERT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 0);  // zero initialized
  EXPECT_EQ(memory[1], 1);
  EXPECT_EQ(memory[2], 2);
  EXPECT_EQ(memory[3], 3);
  EXPECT_EQ(memory[4], 0);  // zero initialized
}

TEST(MemoryTest, ReadFrom_ZeroSize) {
  Memory memory;

  std::vector<uint8_t> buffer;
  memory.ReadFrom(buffer, 42);

  EXPECT_EQ(memory.GetSize(), 0);
}

TEST(MemoryTest, GrowsByMultipleOf32) {
  Memory memory;
  EXPECT_EQ(memory.GetSize(), 0);

  std::vector<uint8_t> buffer(1);
  memory.ReadFrom(buffer, 0);
  EXPECT_EQ(memory.GetSize(), 32);

  buffer.resize(35);
  memory.ReadFrom(buffer, 0);
  EXPECT_EQ(memory.GetSize(), 64);
}

TEST(MemoryTest, ReadFromWithSize_SmallerSize) {
  Memory memory;

  std::vector<uint8_t> buffer = {1, 2, 3};
  memory.ReadFromWithSize(buffer, 1, 2);

  ASSERT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 0);  // zero initialized
  EXPECT_EQ(memory[1], 1);
  EXPECT_EQ(memory[2], 2);
  EXPECT_EQ(memory[3], 0);  // zero initialized
}

TEST(MemoryTest, ReadFromWithSize_LargerSize) {
  Memory memory = {0xFF, 0xFF, 0xFF, 0xFF, 0xFF};

  std::vector<uint8_t> buffer = {1, 2};
  memory.ReadFromWithSize(buffer, 1, 3);

  ASSERT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 0xFF);
  EXPECT_EQ(memory[1], 1);
  EXPECT_EQ(memory[2], 2);
  EXPECT_EQ(memory[3], 0);  // filled with zero
  EXPECT_EQ(memory[4], 0xFF);
  EXPECT_EQ(memory[5], 0);  // zero initialized
}

TEST(MemoryTest, ReadFromWithSize_ZeroSize) {
  Memory memory;

  std::vector<uint8_t> buffer = {1, 2};
  memory.ReadFromWithSize(buffer, 42, 0);

  EXPECT_EQ(memory.GetSize(), 0);
}

TEST(MemoryTest, WriteTo) {
  Memory memory = {1, 2, 3};

  std::vector<uint8_t> buffer(3);
  memory.WriteTo(buffer, 0);

  EXPECT_EQ(buffer[0], 1);
  EXPECT_EQ(buffer[1], 2);
  EXPECT_EQ(buffer[2], 3);
}

TEST(MemoryTest, WriteTo_WritesZeros) {
  Memory memory = {1, 2, 3};

  std::vector<uint8_t> buffer = {4, 5, 7};
  memory.WriteTo(buffer, 1);

  EXPECT_EQ(buffer[0], 2);
  EXPECT_EQ(buffer[1], 3);
  EXPECT_EQ(buffer[2], 0);  // filled with zero
}

TEST(MemoryTest, WriteTo_Grows) {
  Memory memory = {1, 2, 3};

  std::vector<uint8_t> buffer(3);
  memory.WriteTo(buffer, 1);

  ASSERT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 1);
  EXPECT_EQ(memory[1], 2);
  EXPECT_EQ(memory[2], 3);
  EXPECT_EQ(memory[3], 0);  // zero initialized
}

TEST(MemoryTest, WriteTo_ZeroSize) {
  Memory memory;

  std::vector<uint8_t> buffer;
  memory.WriteTo(buffer, 42);

  EXPECT_EQ(memory.GetSize(), 0);
}

TEST(MemoryTest, Grow) {
  Memory memory;
  memory.Grow(0, 16);
  EXPECT_EQ(memory.GetSize(), 32);

  memory.Grow(32, 16);
  EXPECT_EQ(memory.GetSize(), 64);

  memory.Grow(0, 16);
  EXPECT_EQ(memory.GetSize(), 64);
}

TEST(MemoryTest, Grow_ZeroSize) {
  Memory memory;
  memory.Grow(128, 0);
  EXPECT_EQ(memory.GetSize(), 0);
}

TEST(MemoryTest, Subscript) {
  Memory memory = {1, 2, 3};

  ASSERT_EQ(memory.GetSize(), 32);

  memory[1] = 42;

  EXPECT_EQ(memory[0], 1);
  EXPECT_EQ(memory[1], 42);
  EXPECT_EQ(memory[2], 3);
}

TEST(MemoryTest, Equality) {
  Memory m1, m2;
  EXPECT_EQ(m1, m2);

  m1.ReadFrom(std::vector<uint8_t>{0, 0, 0}, 0);
  EXPECT_NE(m1, m2);

  m2.ReadFrom(std::vector<uint8_t>{1, 2, 3}, 0);
  EXPECT_NE(m1, m2);

  m1[0] = 1;
  m1[1] = 2;
  m1[2] = 3;
  EXPECT_EQ(m1, m2);
}

struct TestParams {
  const size_t size;
  const size_t start_offset;
  const size_t dest_offset;
};
class ParametrizedMemoryTest : public testing::TestWithParam<TestParams> {};

constexpr auto kMB = 1024 * 1024;
constexpr auto kBufferSize = 16 * kMB;

INSTANTIATE_TEST_SUITE_P(ParametrizedMemoryTestWithSizes, ParametrizedMemoryTest,
                         testing::Values(
                             TestParams{
                                 .size = 4 * kMB,
                                 .start_offset = 0,
                                 .dest_offset = 0,
                             },
                             TestParams{
                                 .size = 4 * kMB,
                                 .start_offset = 2 * kMB,
                                 .dest_offset = 0,
                             },
                             TestParams{
                                 .size = 4 * kMB,
                                 .start_offset = 0,
                                 .dest_offset = 2 * kMB,
                             },
                             TestParams{
                                 .size = 4 * kMB,
                                 .start_offset = kBufferSize - 1,
                                 .dest_offset = 0,
                             },
                             TestParams{
                                 .size = 4 * kMB,
                                 .start_offset = 0,
                                 .dest_offset = kBufferSize - 1,
                             })

);

TEST_P(ParametrizedMemoryTest, MemCopy) {
  const auto test = GetParam();
  const auto does_expansion = test.start_offset + test.size > kBufferSize || test.dest_offset + test.size > kBufferSize;
  const auto effective_size = std::min(kBufferSize - test.start_offset, test.size);

  std::vector<uint8_t> data(kBufferSize);
  // Fill memory to be copied with 0xFF, so that we can detect zeroes latter on
  std::fill(data.begin(), data.end(), 0xFF);
  // Fill range to be copied with a pattern not congruence of 2,4,8,16... to check for aliasing
  auto to_fill = std::span{data.data() + test.start_offset, effective_size};
  constexpr auto kPrime = 37;
  auto count = 0ul;
  std::transform(to_fill.begin(), to_fill.end(), to_fill.begin(), [&](uint8_t) { return count++ % kPrime; });

  Memory memory(data);
  EXPECT_EQ(memory.GetSize(), kBufferSize);
  auto check_pattern = [&](std::span<uint8_t> span) {
    int count = 0ul;
    for (auto i = 0ul; i < effective_size; ++i) {
      EXPECT_EQ(span[i], count++ % kPrime) << "at index " << i;
    }
    if (does_expansion) {
      for (auto i = effective_size; i < test.size; ++i) {
        EXPECT_EQ(span[i], 0) << "at index " << i;
      }
    }
  };

  // Check pattern before copy in source range.
  check_pattern(memory.GetSpan(test.start_offset, test.size));

  memory.MemCopy(test.dest_offset, test.start_offset, test.size);

  // Check that expansion did not happen. If not, the pattern_checker will check for zero padding.
  // Memory expansion sizes is checked in the Grow test.
  if (!does_expansion) {
    EXPECT_EQ(memory.GetSize(), kBufferSize);
  }

  // Check pattern after copy in dst range.
  check_pattern(memory.GetSpan(test.dest_offset, test.size));
}

}  // namespace
}  // namespace tosca::evmzero

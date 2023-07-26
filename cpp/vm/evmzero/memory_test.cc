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
  EXPECT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 1);
  EXPECT_EQ(memory[1], 2);
  EXPECT_EQ(memory[2], 3);
}

TEST(MemoryTest, ReadFrom) {
  Memory memory;

  std::vector<uint8_t> buffer = {1, 2, 3};
  memory.ReadFrom(buffer, 1);

  EXPECT_EQ(memory.GetSize(), 32);

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

  EXPECT_EQ(memory.GetSize(), 32);

  EXPECT_EQ(memory[0], 0);  // zero initialized
  EXPECT_EQ(memory[1], 1);
  EXPECT_EQ(memory[2], 2);
  EXPECT_EQ(memory[3], 0);  // zero initialized
}

TEST(MemoryTest, ReadFromWithSize_LargerSize) {
  Memory memory = {0xFF, 0xFF, 0xFF, 0xFF, 0xFF};

  std::vector<uint8_t> buffer = {1, 2};
  memory.ReadFromWithSize(buffer, 1, 3);

  EXPECT_EQ(memory.GetSize(), 32);

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

}  // namespace
}  // namespace tosca::evmzero

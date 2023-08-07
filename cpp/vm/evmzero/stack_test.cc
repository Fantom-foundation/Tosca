#include "vm/evmzero/stack.h"

#include <gtest/gtest.h>

namespace tosca::evmzero {
namespace {

TEST(StackTest, Empty) {
  Stack stack;
  EXPECT_EQ(stack.GetSize(), 0);
}

TEST(StackTest, Init) {
  Stack stack = {1, 2, 3};

  EXPECT_EQ(stack.GetSize(), 3);

  EXPECT_EQ(stack.Pop(), 3);
  EXPECT_EQ(stack.Pop(), 2);
  EXPECT_EQ(stack.Pop(), 1);
}

TEST(StackTest, PushPop) {
  Stack stack;

  stack.Push(1);
  stack.Push(2);
  stack.Push(3);

  EXPECT_EQ(stack.GetSize(), 3);

  EXPECT_EQ(stack.Pop(), 3);
  EXPECT_EQ(stack.Pop(), 2);
  EXPECT_EQ(stack.Pop(), 1);

  EXPECT_EQ(stack.GetSize(), 0);
}

TEST(StackTest, Subscript) {
  Stack stack = {1, 2, 3};

  EXPECT_EQ(stack[0], 3);
  EXPECT_EQ(stack[1], 2);
  EXPECT_EQ(stack[2], 1);
}

TEST(StackTest, Equality) {
  Stack s1, s2;

  EXPECT_EQ(s1, s2);

  s1.Push(1);
  EXPECT_NE(s1, s2);

  s2.Push(2);
  EXPECT_NE(s1, s2);

  s2.Pop();
  s2.Push(1);
  EXPECT_EQ(s1, s2);

  s2.Pop();
  EXPECT_NE(s1, s2);

  s1.Pop();
  EXPECT_EQ(s1, s2);
}

TEST(StackTest, Equality2) {
  Stack s1, s2;

  s1.Push(1);
  s1.Pop();
  s2.Push(2);
  s2.Pop();
  EXPECT_EQ(s1, s2);
}

}  // namespace
}  // namespace tosca::evmzero

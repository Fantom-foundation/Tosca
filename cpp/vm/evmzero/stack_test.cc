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

#include "vm/evmzero/stack.h"

#include <gtest/gtest.h>
#include <type_traits>

namespace tosca::evmzero {
namespace {

TEST(StackTest, Traits) {
  EXPECT_TRUE(std::is_copy_constructible_v<Stack>);
  EXPECT_FALSE(std::is_move_constructible_v<Stack>);
  EXPECT_TRUE(std::is_copy_assignable_v<Stack>);
  EXPECT_FALSE(std::is_move_assignable_v<Stack>);
}

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

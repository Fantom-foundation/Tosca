//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of 
// this software will be governed by the GNU Lesser General Public Licence v3.
//

#include "vm/evmzero/uint256.h"

#include <gmock/gmock.h>
#include <gtest/gtest.h>

namespace tosca::evmzero {
namespace {

using ::testing::StrEq;

TEST(Uint256Test, Max) {
  uint256_t value(0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff, 0xffffffffffffffff);
  EXPECT_EQ(value, kUint256Max);
}

TEST(Uint256Test, Underflow) {
  uint256_t value = 0;
  value -= 1;

  EXPECT_EQ(value, kUint256Max);
}

TEST(Uint256Test, ByteAccess) {
  uint256_t value = 0xff;

  EXPECT_EQ(ToBytes(value)[0], 0xff);

  value = 0xff00;
  EXPECT_EQ(ToBytes(value)[0], 0x00);
  EXPECT_EQ(ToBytes(value)[1], 0xff);

  ToBytes(value)[0] = 0xff;
  EXPECT_EQ(value, 0xffff);
}

TEST(Uint256Test, ToString) {
  uint256_t word{0xAF030201};
  EXPECT_THAT(ToString(word), StrEq("2936209921"));
}

TEST(Uint256Test, CanBeEqualityCompared) {
  uint256_t w0{0};
  uint256_t w1{2, 2};
  uint256_t w2{3, 1};

  EXPECT_EQ(w0, w0);
  EXPECT_EQ(w1, w1);
  EXPECT_EQ(w2, w2);

  EXPECT_NE(w0, w1);
  EXPECT_NE(w0, w2);
  EXPECT_NE(w1, w2);
}

TEST(Uint256Test, CanBeLessCompared) {
  uint256_t w0{0};
  uint256_t w1{2, 2};
  uint256_t w2{3, 1};

  EXPECT_LT(w0, w1);
  EXPECT_LT(w0, w2);
  EXPECT_LT(w0, w2);

  EXPECT_FALSE(w0 < w0);
  EXPECT_FALSE(w1 < w0);
  EXPECT_FALSE(w1 < w2);
  EXPECT_FALSE(w2 < w0);
}
TEST(Uint256Test, CanBeLessEqCompared) {
  uint256_t w0{0};
  uint256_t w1{2, 2};
  uint256_t w2{3, 1};

  EXPECT_LE(w0, w0);
  EXPECT_LE(w0, w1);
  EXPECT_LE(w0, w2);
  EXPECT_LE(w0, w2);

  EXPECT_FALSE(w1 <= w0);
  EXPECT_FALSE(w1 <= w2);
  EXPECT_FALSE(w2 <= w0);
}

TEST(Uint256Test, CanBeGreaterCompared) {
  uint256_t w0{0};
  uint256_t w1{2, 2};
  uint256_t w2{3, 1};

  EXPECT_GT(w1, w0);
  EXPECT_GT(w2, w0);
  EXPECT_GT(w2, w0);

  EXPECT_FALSE(w0 > w0);
  EXPECT_FALSE(w0 > w1);
  EXPECT_FALSE(w0 > w2);
  EXPECT_FALSE(w2 > w1);
}

TEST(Uint256Test, CanBeGreaterEqCompared) {
  uint256_t w0{0};
  uint256_t w1{2, 2};
  uint256_t w2{3, 1};

  EXPECT_GE(w0, w0);
  EXPECT_GE(w1, w0);
  EXPECT_GE(w2, w0);
  EXPECT_GE(w2, w0);

  EXPECT_FALSE(w0 >= w1);
  EXPECT_FALSE(w0 >= w2);
  EXPECT_FALSE(w2 >= w1);
}

TEST(Uint256Test, CanBeAdded) { EXPECT_EQ(uint256_t{2} + uint256_t{3}, uint256_t{5}); }

TEST(Uint256Test, CanOverflowWhenAdded) { EXPECT_EQ(kUint256Max + uint256_t{2}, uint256_t{1}); }

TEST(Uint256Test, CanBeSubtracted) { EXPECT_EQ(uint256_t{3} - uint256_t{2}, uint256_t{1}); }

TEST(Uint256Test, CanUnderflowWhenSubtracted) { EXPECT_EQ(uint256_t{0} - uint256_t{1}, kUint256Max); }

TEST(Uint256Test, CanBeMultiplied) { EXPECT_EQ(uint256_t{2} * uint256_t{3}, uint256_t{6}); }

TEST(Uint256Test, CanOverflowWhenMultiplied) { EXPECT_EQ(kUint256Max * uint256_t{2}, kUint256Max - uint256_t{1}); }

TEST(Uint256Test, CanBeDivided) {
  uint256_t w24{24};

  EXPECT_EQ(w24 / uint256_t{8}, uint256_t{3});  // no remainder
  EXPECT_EQ(w24 / uint256_t{5}, uint256_t{4});  // remainder
}

TEST(Uint256Test, CanUseModulo) {
  uint256_t w24{24};

  EXPECT_EQ(w24 % uint256_t{8}, uint256_t{0});  // no remainder
  EXPECT_EQ(w24 % uint256_t{5}, uint256_t{4});  // remainder
}

TEST(Uint256Test, CanBeExponentiated) { EXPECT_EQ(intx::exp(uint256_t{10}, uint256_t{2}), uint256_t{100}); }

TEST(Uint256Test, CanBeShiftedLeft) { EXPECT_EQ(uint256_t{0xF} << uint256_t{4}, uint256_t{0xF0}); }

TEST(Uint256Test, CanBeShiftedRight) {
  uint256_t w{0xF0};
  EXPECT_EQ(w >> uint256_t{4}, uint256_t{0xF});
}

TEST(Uint256Test, CanBeBitwiseORed) {
  EXPECT_EQ(uint256_t{0xF0} | uint256_t{0x0F}, uint256_t{0xFF});
  EXPECT_EQ(uint256_t{0xFF} | uint256_t{0xFF}, uint256_t{0xFF});
}

TEST(Uint256Test, CanBeBitwiseANDed) {
  EXPECT_EQ(uint256_t{0x0F} & uint256_t{0x0F}, uint256_t{0x0F});
  EXPECT_EQ(uint256_t{0xFF} & uint256_t{0x00}, uint256_t{0x00});
}

TEST(Uint256Test, CanBeBitwiseXORed) {
  EXPECT_EQ(uint256_t{0xF0} ^ uint256_t{0x0F}, uint256_t{0xFF});
  EXPECT_EQ(uint256_t{0xFF} ^ uint256_t{0xFF}, uint256_t{0x00});
}

TEST(Uint256Test, CanBeBitwiseNOTed) { EXPECT_EQ(~uint256_t{0}, kUint256Max); }

}  // namespace
}  // namespace tosca::evmzero

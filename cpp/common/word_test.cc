#include "common/word.h"

#include <compare>
#include <type_traits>

#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace tosca {
namespace {

using ::testing::Eq;
using ::testing::PrintToString;
using ::testing::StrEq;

TEST(Word, Zero) {
  Word zero{};
  EXPECT_EQ(zero, Word{0});
}
TEST(Word, Max) {
  Word all_bits_set{
      0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
      0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
  };

  EXPECT_EQ(Word::kMax, all_bits_set);
}

TEST(Word, CanAccessBytes) {
  Word w_ff{0xFF};
  Word w_ff00{0xFF, 0x00};

  EXPECT_EQ(w_ff[Word{31}], std::byte{0xFF});
  EXPECT_EQ(w_ff00[Word{30}], std::byte{0xFF});

  // EVM requires: If the byte offset is out of range, the result is 0.
  EXPECT_EQ(w_ff00[Word{42}], std::byte{0});
}

TEST(Word, PrintProducesHexString) {
  Word word{0xAF, 3, 2, 1};
  EXPECT_THAT(PrintToString(word), StrEq("00000000000000000000000000000000000000000000000000000000AF030201"));
}

TEST(Word, CheckEndianness) {
  Word word{2, 1};
  word = word + Word{1};
  EXPECT_THAT(word, Eq(Word{2, 2}));
}

TEST(Word, CanBeEqualityCompared) {
  Word w0{0};
  Word w1{2, 2};
  Word w2{3, 1};

  EXPECT_EQ(w0, w0);
  EXPECT_EQ(w1, w1);
  EXPECT_EQ(w2, w2);

  EXPECT_NE(w0, w1);
  EXPECT_NE(w0, w2);
  EXPECT_NE(w1, w2);
}

TEST(Word, CanBeComparedLess) {
  Word w0{0};
  Word w1{2, 2};
  Word w2{3, 1};

  EXPECT_LT(w0, w1);
  EXPECT_LT(w0, w2);
  EXPECT_LT(w0, w2);

  EXPECT_FALSE(w0 < w0);
  EXPECT_FALSE(w1 < w0);
  EXPECT_FALSE(w2 < w0);
  EXPECT_FALSE(w2 < w1);
}
TEST(Word, CanBeComparedLessEq) {
  Word w0{0};
  Word w1{2, 2};
  Word w2{3, 1};

  EXPECT_LE(w0, w0);
  EXPECT_LE(w0, w1);
  EXPECT_LE(w0, w2);
  EXPECT_LE(w0, w2);

  EXPECT_FALSE(w1 <= w0);
  EXPECT_FALSE(w2 <= w0);
  EXPECT_FALSE(w2 <= w1);
}

TEST(Word, CanBeComparedGreater) {
  Word w0{0};
  Word w1{2, 2};
  Word w2{3, 1};

  EXPECT_GT(w1, w0);
  EXPECT_GT(w2, w0);
  EXPECT_GT(w2, w0);

  EXPECT_FALSE(w0 > w0);
  EXPECT_FALSE(w0 > w1);
  EXPECT_FALSE(w0 > w2);
  EXPECT_FALSE(w1 > w2);
}

TEST(Word, CanBeComparedGreaterEq) {
  Word w0{0};
  Word w1{2, 2};
  Word w2{3, 1};

  EXPECT_GE(w0, w0);
  EXPECT_GE(w1, w0);
  EXPECT_GE(w2, w0);
  EXPECT_GE(w2, w0);

  EXPECT_FALSE(w0 >= w1);
  EXPECT_FALSE(w0 >= w2);
  EXPECT_FALSE(w1 >= w2);
}

TEST(Word, CanBeCompareSpaceShip) {
  Word w0{0};
  Word w1{2, 2};

  EXPECT_EQ(w0 <=> w0, std::strong_ordering::equal);
  EXPECT_EQ(w0 <=> w1, std::strong_ordering::less);
  EXPECT_EQ(w1 <=> w0, std::strong_ordering::greater);
}

TEST(Word, CanBeAdded) { EXPECT_EQ(Word{2} + Word{3}, Word{5}); }

TEST(Word, CanOverflowWhenAdded) { EXPECT_EQ(Word::kMax + Word{2}, Word{1}); }

TEST(Word, CanBeSubtracted) { EXPECT_EQ(Word{3} - Word{2}, Word{1}); }

TEST(Word, CanUnderflowWhenSubtracted) { EXPECT_EQ(Word{0} - Word{1}, Word::kMax); }

TEST(Word, CanBeMultiplied) { EXPECT_EQ(Word{2} * Word{3}, Word{6}); }

TEST(Word, CanOverflowWhenMultiplied) { EXPECT_EQ(Word::kMax * Word{2}, Word::kMax - Word{1}); }

TEST(Word, CanBeDivided) {
  Word w24{24};

  EXPECT_EQ(w24 / Word{8}, Word{3});  // no remainder
  EXPECT_EQ(w24 / Word{5}, Word{4});  // remainder
}

TEST(Word, CanBeDividedByZero) {
  // EVM requires: If the denominator is 0, the result will be 0.
  EXPECT_EQ(Word{42} / Word{0}, Word{0});
}

TEST(Word, CanUseModulo) {
  Word w24{24};

  EXPECT_EQ(w24 % Word{8}, Word{0});  // no remainder
  EXPECT_EQ(w24 % Word{5}, Word{4});  // remainder
}

TEST(Word, CanUseModuloWithZero) {
  // EVM requires: If the denominator is 0, the result will be 0.
  EXPECT_EQ(Word{42} % Word{0}, Word{0});
}

TEST(Word, CanBeShiftedLeft) {
  EXPECT_EQ(Word{0xF} << Word{4}, Word{0xF0});

  // EVM: requires: If shift is bigger than 255, returns 0.
  Word big_shift{0xff, 0x1};
  EXPECT_EQ(Word::kMax << big_shift, Word{0});
}

TEST(Word, CanBeShiftedRight) {
  Word w{0xF0};
  EXPECT_EQ(w >> Word{4}, Word{0xF});

  // EVM: requires: If shift is bigger than 255, returns 0.
  Word big_shift{0xff, 0x1};
  EXPECT_EQ(Word::kMax << big_shift, Word{0});
}

TEST(Word, CanBeBitwiseORed) {
  EXPECT_EQ(Word{0xF0} | Word{0x0F}, Word{0xFF});
  EXPECT_EQ(Word{0xFF} | Word{0xFF}, Word{0xFF});
}

TEST(Word, CanBeBitwiseANDed) {
  EXPECT_EQ(Word{0x0F} & Word{0x0F}, Word{0x0F});
  EXPECT_EQ(Word{0xFF} & Word{0x00}, Word{0x00});
}

TEST(Word, CanBeBitwiseXORed) {
  EXPECT_EQ(Word{0xF0} ^ Word{0x0F}, Word{0xFF});
  EXPECT_EQ(Word{0xFF} ^ Word{0xFF}, Word{0x00});
}

TEST(Word, CanBeBitwiseNOTed) {
  EXPECT_EQ(~Word{0}, Word::kMax);
  EXPECT_EQ(~Word::kMax, Word{0});
}

}  // namespace
}  // namespace tosca

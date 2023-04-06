#include "common/word.h"

#include <type_traits>

#include "gmock/gmock.h"
#include "gtest/gtest.h"

namespace tosca {
namespace {

using ::testing::PrintToString;
using ::testing::StrEq;

TEST(Word, PrintProducesHexString) {
  Word word{1, 2, 3, 0xAF};
  EXPECT_THAT(PrintToString(word), StrEq("010203AF00000000000000000000000000000000000000000000000000000000"));
}

TEST(Word, CanBeEqualtiyCompared) {
  Word w0{};
  Word w1{1};
  Word w2{2};

  EXPECT_EQ(w0, w0);
  EXPECT_EQ(w1, w1);
  EXPECT_EQ(w2, w2);

  EXPECT_NE(w0, w1);
  EXPECT_NE(w0, w2);
  EXPECT_NE(w1, w2);
}

}  // namespace
}  // namespace tosca

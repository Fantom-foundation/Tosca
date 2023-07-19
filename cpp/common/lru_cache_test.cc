#include "common/lru_cache.h"

#include <gtest/gtest.h>

namespace tosca::evmzero {
namespace {

TEST(LruCacheTest, Init) {
  LruCache<int, int, 32> cache;
  EXPECT_EQ(cache.GetSize(), 0);
}

TEST(LruCacheTest, GetMissing) {
  LruCache<int, int, 32> cache;
  EXPECT_EQ(cache.Get(0), nullptr);
}

TEST(LruCacheTest, Insert) {
  LruCache<int, int, 32> cache;

  bool inserted = cache.InsertOrAssign(0, 42);
  EXPECT_TRUE(inserted);
  EXPECT_EQ(cache.GetSize(), 1);
  EXPECT_EQ(*cache.Get(0), 42);
}

TEST(LruCacheTest, Assign) {
  LruCache<int, int, 32> cache;

  cache.InsertOrAssign(0, 42);
  bool inserted = cache.InsertOrAssign(0, 23);
  EXPECT_FALSE(inserted);
  EXPECT_EQ(cache.GetSize(), 1);
  EXPECT_EQ(*cache.Get(0), 23);
}

TEST(LruCacheTest, LeastRecentlyUsedRemoved) {
  {
    LruCache<int, int, 2> cache;
    cache.InsertOrAssign(0, 40);
    cache.InsertOrAssign(1, 41);

    cache.Get(0);
    cache.InsertOrAssign(2, 42);  // removes key 1
    EXPECT_EQ(cache.GetSize(), 2);
    EXPECT_EQ(*cache.Get(0), 40);
    EXPECT_EQ(*cache.Get(2), 42);
    EXPECT_EQ(cache.Get(1), nullptr);
  }

  {
    LruCache<int, int, 2> cache;
    cache.InsertOrAssign(0, 40);
    cache.InsertOrAssign(1, 41);

    cache.Get(1);
    cache.InsertOrAssign(2, 42);  // removes key 0
    EXPECT_EQ(cache.GetSize(), 2);
    EXPECT_EQ(*cache.Get(1), 41);
    EXPECT_EQ(*cache.Get(2), 42);
    EXPECT_EQ(cache.Get(0), nullptr);
  }
}

TEST(LruCacheTest, Clear) {
  LruCache<int, int, 32> cache;
  cache.InsertOrAssign(0, 42);
  cache.Clear();
  EXPECT_EQ(cache.GetSize(), 0);
}

}  // namespace
}  // namespace tosca::evmzero

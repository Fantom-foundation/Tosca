#pragma once

#include <cstddef>
#include <list>
#include <memory>
#include <mutex>
#include <unordered_map>
#include <optional>

#include "common/assert.h"

namespace tosca {

// This container serves as a key-value storage with a fixed maximum capacity.
// Adding elements beyond Capacity will cause the least recently used elements
// to be removed. This container is thread-safe.
template <typename Key, typename Value, size_t Capacity,  //
          typename Hash = std::hash<Key>, typename KeyEqual = std::equal_to<Key>>
class LruCache {
 public:
  // Retrieves the value with the given key and updates the least recently used
  // list. Returns nullopt when the key is not present.
  std::optional<Value> Get(const Key& key) {
    std::scoped_lock lock(mutex_);

    if (auto it = entries_.find(key); it != entries_.end()) {
      lru_.erase(it->second.lru_entry);
      lru_.push_front(key);
      it->second.lru_entry = lru_.begin();
      return it->second.value;
    } else {
      return std::nullopt;
    }
  }

  // Adds or updates the value with the given key. Removes the least recently
  // used element when Capacity is exceeded. Returns the added/updated value.
  Value InsertOrAssign(const Key& key, Value value) {
    std::scoped_lock lock(mutex_);

    if (auto it = entries_.find(key); it != entries_.end()) {
      it->second.value = std::move(value);
      return it->second.value;
    }

    if (entries_.size() == Capacity) {
      entries_.erase(lru_.back());
      lru_.pop_back();
    }
    lru_.push_front(key);

    auto [iter, _]  = entries_.insert_or_assign(key, Entry{
        .lru_entry = lru_.begin(),
        .value = std::move(value),
    });

    TOSCA_ASSERT(entries_.size() <= Capacity);
    TOSCA_ASSERT(entries_.size() == lru_.size());

    return iter->second.value;
  }

  // Tries to get the value with the given key. If the key is not contained,
  // creates and inserts a value by calling make_value and returns it. Removes
  // the least recently used element when Capacity is exceeded
  template <typename F>
  Value GetOrInsert(const Key& key, F make_value) {
    if (auto entry = Get(key)) {
      return *entry;
    } else {
      return InsertOrAssign(key, make_value());
    }
  }

  size_t GetSize() {
    std::scoped_lock lock(mutex_);
    return entries_.size();
  }

  constexpr size_t GetMaxSize() const { return Capacity; }

  void Clear() {
    std::scoped_lock lock(mutex_);
    entries_.clear();
    lru_.clear();
  }

 private:
  using LruList = std::list<Key>;

  struct Entry {
    typename LruList::const_iterator lru_entry;
    Value value;
  };

  std::mutex mutex_;
  std::unordered_map<Key, Entry, Hash, KeyEqual> entries_;
  LruList lru_;
};

}  // namespace tosca

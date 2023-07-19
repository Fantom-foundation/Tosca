#pragma once

#include <cstddef>
#include <list>
#include <memory>
#include <mutex>
#include <unordered_map>

#include "common/assert.h"

namespace tosca {

// This container serves as a key-value storage with a fixed maximum capacity.
// Adding elements beyond Capacity will cause the least recently used elements
// to be removed. This container is thread-safe.
template <typename Key, typename Value, size_t Capacity>
class LruCache {
 public:
  // Retrieves the value with the given key and updates the least recently used
  // list. Returns nullptr when the key is not present.
  std::shared_ptr<const Value> Get(const Key& key) {
    std::scoped_lock lock(mutex_);

    if (auto it = entries_.find(key); it != entries_.end()) {
      lru_.erase(it->second.lru_entry);
      lru_.push_front(key);
      it->second.lru_entry = lru_.begin();
      return it->second.value;
    } else {
      return nullptr;
    }
  }

  // Adds or updates the value with the given key. Removes the least recently
  // used element when Capacity is exceeded. Returns true if a new key was
  // inserted.
  bool InsertOrAssign(const Key& key, Value value) {
    std::scoped_lock lock(mutex_);

    const auto value_ptr = std::make_shared<Value>(std::move(value));

    if (auto it = entries_.find(key); it != entries_.end()) {
      it->second.value = value_ptr;
      return false;
    }

    if (entries_.size() == Capacity) {
      entries_.erase(lru_.back());
      lru_.pop_back();
    }
    lru_.push_front(key);

    entries_[key] = Entry{
        .lru_entry = lru_.begin(),
        .value = value_ptr,
    };

    TOSCA_ASSERT(entries_.size() <= Capacity);
    TOSCA_ASSERT(entries_.size() == lru_.size());

    return true;
  }

  size_t GetSize() {
    std::scoped_lock lock(mutex_);
    return entries_.size();
  }

  void Clear() {
    std::scoped_lock lock(mutex_);
    entries_.clear();
    lru_.clear();
  }

 private:
  using LruList = std::list<Key>;

  struct Entry {
    typename LruList::const_iterator lru_entry;
    std::shared_ptr<Value> value;
  };

  std::mutex mutex_;
  std::unordered_map<Key, Entry> entries_;
  LruList lru_;
};

}  // namespace tosca

// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#pragma once

#include <cstddef>
#include <list>
#include <memory>
#include <mutex>
#include <optional>
#include <vector>

#include "absl/container/flat_hash_map.h"

#include "common/assert.h"

namespace tosca {

// This container serves as a key-value storage with a fixed maximum capacity.
// Adding elements beyond Capacity will cause the least recently used elements
// to be removed. This container is thread-safe.
template <typename Key, typename Value, size_t Capacity,  //
          typename Hash = std::hash<Key>, typename KeyEqual = std::equal_to<Key>>
class LruCache {
 public:
  LruCache() { Clear(); }

  // Retrieves the value with the given key and updates the least recently used
  // list. Returns nullopt when the key is not present.
  std::optional<Value> Get(const Key& key) {
    std::scoped_lock lock(mutex_);
    if (auto it = index_.find(key); it != index_.end()) {
      auto entry = it->second;
      Touch(entry);
      return entry->value;
    }
    return std::nullopt;
  }

  // Adds or updates the value with the given key. Removes the least recently
  // used element when Capacity is exceeded. Returns the added/updated value.
  Value InsertOrAssign(const Key& key, Value value) {
    std::scoped_lock lock(mutex_);

    if (auto it = index_.find(key); it != index_.end()) {
      it->second->value = std::move(value);
      return it->second->value;
    }

    auto entry = GetNewHead();
    entry->key = key;
    entry->value = std::move(value);
    index_[key] = entry;

    TOSCA_ASSERT(index_.size() <= Capacity);

    return entry->value;
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

  constexpr size_t GetMaxSize() const { return Capacity; }

  void Clear() {
    std::scoped_lock lock(mutex_);
    entries_.clear();
    entries_.resize(Capacity);
    for (size_t i = 0; i < entries_.size(); i++) {
      entries_[i].pred = i > 0 ? &entries_[i - 1] : nullptr;
      entries_[i].succ = i < entries_.size() ? &entries_[i + 1] : nullptr;
    }
    head_ = &entries_[0];
    tail_ = &entries_[Capacity - 1];
    index_.clear();
    index_.reserve(Capacity);
  }

 private:
  struct Entry {
    Key key;
    Value value;
    Entry* pred;
    Entry* succ;
  };

  // Registers an access to an entry by moving it to the front of the LRU queue.
  void Touch(Entry* entry) {
    if (entry == head_) {
      return;
    }

    // Remove entry from current position in list.
    entry->pred->succ = entry->succ;

    if (entry->succ) {
      entry->succ->pred = entry->pred;
    } else {
      tail_ = entry->pred;
    }

    // Make the entry the new head.
    entry->pred = nullptr;
    entry->succ = head_;
    head_->pred = entry;
    head_ = entry;
  }

  Entry* GetNewHead() {
    // Remove tail element.
    auto new_tail = tail_->pred;
    new_tail->succ = nullptr;
    if (index_.size() >= entries_.size()) {
      index_.erase(tail_->key);
    }
    auto result = tail_;
    tail_ = new_tail;

    // Make the result the new head.
    result->pred = nullptr;
    result->succ = head_;
    head_->pred = result;
    head_ = result;
    return result;
  }

  std::mutex mutex_;
  std::vector<Entry> entries_;
  absl::flat_hash_map<Key, Entry*, Hash, KeyEqual> index_;

  Entry* head_;
  Entry* tail_;
};

}  // namespace tosca

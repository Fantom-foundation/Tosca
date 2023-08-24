#pragma once

#include <array>
#include <atomic>
#include <cstdint>
#include <initializer_list>
#include <mutex>
#include <ostream>
#include <vector>

#include "common/assert.h"
#include "vm/evmzero/uint256.h"

namespace tosca::evmzero {

// This data structure is used as the interpreter's stack during execution.
class Stack {
 public:
  Stack();
  Stack(const Stack&);
  Stack(Stack&&) = delete;
  Stack(std::initializer_list<uint256_t>);
  ~Stack();

  uint64_t GetSize() const { return uint64_t(end_ - top_); }
  uint64_t GetMaxSize() const { return kStackSize; }

  const uint256_t* GetBase() const { return end_; }
  uint256_t* GetTop() { return top_; }
  void SetTop(uint256_t* top) { top_ = top; }

  void Push(const uint256_t& value) {
    TOSCA_ASSERT(GetSize() < kStackSize);
    *(--top_) = value;
  }

  uint256_t Pop() {
    TOSCA_ASSERT(GetSize() > 0);
    return *(top_++);
  }

  Stack& operator=(const Stack&);
  Stack& operator=(Stack&&) = delete;

  // Accesses elements starting from the top; index 0 is the top element.
  uint256_t& operator[](size_t index) {
    TOSCA_ASSERT(index < GetSize());
    return top_[index];
  }
  const uint256_t& operator[](size_t index) const { return const_cast<Stack&>(*this)[index]; }

  friend bool operator==(const Stack&, const Stack&);
  friend bool operator!=(const Stack&, const Stack&);

 private:
  static constexpr size_t kStackSize = 1024;

  // The type retaining the actual storage for the stack.
  class Data {
   public:
    Data() {}  // needed to avoid zero-initialization of data_ and use default-initialization instead (see t.ly/MkD0M).

    // Retrieves an instance from a global re-use list.
    static Data* Get();

    // Releases this instance to be re-used by another stack.
    void Release();

    uint256_t* end() { return reinterpret_cast<uint256_t*>(data_.end()); }
    size_t size() { return kStackSize; }

   private:
    static Data* free_list;
    static std::mutex free_list_mutex;

    // An aligned blob of not auto-initialized data. Stack memory does not have
    // to be initialized, since any read is preceeded by a push or dup.
    alignas(sizeof(uint256_t)) std::array<std::byte, kStackSize * sizeof(uint256_t)> data_;

    // A next-pointer to be used when enqueuing instances in the free list.
    Data* next_ = nullptr;
  };

  Data* const data_ = nullptr;
  uint256_t* top_ = nullptr;
  uint256_t* const end_ = nullptr;
};

std::ostream& operator<<(std::ostream&, const Stack&);

}  // namespace tosca::evmzero

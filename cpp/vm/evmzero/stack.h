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

class StackView {
 public:
 private:
  friend class Stack;
  StackView(uint256_t* top) : top_(top) {}
  uint256_t* top_;
};

// This data structure is used as the interpreter's stack during execution.
class Stack {
 public:
  static constexpr size_t kStackSize = 1024;

  Stack();
  Stack(const Stack&);
  Stack(Stack&&) = delete;
  Stack(std::initializer_list<uint256_t>);
  ~Stack();

  uint64_t GetSize() const { return uint64_t(end_ - top_); }
  uint64_t GetMaxSize() const { return kStackSize; }

  void Push(const uint256_t& value) {
    TOSCA_ASSERT(GetSize() < kStackSize);
    *(--top_) = value;
  }

  uint256_t Pop() {
    TOSCA_ASSERT(GetSize() > 0);
    return *(top_++);
  }

  void Pop(int32_t num_values) { top_ += num_values; }

  StackView GetView() { return StackView(top_); }

  uint256_t* Peek() { return top_; }

  uint256_t* Base() { return end_; }

  // DO NOT SUBMIT!
  void SetTop(uint256_t* top) { top_ = top; }

  template <size_t N>
  void Swap() {
    TOSCA_ASSERT(N < GetSize());
    auto tmp = top_[N];
    top_[N] = top_[0];
    top_[0] = tmp;
  }

  template <size_t N>
  void Dup() {
    TOSCA_ASSERT(N - 1 < GetSize());
    Push(top_[N - 1]);
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
    static_assert(sizeof(uint256_t) * kStackSize * 2 == 1<<16);
    alignas(1<<16) std::array<std::byte, kStackSize * sizeof(uint256_t)> data_;

    // A next-pointer to be used when enqueuing instances in the free list.
    Data* next_ = nullptr;
  };

  Data* const data_ = nullptr;
  uint256_t* top_ = nullptr;
  uint256_t* const end_ = nullptr;
};

std::ostream& operator<<(std::ostream&, const Stack&);

}  // namespace tosca::evmzero

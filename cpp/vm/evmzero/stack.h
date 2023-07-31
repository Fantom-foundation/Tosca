#pragma once

#include <cstdint>
#include <initializer_list>
#include <memory>
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

  uint16_t GetSize() const { return size_; }
  uint64_t GetMaxSize() const { return kStackSize; }

  void Push(const uint256_t& value) {
    TOSCA_ASSERT(GetSize() < kStackSize);
    *(--top_) = value;
    size_++;
  }

  uint256_t& Pop() {
    TOSCA_ASSERT(GetSize() > 0);
    size_--;
    return *(top_++);
  }

  uint256_t& Peek() {
    TOSCA_ASSERT(GetSize() > 0);
    return *top_;
  }

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
  static constexpr size_t kStackSize = 1024;

  struct Data {
    Data() {}  // Required to disable default initialization of data_ array

    uint256_t* end() { return reinterpret_cast<uint256_t*>(data_) + kStackSize; }
    std::size_t size() const { return kStackSize; }

    // Provides uninitialized, reinterpretable stack storage.
    alignas(sizeof(uint256_t)) std::byte data_[kStackSize * sizeof(uint256_t)];
  };

  std::unique_ptr<Data> stack_;
  uint256_t* top_ = nullptr;
  uint256_t* const end_ = nullptr;
  uint16_t size_ = 0;
};

std::ostream& operator<<(std::ostream&, const Stack&);

}  // namespace tosca::evmzero

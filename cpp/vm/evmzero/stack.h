#pragma once

#include <array>
#include <cassert>
#include <cstdint>
#include <initializer_list>

#include "vm/evmzero/uint256.h"

namespace tosca::evmzero {

// This data structure is used as the interpreter's stack during execution.
class Stack {
 public:
  uint64_t GetSize() const { return position_; }
  uint64_t GetMaxSize() const { return stack_.size(); }

  void Push(uint256_t value) {
    assert(position_ < stack_.size());
    stack_[position_++] = value;
  }

  uint256_t Pop() {
    assert(position_ > 0);
    return stack_[--position_];
  }

  void SetElements(std::initializer_list<uint256_t>);

  // Accesses elements starting from the top; index 0 is the top element.
  uint256_t& operator[](size_t index) {
    assert(index < position_);
    return stack_[position_ - 1 - index];
  }
  const uint256_t& operator[](size_t index) const { return const_cast<Stack&>(*this)[index]; }

  friend bool operator==(const Stack&, const Stack&);
  friend bool operator!=(const Stack&, const Stack&);

 private:
  std::array<uint256_t, 1024> stack_;
  uint64_t position_ = 0;
};

}  // namespace tosca::evmzero

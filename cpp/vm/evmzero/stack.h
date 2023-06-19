#pragma once

#include <array>
#include <cstdint>
#include <initializer_list>
#include <ostream>

#include "common/assert.h"
#include "vm/evmzero/uint256.h"

namespace tosca::evmzero {

// This data structure is used as the interpreter's stack during execution.
class Stack {
 public:
  Stack() = default;
  Stack(std::initializer_list<uint256_t>);

  uint64_t GetSize() const { return position_; }
  uint64_t GetMaxSize() const { return stack_.size(); }

  void Push(uint256_t value) {
    TOSCA_ASSERT(position_ < stack_.size());
    stack_[position_++] = value;
  }

  uint256_t Pop() {
    TOSCA_ASSERT(position_ > 0);
    return stack_[--position_];
  }

  // Accesses elements starting from the top; index 0 is the top element.
  uint256_t& operator[](size_t index) {
    TOSCA_ASSERT(index < position_);
    return stack_[position_ - 1 - index];
  }
  const uint256_t& operator[](size_t index) const { return const_cast<Stack&>(*this)[index]; }

  friend bool operator==(const Stack&, const Stack&);
  friend bool operator!=(const Stack&, const Stack&);

 private:
  std::array<uint256_t, 1024> stack_;
  uint64_t position_ = 0;
};

std::ostream& operator<<(std::ostream&, const Stack&);

}  // namespace tosca::evmzero

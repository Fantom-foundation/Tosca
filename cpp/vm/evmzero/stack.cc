#include "vm/evmzero/stack.h"

#include <algorithm>

namespace tosca::evmzero {

Stack::Stack() : stack_(std::make_unique<std::array<uint256_t, 1024>>()), top_(stack_->end()), end_(top_) {}

Stack::Stack(std::initializer_list<uint256_t> elements) : Stack() {
  TOSCA_ASSERT(elements.size() <= stack_->size());
  for (auto cur : elements) {
    Push(cur);
  }
}

Stack::Stack(const Stack& other) : Stack() {
  *stack_ = *other.stack_;
  top_ = stack_->end() - other.GetSize();
}

bool operator==(const Stack& a, const Stack& b) { return std::equal(a.top_, a.end_, b.top_, b.end_); }

bool operator!=(const Stack& a, const Stack& b) { return !(a == b); }

std::ostream& operator<<(std::ostream& out, const Stack& stack) {
  out << "T[ ";
  for (uint64_t i = 0; i < stack.GetSize(); ++i) {
    if (i != 0) {
      out << ", ";
    }
    out << stack[i];
  }
  return out << " ]B";
}

}  // namespace tosca::evmzero

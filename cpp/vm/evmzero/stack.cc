#include "vm/evmzero/stack.h"

#include <algorithm>

namespace tosca::evmzero {

Stack::Stack(std::initializer_list<uint256_t> elements) {
  assert(elements.size() <= stack_.size());
  std::copy(elements.begin(), elements.end(), stack_.begin());
  position_ = elements.size();
}

bool operator==(const Stack& a, const Stack& b) {
  return std::equal(a.stack_.data(), a.stack_.data() + a.position_,  //
                    b.stack_.data(), b.stack_.data() + b.position_);
}

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

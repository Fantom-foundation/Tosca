#include "vm/evmzero/stack.h"

#include <algorithm>

namespace tosca::evmzero {

void Stack::SetElements(std::initializer_list<uint256_t> elements) {
  assert(elements.size() <= stack_.size());
  std::copy(elements.begin(), elements.end(), stack_.begin());
  position_ = elements.size();
}

bool operator==(const Stack& a, const Stack& b) {
  return std::equal(a.stack_.begin(), a.stack_.begin() + static_cast<ptrdiff_t>(a.position_),  //
                    b.stack_.begin(), b.stack_.begin() + static_cast<ptrdiff_t>(b.position_));
}

bool operator!=(const Stack& a, const Stack& b) { return !(a == b); }

}  // namespace tosca::evmzero

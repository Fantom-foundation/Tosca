//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public Licence v3.
//

#include "vm/evmzero/stack.h"

#include <algorithm>
#include <mutex>

namespace tosca::evmzero {

Stack::Stack() : data_(Data::Get()), top_(data_->end()), end_(top_) {}

Stack::Stack(std::initializer_list<uint256_t> elements) : Stack() {
  TOSCA_ASSERT(elements.size() <= data_->size());
  for (auto cur : elements) {
    Push(cur);
  }
}

Stack::Stack(const Stack& other) : Stack() {
  *data_ = *other.data_;
  top_ = data_->end() - other.GetSize();
}

Stack::~Stack() {
  if (data_) {
    data_->Release();
  }
}

Stack& Stack::operator=(const Stack& other) {
  if (this == &other) {
    return *this;
  }
  *data_ = *other.data_;
  top_ = data_->end() - other.GetSize();
  return *this;
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

// freeList is the head of a synchronized linked list of Data instances that
// are free to be reused by arbitrary threads. The list size is limited to the maximum
// number simultaneously used stack instances and once allocated stacks are never released.
Stack::Data* Stack::Data::free_list = nullptr;
std::mutex Stack::Data::free_list_mutex;

Stack::Data* Stack::Data::Get() {
  std::lock_guard guard(free_list_mutex);
  if (free_list == nullptr) {
    return new Data();
  }
  auto res = free_list;
  free_list = res->next_;
  return res;
}

void Stack::Data::Release() {
  std::lock_guard guard(free_list_mutex);
  next_ = free_list;
  free_list = this;
}

}  // namespace tosca::evmzero

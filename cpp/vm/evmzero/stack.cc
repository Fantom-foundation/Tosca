#include "vm/evmzero/stack.h"

#include <algorithm>

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

// kFreeList is the head of a lock-free synchronized linked list of Data instances that
// are free to be reused by arbitrary threads. The list size is limited to the maximum
// number simultaneously used stack instances and once allocated stacks are never released.
std::atomic<Stack::Data*> Stack::Data::freeList = nullptr;

Stack::Data* Stack::Data::Get() {
  Data* res = freeList.load(std::memory_order_relaxed);
  for (;;) {
    if (res == nullptr) {
      return new Data();
    }
    // Try to retrieve the instance by updating the head pointer. If successful,
    // the instance can be used by this thread. If not, res is updated to the new
    // head of the list.
    if (freeList.compare_exchange_weak(res, res->next_)) {
      return res;
    }
  }
}

void Stack::Data::Release() {
  // On the first call next_ is updated, and on subsequent calls the head is replaced.
  while (!freeList.compare_exchange_weak(next_, this, std::memory_order_relaxed, std::memory_order_relaxed)) {
  }
}

}  // namespace tosca::evmzero

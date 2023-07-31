#include "vm/evmzero/stack.h"

#include <algorithm>
#include <atomic>

namespace tosca::evmzero {

Stack::Stack() : data_(GetData()), top_(data_->end()), end_(top_) {}

Stack::Stack(std::initializer_list<uint256_t> elements) : Stack() {
  TOSCA_ASSERT(elements.size() <= stack_->size());
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
    Release(data_);
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

std::atomic<Stack::Data*> Stack::kFreeList = nullptr;

Stack::Data* Stack::GetData() {
  Data* res = kFreeList.load(std::memory_order_relaxed);
  for (;;) {
    if (res == nullptr) {
      return new Data();
    }
    if (kFreeList.compare_exchange_weak(res, res->next_)) {
      return res;
    }
  }
}

void Stack::Release(Data* data) {
  while (!kFreeList.compare_exchange_weak(data->next_, data, std::memory_order_relaxed, std::memory_order_relaxed)) {
  }
}

}  // namespace tosca::evmzero

#pragma once

#include <algorithm>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <initializer_list>
#include <ostream>
#include <span>
#include <vector>

#include <ethash/keccak.hpp>

#include "vm/evmzero/uint256.h"

namespace tosca::evmzero {

// This data structure is used as the interpreter's memory during execution.
//
// Invariant: memory size is always a multiple of 32.
class Memory {
 public:
  Memory() : data_(reinterpret_cast<uint8_t*>(std::malloc(kInitialSize))) {}

  Memory(std::initializer_list<uint8_t>);

  Memory(const Memory&);
  Memory(Memory&&) = delete;

  ~Memory() {
    std::free(data_);
  }

  size_t GetSize() const { return size_; }

  // Get a span for the given memory offset and size that can be used for
  // reading or writing. Grows memory automatically, unless size == 0.
  std::span<uint8_t> GetSpan(uint64_t offset, uint64_t size) {
    Grow(offset, size);
    return {data_ + offset, size};
  }

  // Read from the given buffer into memory at memory_offset. Grows memory
  // automatically, unless buffer.size() == 0.
  void ReadFrom(std::span<const uint8_t> buffer, uint64_t memory_offset) {
    Grow(memory_offset, buffer.size());
    std::copy(buffer.begin(), buffer.end(), data_ + memory_offset);
  }

  // Read from the given buffer into memory at memory_offset. Will write exactly
  // memory_write_size bytes. If the provided buffer is smaller than
  // memory_write_size, it is implicitly padded with zero values. Grows memory
  // automatically, unless memory_write_size == 0.
  void ReadFromWithSize(std::span<const uint8_t> buffer, uint64_t memory_offset, uint64_t memory_write_size) {
    Grow(memory_offset, memory_write_size);

    auto bytes_to_copy = std::min<uint64_t>(buffer.size(), memory_write_size);
    std::copy_n(buffer.data(), bytes_to_copy, data_ + memory_offset);

    std::fill_n(data_ + memory_offset + bytes_to_copy, memory_write_size - bytes_to_copy, 0);
  }

  // Writes to the given buffer from memory at memory_offset. Grows memory
  // automatically, unless buffer.size() == 0.
  void WriteTo(std::span<uint8_t> buffer, uint64_t memory_offset) {
    Grow(memory_offset, buffer.size());
    std::copy_n(data_ + memory_offset, buffer.size(), buffer.data());
  }

  // Grow memory to accommodate offset + size bytes. Memory is not grown when
  // size == 0.
  void Grow(uint64_t offset, uint64_t size) {
    if (size != 0) {
      Grow(offset+size);
    }
  }

  uint8_t& operator[](size_t index) { return data_[index]; }
  const uint8_t& operator[](size_t index) const { return data_[index]; }

  bool operator==(const Memory&) const;

 private:
  const static size_t kInitialSize = 4*1024;

  void Grow(size_t new_size) {
    if (new_size <= size_) [[unlikely]] {
      return;
    }

    // Size must be a multiple of 32.
    new_size = ((new_size + 31) / 32) * 32;

    // Make sure there is enough capacity.
    while (new_size > capacity_) {
      // grow by a factor of 1.5
      capacity_ = (capacity_ * 3)/2;  
      data_ = reinterpret_cast<uint8_t*>(std::realloc(data_, capacity_));
      // TODO: check that data_ is not null.
    }

    // Initialize only new data range.
    std::memset(data_ + size_, 0, new_size - size_);
    size_ = new_size;
  }

  size_t size_ = 0;
  size_t capacity_ = kInitialSize;
  uint8_t* data_ = nullptr;
};

/*
// This data structure is used as the interpreter's memory during execution.
//
// Invariant: memory size is always a multiple of 32.
class Memory {
 public:
  Memory() = default;
  Memory(std::initializer_list<uint8_t>);

  uint64_t GetSize() const { return memory_.size(); }

  // Get a span for the given memory offset and size that can be used for
  // reading or writing. Grows memory automatically, unless size == 0.
  std::span<uint8_t> GetSpan(uint64_t offset, uint64_t size) {
    Grow(offset, size);
    return {memory_.data() + offset, size};
  }

  // Read from the given buffer into memory at memory_offset. Grows memory
  // automatically, unless buffer.size() == 0.
  void ReadFrom(std::span<const uint8_t> buffer, uint64_t memory_offset) {
    Grow(memory_offset, buffer.size());
    std::copy(buffer.begin(), buffer.end(), memory_.data() + memory_offset);
  }

  // Read from the given buffer into memory at memory_offset. Will write exactly
  // memory_write_size bytes. If the provided buffer is smaller than
  // memory_write_size, it is implicitly padded with zero values. Grows memory
  // automatically, unless memory_write_size == 0.
  void ReadFromWithSize(std::span<const uint8_t> buffer, uint64_t memory_offset, uint64_t memory_write_size) {
    Grow(memory_offset, memory_write_size);

    auto bytes_to_copy = std::min<uint64_t>(buffer.size(), memory_write_size);
    std::copy_n(buffer.data(), bytes_to_copy, memory_.data() + memory_offset);

    std::fill_n(memory_.data() + memory_offset + bytes_to_copy, memory_write_size - bytes_to_copy, 0);
  }

  // Writes to the given buffer from memory at memory_offset. Grows memory
  // automatically, unless buffer.size() == 0.
  void WriteTo(std::span<uint8_t> buffer, uint64_t memory_offset) {
    Grow(memory_offset, buffer.size());
    std::copy_n(memory_.data() + memory_offset, buffer.size(), buffer.data());
  }

  // Grow memory to accommodate offset + size bytes. Memory is not grown when
  // size == 0.
  void Grow(uint64_t offset, uint64_t size) {
    if (size != 0) {
      const auto new_size = offset + size;
      if (new_size > memory_.size()) {
        memory_.resize(((new_size + 31) / 32) * 32);
      }
    }
  }

  uint8_t& operator[](size_t index) { return memory_[index]; }
  const uint8_t& operator[](size_t index) const { return memory_[index]; }

  bool operator==(const Memory&) const = default;

 private:
  std::vector<uint8_t> memory_;
};
*/
std::ostream& operator<<(std::ostream&, const Memory&);

}  // namespace tosca::evmzero

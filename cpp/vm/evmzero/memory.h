#pragma once

#include <algorithm>
#include <cassert>
#include <cstdint>
#include <initializer_list>
#include <ostream>
#include <span>
#include <vector>

namespace tosca::evmzero {

// This data structure is used as the interpreter's memory during execution.
class Memory {
 public:
  Memory() = default;
  Memory(std::initializer_list<uint8_t>);

  uint64_t GetSize() const { return memory_.size(); }

  void Grow(size_t new_size) {
    if (new_size > memory_.size()) {
      memory_.resize(new_size);
    }
  }

  void SetMemory(std::initializer_list<uint8_t>);

  // Read from the given buffer into memory at memory_offset. Grows memory
  // automatically.
  void ReadFrom(std::span<const uint8_t> buffer, uint64_t memory_offset) {
    Grow(memory_offset + buffer.size());
    std::copy(buffer.begin(), buffer.end(), memory_.data() + memory_offset);
  }

  // Read from the given buffer into memory at memory_offset. Will write exactly
  // memory_write_size bytes. If the provided buffer is smaller than
  // memory_write_size, it is implicitly padded with zero values. Grows memory
  // automatically.
  void ReadFromWithSize(std::span<const uint8_t> buffer, uint64_t memory_offset, uint64_t memory_write_size) {
    Grow(memory_offset + memory_write_size);

    auto bytes_to_copy = std::min<uint64_t>(buffer.size(), memory_write_size);
    std::copy_n(buffer.data(), bytes_to_copy, memory_.data() + memory_offset);

    std::fill_n(memory_.data() + memory_offset + bytes_to_copy, memory_write_size - bytes_to_copy, 0);
  }

  // Writes to the given buffer from memory at memory_offset. Grows memory
  // automatically.
  void WriteTo(std::span<uint8_t> buffer, uint64_t memory_offset) {
    Grow(memory_offset + buffer.size());
    std::copy_n(memory_.data() + memory_offset, buffer.size(), buffer.data());
  }

  uint8_t& operator[](size_t index) { return memory_[index]; }
  const uint8_t& operator[](size_t index) const { return memory_[index]; }

  bool operator==(const Memory&) const = default;

 private:
  std::vector<uint8_t> memory_;
};

std::ostream& operator<<(std::ostream&, const Memory&);

}  // namespace tosca::evmzero

#pragma once

#include <algorithm>
#include <cstdint>
#include <initializer_list>
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

  // Writes to the given buffer from memory at memory_offset. Grows memory
  // automatically.
  void WriteTo(std::span<uint8_t> buffer, uint64_t memory_offset) {
    Grow(memory_offset + buffer.size());
    std::copy_n(memory_.data() + memory_offset, buffer.size(), buffer.data());
  }

  // Accesses byte in memory at offset. Grows memory automatically.
  uint8_t& operator[](size_t offset) {
    Grow(offset + 1);
    return memory_[offset];
  }

  bool operator==(const Memory&) const = default;

 private:
  std::vector<uint8_t> memory_;
};

}  // namespace tosca::evmzero

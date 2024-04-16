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

#pragma once

#include <algorithm>
#include <cstdint>
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
  Memory() = default;
  Memory(std::initializer_list<uint8_t>);
  Memory(std::span<const uint8_t>);

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

std::ostream& operator<<(std::ostream&, const Memory&);

}  // namespace tosca::evmzero

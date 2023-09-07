#include "vm/evmzero/memory.h"

#include <cstring>
#include <iomanip>

namespace tosca::evmzero {

Memory::Memory(std::initializer_list<uint8_t> init) : Memory() {
  // Ensure size is a multiple of 32.
  auto size = ((init.size() + 31) / 32) * 32;
  Grow(size);
  ReadFrom(init, 0);
}

Memory::Memory(const Memory& other) : size_(other.size_), capacity_(other.capacity_) {
  data_ = reinterpret_cast<uint8_t*>(std::malloc(capacity_));
  std::memcpy(data_, other.data_, capacity_);
}


std::ostream& operator<<(std::ostream& out, const Memory& memory) {
  const auto flag_backup = out.flags();
  out << std::hex;

  for (size_t i = 0; i < memory.GetSize(); ++i) {
    if (i % 8 == 0) {
      out << "\n"
          << "0x" << std::setfill('0') << std::setw(4) << i << ": ";
    }
    out << std::setw(2) << static_cast<int>(memory[i]) << " ";
  }

  out.flags(flag_backup);
  return out;
}

bool Memory::operator==(const Memory& other) const {
  if (this == &other) return true;
  if (size_ != other.size_) return false;
  return std::memcmp(data_, other.data_, size_) == 0;
}

}  // namespace tosca::evmzero

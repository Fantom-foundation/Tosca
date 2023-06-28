#include "vm/evmzero/memory.h"

#include <iomanip>

namespace tosca::evmzero {

Memory::Memory(std::initializer_list<uint8_t> init) : memory_(init) {
  // Ensure size is a multiple of 32.
  memory_.resize(((init.size() + 31) / 32) * 32);
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

}  // namespace tosca::evmzero

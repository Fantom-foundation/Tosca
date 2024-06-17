// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#include "vm/evmzero/memory.h"

#include <iomanip>

namespace tosca::evmzero {

Memory::Memory(std::initializer_list<uint8_t> init) : memory_(init) {
  // Ensure size is a multiple of 32.
  memory_.resize(((init.size() + 31) / 32) * 32);
}

Memory::Memory(std::span<const uint8_t> init) : memory_(init.begin(), init.end()) {
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

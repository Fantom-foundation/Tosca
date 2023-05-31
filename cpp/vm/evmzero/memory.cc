#include "vm/evmzero/memory.h"

namespace tosca::evmzero {

Memory::Memory(std::initializer_list<uint8_t> init) : memory_(init) {}

void Memory::SetMemory(std::initializer_list<uint8_t> elements) { memory_.assign(elements.begin(), elements.end()); }

}  // namespace tosca::evmzero

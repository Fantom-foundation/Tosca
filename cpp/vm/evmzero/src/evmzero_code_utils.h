#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace tosca::evmzero {

[[nodiscard]] std::vector<uint8_t> ByteCodeStringToBinary(const std::string& code) noexcept;

}  // namespace tosca::evmzero

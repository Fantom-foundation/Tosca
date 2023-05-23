#include "evmzero_code_utils.h"

#include <cstdio>

#include "evmzero.h"

namespace tosca::evmzero {

[[nodiscard]] std::vector<uint8_t> ByteCodeStringToBinary(const std::string& code) noexcept {
  std::vector<uint8_t> ret(code.size() / 2);
  for (size_t count = 0; count < code.size() / 2; count++) {
    sscanf(code.data() + 2 * count, "%2hhx", ret.data() + count);
  }
  // we always assume that all code ends on STOP
  ret.push_back(op::STOP);
  return ret;
}

}  // namespace tosca::evmzero

#include <evmc/evmc.h>
#include <evmc/utils.h>

namespace tosca::evmzero {

extern "C" {
EVMC_EXPORT evmc_vm* evmc_create_evmzero() noexcept { return nullptr; }
}

}  // namespace tosca::evmzero

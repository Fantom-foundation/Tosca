#include "evmzero.h"

#include <evmc/evmc.h>
#include <evmc/utils.h>

#include "evmzero_interpreter.h"

namespace tosca::evmzero {

class VM : public evmc_vm {
 public:
  constexpr VM() noexcept
      : evmc_vm{
            .abi_version = EVMC_ABI_VERSION,
            .name = "evmzero",
            .version = "0.1.0",

            .destroy = [](evmc_vm* vm) { delete static_cast<VM*>(vm); },

            .execute = [](evmc_vm* vm, const evmc_host_interface* host, evmc_host_context* host_context,
                          evmc_revision revision, const evmc_message* message, const uint8_t* code,
                          size_t code_size) -> evmc_result {
              return static_cast<VM*>(vm)->execute({code, code_size}, message, *host, host_context, revision);
            },

            .get_capabilities = [](evmc_vm*) -> evmc_capabilities_flagset { return EVMC_CAPABILITY_EVM1; },
        } {}

  evmc_result execute(std::span<const uint8_t> code, const evmc_message* message,                  //
                      const evmc_host_interface& host_interface, evmc_host_context* host_context,  //
                      evmc_revision) {
    Context ctx(code, message, host_interface, host_context);
    ctx.gas = static_cast<uint64_t>(message->gas);

    RunInterpreter(ctx);

    evmc_result result{
        .status_code = EVMC_SUCCESS,
        .gas_left = static_cast<int64_t>(ctx.gas),
        .release = [](const evmc_result* result) { delete[] result->output_data; },
    };

    // Move return data to a dedicated buffer so we can release the context.
    if (!ctx.return_data.empty()) {
      auto* output_buffer = new uint8_t[ctx.return_data.size()];
      std::copy(ctx.return_data.begin(), ctx.return_data.end(), output_buffer);
      result.output_data = output_buffer;
      result.output_size = ctx.return_data.size();
    }

    return result;
  }
};

extern "C" {
EVMC_EXPORT evmc_vm* evmc_create_evmzero() noexcept { return new VM; }
}  // extern "C"

}  // namespace tosca::evmzero

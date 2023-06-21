#include <string_view>

#include <evmc/evmc.h>
#include <evmc/utils.h>

#include "vm/evmzero/interpreter.h"

namespace tosca::evmzero {

evmc_status_code ToEvmcStatusCode(RunState state) {
  switch (state) {
    case RunState::kRunning:
      return EVMC_FAILURE;
    case RunState::kDone:
      return EVMC_SUCCESS;
    case RunState::kRevert:
      return EVMC_REVERT;
    case RunState::kInvalid:
      return EVMC_INVALID_INSTRUCTION;
    case RunState::kErrorOpcode:
      return EVMC_INVALID_INSTRUCTION;
    case RunState::kErrorGas:
      return EVMC_OUT_OF_GAS;
    case RunState::kErrorStackUnderflow:
      return EVMC_STACK_UNDERFLOW;
    case RunState::kErrorStackOverflow:
      return EVMC_STACK_OVERFLOW;
    case RunState::kErrorJump:
      return EVMC_BAD_JUMP_DESTINATION;
    case RunState::kErrorCall:
      return EVMC_CALL_DEPTH_EXCEEDED;
    case RunState::kErrorCreate:
      return EVMC_FAILURE;
    case RunState::kErrorStaticCall:
      return EVMC_STATIC_MODE_VIOLATION;
  }
  return EVMC_FAILURE;
}

// This class represents the evmzero virtual machine (VM) which consists
// primarily of the evmzero interpreter. This class connects evmzero to the host
// infrastructure via the evmc interface.
class VM : public evmc_vm {
 public:
  constexpr VM() noexcept
      : evmc_vm{
            .abi_version = EVMC_ABI_VERSION,
            .name = "evmzero",
            .version = "0.1.0",

            .destroy = [](evmc_vm* vm) { delete static_cast<VM*>(vm); },

            .execute = [](evmc_vm* vm, const evmc_host_interface* host_interface, evmc_host_context* host_context,
                          evmc_revision revision, const evmc_message* message, const uint8_t* code,
                          size_t code_size) -> evmc_result {
              return static_cast<VM*>(vm)->Execute({code, code_size}, message, host_interface, host_context, revision);
            },

            .get_capabilities = [](evmc_vm*) -> evmc_capabilities_flagset { return EVMC_CAPABILITY_EVM1; },

            .set_option = [](evmc_vm* vm, char const* name, char const* value) -> evmc_set_option_result {
              return static_cast<VM*>(vm)->SetOption(name, value);
            },
        } {}

  evmc_result Execute(std::span<const uint8_t> code, const evmc_message* message,                  //
                      const evmc_host_interface* host_interface, evmc_host_context* host_context,  //
                      evmc_revision revision) {
    const InterpreterArgs interpreter_args{
        .code = code,
        .message = message,
        .host_interface = host_interface,
        .host_context = host_context,
        .revision = revision,
    };
    InterpreterResult interpreter_result;
    if (tracing_enabled_) {
      interpreter_result = Interpret<true>(interpreter_args);
    } else {
      interpreter_result = Interpret<false>(interpreter_args);
    }

    // Move output data to a dedicated buffer so we can release the interpreter
    // result.
    uint8_t* output_data = nullptr;
    if (!interpreter_result.return_data.empty()) {
      output_data = new uint8_t[interpreter_result.return_data.size()];
      std::copy(interpreter_result.return_data.begin(), interpreter_result.return_data.end(), output_data);
    }

    return {
        .status_code = ToEvmcStatusCode(interpreter_result.state),
        .gas_left = interpreter_result.remaining_gas,
        .gas_refund = interpreter_result.refunded_gas,
        .output_data = output_data,
        .output_size = interpreter_result.return_data.size(),
        .release = [](const evmc_result* result) { delete[] result->output_data; },
    };
  }

  evmc_set_option_result SetOption(std::string_view name, std::string_view value) {
    if (name == "tracing") {
      if (value == "true") {
        tracing_enabled_ = true;
        return EVMC_SET_OPTION_SUCCESS;
      } else if (value == "false") {
        tracing_enabled_ = false;
        return EVMC_SET_OPTION_SUCCESS;
      } else {
        return EVMC_SET_OPTION_INVALID_VALUE;
      }
    } else {
      return EVMC_SET_OPTION_INVALID_NAME;
    }
  }

 private:
  bool tracing_enabled_ = false;
};

extern "C" {
EVMC_EXPORT evmc_vm* evmc_create_evmzero() noexcept { return new VM; }
}

}  // namespace tosca::evmzero

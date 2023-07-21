#include <cstdint>
#include <iostream>
#include <span>
#include <string_view>

#include <evmc/evmc.h>
#include <evmc/utils.h>

#include "common/lru_cache.h"
#include "vm/evmzero/interpreter.h"
#include "vm/evmzero/opcodes.h"
#include "vm/evmzero/profiler.h"

namespace tosca::evmzero {

evmc_status_code ToEvmcStatusCode(RunState state) {
  switch (state) {
    case RunState::kRunning:
      return EVMC_FAILURE;
    case RunState::kDone:
      return EVMC_SUCCESS;
    case RunState::kReturn:
      return EVMC_SUCCESS;
    case RunState::kRevert:
      return EVMC_REVERT;
    case RunState::kInvalid:
      return EVMC_INVALID_INSTRUCTION;
    case RunState::kErrorOpcode:
      return EVMC_UNDEFINED_INSTRUCTION;
    case RunState::kErrorGas:
      return EVMC_OUT_OF_GAS;
    case RunState::kErrorStackUnderflow:
      return EVMC_STACK_UNDERFLOW;
    case RunState::kErrorStackOverflow:
      return EVMC_STACK_OVERFLOW;
    case RunState::kErrorJump:
      return EVMC_BAD_JUMP_DESTINATION;
    case RunState::kErrorReturnDataCopyOutOfBounds:
      return EVMC_INVALID_MEMORY_ACCESS;
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
  VM()
  noexcept
      : evmc_vm{
            .abi_version = EVMC_ABI_VERSION,
            .name = "evmzero",
            .version = "0.1.0",

            .destroy = [](evmc_vm* vm) { delete static_cast<VM*>(vm); },

            .execute = [](evmc_vm* vm, const evmc_host_interface* host_interface, evmc_host_context* host_context,
                          evmc_revision revision, const evmc_message* message, const evmc_bytes32* code_hash,
                          const uint8_t* code, size_t code_size) -> evmc_result {
              return static_cast<VM*>(vm)->Execute({code, code_size}, code_hash, message, host_interface, host_context,
                                                   revision);
            },

            .get_capabilities = [](evmc_vm*) -> evmc_capabilities_flagset { return EVMC_CAPABILITY_EVM1; },

            .set_option = [](evmc_vm* vm, char const* name, char const* value) -> evmc_set_option_result {
              return static_cast<VM*>(vm)->SetOption(name, value);
            },
        } {
    enabled_profiler_.Start<Markers::TOTAL>();
  }

  evmc_result Execute(std::span<const uint8_t> code, const evmc_bytes32* code_hash,                //
                      const evmc_message* message,                                                 //
                      const evmc_host_interface* host_interface, evmc_host_context* host_context,  //
                      evmc_revision revision) {
    std::shared_ptr<const op::ValidJumpTargetsBuffer> valid_jump_targets;
    if (analysis_cache_enabled_ && code_hash && *code_hash != evmc::bytes32{0}) [[likely]] {
      valid_jump_targets = valid_jump_targets_cache_.Get(*code_hash);
      if (!valid_jump_targets) {
        valid_jump_targets = valid_jump_targets_cache_.InsertOrAssign(*code_hash, op::CalculateValidJumpTargets(code));
      }
    } else {
      valid_jump_targets = std::make_shared<std::vector<uint8_t>>(op::CalculateValidJumpTargets(code));
    }

    const auto _ = enabled_profiler_.Scoped<Markers::INTERPRETER>();
    const InterpreterArgs interpreter_args{
        .code = code,
        .valid_jump_targets = *valid_jump_targets,
        .message = message,
        .host_interface = host_interface,
        .host_context = host_context,
        .revision = revision,
        .profiler =
            (profiling_enabled_ ? static_cast<void*>(&enabled_profiler_) : static_cast<void*>(&disabled_profiler_)),
    };
    InterpreterResult interpreter_result;
    if (logging_enabled_ && profiling_enabled_) {
      interpreter_result = Interpret<true, true>(interpreter_args);
    } else if (logging_enabled_ && !profiling_enabled_) {
      interpreter_result = Interpret<true, false>(interpreter_args);
    } else if (!logging_enabled_ && profiling_enabled_) {
      interpreter_result = Interpret<false, true>(interpreter_args);
    } else if (!logging_enabled_ && !profiling_enabled_) {
      interpreter_result = Interpret<false, false>(interpreter_args);
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
    const auto on_off_options = {
        std::pair("logging", &logging_enabled_),
        std::pair("analysis_cache", &analysis_cache_enabled_),
        std::pair("profiling", &profiling_enabled_),
    };

    for (const auto& [option_name, member] : on_off_options) {
      if (name == option_name) {
        if (value == "true") {
          *member = true;
          return EVMC_SET_OPTION_SUCCESS;
        } else if (value == "false") {
          *member = false;
          return EVMC_SET_OPTION_SUCCESS;
        } else {
          return EVMC_SET_OPTION_INVALID_VALUE;
        }
      }
    }
    return EVMC_SET_OPTION_INVALID_NAME;
  }

  void DumpProfiler() {
    enabled_profiler_.End<Markers::TOTAL>();
    disabled_profiler_.Dump();
    enabled_profiler_.Dump();
    enabled_profiler_.Start<Markers::TOTAL>();
  }

  void ResetProfiler() {
    disabled_profiler_.Reset();
    enabled_profiler_.Reset();
  }

 private:
  bool logging_enabled_ = false;
  bool analysis_cache_enabled_ = true;
  bool profiling_enabled_ = false;

  LruCache<evmc::bytes32, op::ValidJumpTargetsBuffer, 1 << 16> valid_jump_targets_cache_;

  Profiler<false> disabled_profiler_;
  Profiler<true> enabled_profiler_;
};

extern "C" {
EVMC_EXPORT evmc_vm* evmc_create_evmzero() noexcept { return new VM; }

EVMC_EXPORT void evmzero_dump_profiler(evmc_vm* vm) noexcept { reinterpret_cast<VM*>(vm)->DumpProfiler(); }

EVMC_EXPORT void evmzero_reset_profiler(evmc_vm* vm) noexcept { reinterpret_cast<VM*>(vm)->ResetProfiler(); }
}

}  // namespace tosca::evmzero

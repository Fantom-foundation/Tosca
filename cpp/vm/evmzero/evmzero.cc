// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#include <cstdint>
#include <iostream>
#include <span>
#include <string_view>

#include <evmc/evmc.h>
#include <evmc/utils.h>

#include "common/lru_cache.h"
#include "vm/evmzero/interpreter.h"
#include "vm/evmzero/logger.h"
#include "vm/evmzero/opcodes.h"
#include "vm/evmzero/profiler.h"
#include "vm/evmzero/sha3_cache.h"

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
    case RunState::kErrorInitCodeSizeExceeded:
      return EVMC_FAILURE;
  }
  return EVMC_FAILURE;
}

RunState ToRunStateCode(evmc_step_status_code state) {
  switch (state) {
    case EVMC_STEP_RUNNING:
      return RunState::kRunning;
    case EVMC_STEP_STOPPED:
      return RunState::kDone;
    case EVMC_STEP_RETURNED:
      return RunState::kReturn;
    case EVMC_STEP_REVERTED:
      return RunState::kRevert;
    case EVMC_STEP_FAILED:
      return RunState::kInvalid;
  }
  return RunState::kErrorOpcode;
}

evmc_step_status_code ToEvmcStepStatusCode(RunState state) {
  switch (state) {
    case RunState::kRunning:
      return EVMC_STEP_RUNNING;
    case RunState::kDone:
      return EVMC_STEP_STOPPED;
    case RunState::kReturn:
      return EVMC_STEP_RETURNED;
    case RunState::kRevert:
      return EVMC_STEP_REVERTED;
    case RunState::kInvalid:
      return EVMC_STEP_FAILED;
    default:
      return EVMC_STEP_FAILED;
  }
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
                          evmc_revision revision, const evmc_message* message, const uint8_t* code,
                          size_t code_size) -> evmc_result {
              return static_cast<VM*>(vm)->Execute({code, code_size}, message, host_interface, host_context, revision);
            },

            .get_capabilities = [](evmc_vm*) -> evmc_capabilities_flagset { return EVMC_CAPABILITY_EVM1; },

            .set_option = [](evmc_vm* vm, char const* name, char const* value) -> evmc_set_option_result {
              return static_cast<VM*>(vm)->SetOption(name, value);
            },
        } {}

  evmc_result Execute(std::span<const uint8_t> code,              //
                      const evmc_message* message,                //
                      const evmc_host_interface* host_interface,  //
                      evmc_host_context* host_context,            //
                      evmc_revision revision) {
    std::shared_ptr<const ContractInfo> contract_info;
    const auto code_hash = message->code_hash;
    if (analysis_cache_enabled_ && code_hash && *code_hash != evmc::bytes32{0}) [[likely]] {
      contract_info = contract_info_cache_.GetOrInsert(*code_hash, [&] { return ComputeContractInfo(code); });
    } else {
      contract_info = ComputeContractInfo(code);
    }

    const auto interpreter_args = InterpreterArgs{
        .padded_code = contract_info->padded_code,
        .valid_jump_targets = contract_info->valid_jump_targets,
        .message = message,
        .host_interface = host_interface,
        .host_context = host_context,
        .revision = revision,
        .sha3_cache = sha3_cache_enabled_ ? &sha3_cache_ : nullptr,
    };

    InterpreterResult interpreter_result;
    if (logging_enabled_) {
      interpreter_result = Interpret(interpreter_args, logger_);
    } else if (profiling_enabled_) {
      interpreter_result = Interpret(interpreter_args, profiler_);
    } else if (profiling_external_enabled_) {
      interpreter_result = Interpret(interpreter_args, profiler_external_);
    } else {
      interpreter_result = Interpret(interpreter_args, no_observer_);
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

  evmc_step_result StepN(std::span<const uint8_t> code,                                               //
                         const evmc_message* message,                                                 //
                         const evmc_host_interface* host_interface, evmc_host_context* host_context,  //
                         evmc_revision revision, evmc_step_status_code status, uint64_t pc, int64_t gas_refunds,
                         std::span<evmc_uint256be> stack, std::span<uint8_t> memory,
                         std::span<const uint8_t> last_call_return_data, int32_t steps) {
    std::shared_ptr<const ContractInfo> contract_info;
    const auto code_hash = message->code_hash;
    if (analysis_cache_enabled_ && code_hash && *code_hash != evmc::bytes32{0}) [[likely]] {
      contract_info = contract_info_cache_.GetOrInsert(*code_hash, [&] { return ComputeContractInfo(code); });
    } else {
      contract_info = ComputeContractInfo(code);
    }

    auto convertedStack = Stack();
    for (const auto& value : stack) {
      const auto v = ToUint256(value);
      convertedStack.Push(v);
    }

    auto stepping_args = SteppingArgs{};
    stepping_args.padded_code = contract_info->padded_code;
    stepping_args.valid_jump_targets = contract_info->valid_jump_targets;
    stepping_args.message = message;
    stepping_args.host_interface = host_interface;
    stepping_args.host_context = host_context;
    stepping_args.revision = revision;
    stepping_args.sha3_cache = sha3_cache_enabled_ ? &sha3_cache_ : nullptr;
    stepping_args.state = ToRunStateCode(status);
    stepping_args.pc = pc;
    stepping_args.gas_refunds = gas_refunds;
    stepping_args.stack = convertedStack;
    stepping_args.memory = Memory(memory);
    stepping_args.steps = status == EVMC_STEP_RUNNING ? steps : 0;
    stepping_args.last_call_return_data.assign(last_call_return_data.begin(), last_call_return_data.end());

    auto stepping_result = InterpretNSteps(stepping_args);

    // Move output data to a dedicated buffer so we can release the interpreter
    // result.
    uint8_t* output_data = nullptr;
    if (!stepping_result.return_data.empty()) {
      output_data = new uint8_t[stepping_result.return_data.size()];
      std::copy(stepping_result.return_data.begin(), stepping_result.return_data.end(), output_data);
    }

    // Copy stack to raw buffer.
    evmc_uint256be* stack_data = nullptr;
    if (stepping_result.stack.GetSize()) {
      auto stack_size = stepping_result.stack.GetSize();
      stack_data = new evmc_uint256be[stack_size];
      for (size_t i = 0; i < stack_size; ++i) {
        stack_data[stack_size - 1 - i] = ToEvmcBytes(stepping_result.stack[i]);
      }
    }

    // Copy memory to raw buffer.
    uint8_t* memory_data = nullptr;
    if (stepping_result.memory.GetSize()) {
      memory_data = new uint8_t[stepping_result.memory.GetSize()];
      stepping_result.memory.WriteTo({memory_data, stepping_result.memory.GetSize()}, 0);
    }

    // Copy last return data to buffer.
    uint8_t* output_last_call_return_data = nullptr;
    size_t output_last_call_return_data_size = stepping_result.last_call_return_data.size();
    if (output_last_call_return_data_size > 0) {
      output_last_call_return_data = new uint8_t[output_last_call_return_data_size];
      memcpy(output_last_call_return_data, stepping_result.last_call_return_data.data(),
             output_last_call_return_data_size);
    }

    return {
        .step_status_code = ToEvmcStepStatusCode(stepping_result.state),
        .status_code = ToEvmcStatusCode(stepping_result.state),
        .revision = revision,
        .pc = stepping_result.pc,
        .gas_left = stepping_result.remaining_gas,
        .gas_refund = stepping_result.refunded_gas,
        .output_data = output_data,
        .output_size = stepping_result.return_data.size(),
        .stack = stack_data,
        .stack_size = stepping_result.stack.GetSize(),
        .memory = memory_data,
        .memory_size = stepping_result.memory.GetSize(),
        .last_call_return_data = output_last_call_return_data,
        .last_call_return_data_size = output_last_call_return_data_size,
        .release =
            [](const evmc_step_result* result) {
              delete[] result->output_data;
              delete[] result->stack;
              delete[] result->memory;
              delete[] result->last_call_return_data;
            },
    };
  }

  evmc_set_option_result SetOption(std::string_view name, std::string_view value) {
    const auto on_off_options = {
        std::pair("logging", &logging_enabled_),
        std::pair("analysis_cache", &analysis_cache_enabled_),
        std::pair("sha3_cache", &sha3_cache_enabled_),
        std::pair("profiling", &profiling_enabled_),
        std::pair("profiling_external", &profiling_external_enabled_),
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

  void DumpProfile() {
    if (profiling_enabled_) {
      profiler_.Collect().Dump();
    } else if (profiling_external_enabled_) {
      profiler_external_.Collect().Dump();
    }
  }

  void ResetProfiler() {
    if (profiling_enabled_) {
      profiler_.Reset();
    } else if (profiling_external_enabled_) {
      profiler_external_.Reset();
    }
  }

 private:
  struct ContractInfo {
    std::vector<uint8_t> padded_code;
    op::ValidJumpTargetsBuffer valid_jump_targets;
  };

  static std::shared_ptr<ContractInfo> ComputeContractInfo(std::span<const uint8_t> code) {
    return std::make_shared<ContractInfo>(ContractInfo{
        .padded_code = internal::PadCode(code),
        .valid_jump_targets = op::CalculateValidJumpTargets(code),
    });
  }

  bool logging_enabled_ = false;
  bool analysis_cache_enabled_ = true;
  bool sha3_cache_enabled_ = true;
  bool profiling_enabled_ = false;
  bool profiling_external_enabled_ = false;

  LruCache<evmc::bytes32, std::shared_ptr<ContractInfo>, 1 << 16> contract_info_cache_;

  Sha3Cache sha3_cache_;

  NoObserver no_observer_;
  Logger logger_;
  Profiler<ProfilerMode::kFull> profiler_;
  Profiler<ProfilerMode::kExternal> profiler_external_;
};

// This class represents the evmzero virtual machine (VM) which consists
// primarily of the evmzero interpreter. This class connects evmzero to the host
// infrastructure via the evmc interface.
class VMSteppable : public evmc_vm_steppable {
 public:
  VMSteppable(evmc_vm* vm) noexcept
      : evmc_vm_steppable{
            .vm = vm,

            .step_n = [](evmc_vm_steppable* vm, const evmc_host_interface* host_interface,
                         evmc_host_context* host_context, evmc_revision revision, const evmc_message* message,
                         const uint8_t* code, size_t code_size, evmc_step_status_code status, uint64_t pc,
                         int64_t gas_refunds, evmc_uint256be* stack, size_t stack_size, uint8_t* memory,
                         size_t memory_size, uint8_t* last_call_return_data, size_t last_call_return_data_size,
                         int32_t steps) -> evmc_step_result {
              return static_cast<VM*>(vm->vm)->StepN({code, code_size}, message, host_interface, host_context, revision,
                                                     status, pc, gas_refunds, {stack, stack_size},
                                                     {memory, memory_size},
                                                     {last_call_return_data, last_call_return_data_size}, steps);
            },

            .destroy =
                [](evmc_vm_steppable* vm) {
                  vm->vm->destroy(vm->vm);
                  delete static_cast<VMSteppable*>(vm);
                },

        } {}
};

extern "C" {
EVMC_EXPORT evmc_vm* evmc_create_evmzero() noexcept { return new VM; }

EVMC_EXPORT evmc_vm_steppable* evmc_create_steppable_evmzero() noexcept {
  return new VMSteppable(evmc_create_evmzero());
}

EVMC_EXPORT void evmzero_dump_profile(evmc_vm* vm) noexcept { reinterpret_cast<VM*>(vm)->DumpProfile(); }

EVMC_EXPORT void evmzero_reset_profiler(evmc_vm* vm) noexcept { reinterpret_cast<VM*>(vm)->ResetProfiler(); }
}

}  // namespace tosca::evmzero

#pragma once

#include <concepts>
#include <utility>

#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero {

template<typename O>
concept Observer = requires(O a) {
    { a.PreRun() } -> std::same_as<void>;
    { a.PreInstruction(std::declval<op::OpCode>()) } -> std::same_as<void>;
    { a.PostInstruction(std::declval<op::OpCode>()) } -> std::same_as<void>;
    { a.PostRun() } -> std::same_as<void>;
};

}

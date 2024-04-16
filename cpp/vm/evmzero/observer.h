//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of 
// this software will be governed by the GNU Lesser General Public Licence v3.
//

#pragma once

#include <concepts>
#include <utility>

#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero {

// Forward declarations to break circular dependencies.

struct InterpreterArgs;

namespace internal {

struct Context;

}  // namespace internal

// The Observer concept defines an interface for observing the interpreter as a whole as well as each instruction being
// interpreted.
// The PreRun/PostRun functions get called at the beginning/end of each interpreter invocation, respectively.
// The PreInstruction/PostInstruction get called before/after interpreting each instruction, respectively.
template <typename O>
concept Observer = requires(O o, const InterpreterArgs &args, op::OpCode opcode, const internal::Context &ctx) {
  { o.uses_context } -> std::same_as<const bool &>;
  { o.PreRun(args) } -> std::same_as<void>;
  { o.PreInstruction(opcode, ctx) } -> std::same_as<void>;
  { o.PostInstruction(opcode, ctx) } -> std::same_as<void>;
  { o.PostRun(args) } -> std::same_as<void>;
};

// This type serves as a no-op implementation of the observer concept that is used when no observer is to be used.
struct NoObserver {
  static constexpr bool uses_context = false;
  inline void PreRun(const InterpreterArgs &) {}
  inline void PreInstruction(op::OpCode, const internal::Context &) {}
  inline void PostInstruction(op::OpCode, const internal::Context &) {}
  inline void PostRun(const InterpreterArgs &) {}
};

}  // namespace tosca::evmzero

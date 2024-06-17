// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#include <iostream>

#include "vm/evmzero/interpreter.h"
#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero {

class Logger {
 public:
  static constexpr bool uses_context = true;

  inline void PreRun(const InterpreterArgs&) {}

  inline void PreInstruction(op::OpCode opcode, const internal::Context& ctx) {
    std::cout << ToString(opcode) << ", " << ctx.gas << ", ";
    if (ctx.stack.GetSize() == 0) {
      std::cout << "-empty-";
    } else {
      std::cout << ctx.stack[0];
    }
    std::cout << "\n" << std::flush;
  }

  inline void PostInstruction(op::OpCode, const internal::Context&) {}

  inline void PostRun(const InterpreterArgs&) {}
};

}  // namespace tosca::evmzero

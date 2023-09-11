#include <iostream>

#include "vm/evmzero/interpreter.h"
#include "vm/evmzero/opcodes.h"

namespace tosca::evmzero {

class Logger {
 public:
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

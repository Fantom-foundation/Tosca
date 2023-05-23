#pragma once

#include <cstdio>

#include "evmzero.h"
#include "evmzero_opcode_implementation.h"

namespace tosca::evmzero {

inline void RunInterpreter(Context& ctx) noexcept {
  // Performance note: adding an extra check here to make sure that the PC is in
  // range of code slows down the interpreter by up to 20%; instead, we always
  // make sure to pre-pad the code array, and only check on jumps.

  while (ctx.state == RunState::kRunning) {
    switch (ctx.code[ctx.pc]) {
      // clang-format off
      case op::STOP: op::stop(ctx); break;

      case op::ADD: op::add(ctx); break;
      case op::MUL: op::mul(ctx); break;
      case op::SUB: op::sub(ctx); break;
      case op::DIV: op::div(ctx); break;
      case op::SDIV: op::sdiv(ctx); break;
      case op::MOD: op::mod(ctx); break;
      case op::SMOD: op::smod(ctx); break;
      case op::ADDMOD: op::addmod(ctx); break;
      case op::MULMOD: op::mulmod(ctx); break;
      case op::EXP: op::exp(ctx); break;
      case op::SIGNEXTEND: op::signextend(ctx); break;
      case op::LT: op::lt(ctx); break;
      case op::GT: op::gt(ctx); break;
      case op::SLT: op::slt(ctx); break;
      case op::SGT: op::sgt(ctx); break;
      case op::EQ: op::eq(ctx); break;
      case op::ISZERO: op::iszero(ctx); break;
      case op::AND: op::bit_and(ctx); break;
      case op::OR: op::bit_or(ctx); break;
      case op::XOR: op::bit_xor(ctx); break;
      case op::NOT: op::bit_not(ctx); break;
      case op::BYTE: op::byte(ctx); break;
      case op::SHL: op::shl(ctx); break;
      case op::SHR: op::shr(ctx); break;
      case op::SAR: op::sar(ctx); break;
      case op::SHA3: op::sha3(ctx); break;
      case op::ADDRESS: op::address(ctx); break;
      case op::BALANCE: op::balance(ctx); break;
      case op::ORIGIN: op::origin(ctx); break;
      case op::CALLER: op::caller(ctx); break;
      case op::CALLVALUE: op::callvalue(ctx); break;
      case op::CALLDATALOAD: op::calldataload(ctx); break;
      case op::CALLDATASIZE: op::calldatasize(ctx); break;
      case op::CALLDATACOPY: op::calldatacopy(ctx); break;
      case op::CODESIZE: op::codesize(ctx); break;
      case op::CODECOPY: op::codecopy(ctx); break;
      case op::GASPRICE: op::gasprice(ctx); break;
      case op::EXTCODESIZE: op::extcodesize(ctx); break;
      case op::EXTCODECOPY: op::extcodecopy(ctx); break;
      case op::RETURNDATASIZE: op::returndatasize(ctx); break;
      case op::RETURNDATACOPY: op::returndatacopy(ctx); break;
      case op::EXTCODEHASH: op::extcodehash(ctx); break;
      case op::BLOCKHASH: op::blockhash(ctx); break;
      case op::COINBASE: op::coinbase(ctx); break;
      case op::TIMESTAMP: op::timestamp(ctx); break;
      case op::NUMBER: op::number(ctx); break;
      case op::DIFFICULTY: op::prevrandao(ctx); break; // intentional
      case op::GASLIMIT: op::gaslimit(ctx); break;
      case op::CHAINID: op::chainid(ctx); break;
      case op::SELFBALANCE: op::selfbalance(ctx); break;
      case op::BASEFEE: op::basefee(ctx); break;

      case op::POP: op::pop(ctx); break;
      case op::MLOAD: op::mload(ctx); break;
      case op::MSTORE: op::mstore(ctx); break;
      case op::MSTORE8: op::mstore8(ctx); break;
      case op::SLOAD: op::sload(ctx); break;
      case op::SSTORE: op::sstore(ctx); break;

      case op::JUMP: op::jump(ctx); break;
      case op::JUMPI: op::jumpi(ctx); break;
      case op::PC: op::pc(ctx); break;
      case op::MSIZE: op::msize(ctx); break;
      case op::GAS: op::gas(ctx); break;
      case op::JUMPDEST: op::jumpdest(ctx); break;

      case op::PUSH1: op::push<1>(ctx); break;
      case op::PUSH2: op::push<2>(ctx); break;
      case op::PUSH3: op::push<3>(ctx); break;
      case op::PUSH4: op::push<4>(ctx); break;
      case op::PUSH5: op::push<5>(ctx); break;
      case op::PUSH6: op::push<6>(ctx); break;
      case op::PUSH7: op::push<7>(ctx); break;
      case op::PUSH8: op::push<8>(ctx); break;
      case op::PUSH9: op::push<9>(ctx); break;
      case op::PUSH10: op::push<10>(ctx); break;
      case op::PUSH11: op::push<11>(ctx); break;
      case op::PUSH12: op::push<12>(ctx); break;
      case op::PUSH13: op::push<13>(ctx); break;
      case op::PUSH14: op::push<14>(ctx); break;
      case op::PUSH15: op::push<15>(ctx); break;
      case op::PUSH16: op::push<16>(ctx); break;
      case op::PUSH17: op::push<17>(ctx); break;
      case op::PUSH18: op::push<18>(ctx); break;
      case op::PUSH19: op::push<19>(ctx); break;
      case op::PUSH20: op::push<20>(ctx); break;
      case op::PUSH21: op::push<21>(ctx); break;
      case op::PUSH22: op::push<22>(ctx); break;
      case op::PUSH23: op::push<23>(ctx); break;
      case op::PUSH24: op::push<24>(ctx); break;
      case op::PUSH25: op::push<25>(ctx); break;
      case op::PUSH26: op::push<26>(ctx); break;
      case op::PUSH27: op::push<27>(ctx); break;
      case op::PUSH28: op::push<28>(ctx); break;
      case op::PUSH29: op::push<29>(ctx); break;
      case op::PUSH30: op::push<30>(ctx); break;
      case op::PUSH31: op::push<31>(ctx); break;
      case op::PUSH32: op::push<32>(ctx); break;

      case op::DUP1: op::dup<1>(ctx); break;
      case op::DUP2: op::dup<2>(ctx); break;
      case op::DUP3: op::dup<3>(ctx); break;
      case op::DUP4: op::dup<4>(ctx); break;
      case op::DUP5: op::dup<5>(ctx); break;
      case op::DUP6: op::dup<6>(ctx); break;
      case op::DUP7: op::dup<7>(ctx); break;
      case op::DUP8: op::dup<8>(ctx); break;
      case op::DUP9: op::dup<9>(ctx); break;
      case op::DUP10: op::dup<10>(ctx); break;
      case op::DUP11: op::dup<11>(ctx); break;
      case op::DUP12: op::dup<12>(ctx); break;
      case op::DUP13: op::dup<13>(ctx); break;
      case op::DUP14: op::dup<14>(ctx); break;
      case op::DUP15: op::dup<15>(ctx); break;
      case op::DUP16: op::dup<16>(ctx); break;

      case op::SWAP1: op::swap<1>(ctx); break;
      case op::SWAP2: op::swap<2>(ctx); break;
      case op::SWAP3: op::swap<3>(ctx); break;
      case op::SWAP4: op::swap<4>(ctx); break;
      case op::SWAP5: op::swap<5>(ctx); break;
      case op::SWAP6: op::swap<6>(ctx); break;
      case op::SWAP7: op::swap<7>(ctx); break;
      case op::SWAP8: op::swap<8>(ctx); break;
      case op::SWAP9: op::swap<9>(ctx); break;
      case op::SWAP10: op::swap<10>(ctx); break;
      case op::SWAP11: op::swap<11>(ctx); break;
      case op::SWAP12: op::swap<12>(ctx); break;
      case op::SWAP13: op::swap<13>(ctx); break;
      case op::SWAP14: op::swap<14>(ctx); break;
      case op::SWAP15: op::swap<15>(ctx); break;
      case op::SWAP16: op::swap<16>(ctx); break;

      case op::LOG0: op::log<0>(ctx); break;
      case op::LOG1: op::log<1>(ctx); break;
      case op::LOG2: op::log<2>(ctx); break;
      case op::LOG3: op::log<3>(ctx); break;
      case op::LOG4: op::log<4>(ctx); break;

      case op::CREATE: op::create_impl<op::CREATE>(ctx); break;
      case op::CREATE2: op::create_impl<op::CREATE2>(ctx); break;

      case op::RETURN: op::return_op(ctx); break;
      case op::REVERT: op::return_op(ctx); break; // TODO

      case op::CALL: op::call_impl<op::CALL>(ctx); break;
      case op::CALLCODE: op::call_impl<op::CALLCODE>(ctx); break;
      case op::DELEGATECALL: op::call_impl<op::DELEGATECALL>(ctx); break;
      case op::STATICCALL: op::call_impl<op::STATICCALL>(ctx); break;

      case op::INVALID: op::invalid(ctx); break;
      case op::SELFDESTRUCT: op::selfdestruct(ctx); break;
        // clang-format on

      default:
        printf("Unknown opcode: 0x%02hhX at %" PRIu64 "\n", ctx.code[ctx.pc], ctx.pc);
        ctx.state = RunState::kErrorOpcode;
    }
  }
}

}  // namespace tosca::evmzero

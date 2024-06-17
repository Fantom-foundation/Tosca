// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

#pragma once

#include <array>
#include <ostream>
#include <string>

#include <ethash/keccak.hpp>
#include <evmc/evmc.hpp>
#include <intx/intx.hpp>

namespace tosca::evmzero {

using uint256_t = intx::uint256;

constexpr uint256_t kUint256Max = ~uint256_t{0};

inline uint8_t* ToBytes(uint256_t& i) { return intx::as_bytes(i); }
inline const uint8_t* ToBytes(const uint256_t& i) { return intx::as_bytes(i); }

inline std::string ToString(const uint256_t& i) { return intx::to_string(i); }
inline std::ostream& operator<<(std::ostream& out, const uint256_t& i) { return out << ToString(i); }

inline evmc_address ToEvmcAddress(const uint256_t& i) { return intx::be::trunc<evmc_address>(i); }
inline uint256_t ToUint256(const evmc_address& address) { return intx::be::load<uint256_t>(address); }

inline evmc::bytes32 ToEvmcBytes(const uint256_t& i) { return intx::be::store<evmc::bytes32>(i); }
inline uint256_t ToUint256(const evmc::bytes32& bytes) { return intx::be::load<uint256_t>(bytes); }

inline ethash::hash256 ToEthash(const uint256_t& i) { return intx::be::store<ethash::hash256>(i); }
inline uint256_t ToUint256(const ethash::hash256& hash) { return intx::be::load<uint256_t>(hash); }

inline std::array<uint8_t, 32> ToByteArrayLe(const uint256_t& i) {
  return reinterpret_cast<const std::array<uint8_t, 32>&>(i);
}
inline uint256_t ToUint256(const std::array<uint8_t, 32>& array_le) {
  return reinterpret_cast<const uint256_t&>(array_le);
}

}  // namespace tosca::evmzero

template <>
struct std::hash<tosca::evmzero::uint256_t> {
  std::size_t operator()(const tosca::evmzero::uint256_t& v) const noexcept {
    const auto h = hash<uint64_t>();
    return h((uint64_t)(v >> 0)) ^ h((uint64_t)(v >> 64)) ^ h((uint64_t)(v >> 128)) ^ h((uint64_t)(v >> 192));
  }
};

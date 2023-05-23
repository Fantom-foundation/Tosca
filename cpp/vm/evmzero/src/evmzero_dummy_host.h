#pragma once

#include <algorithm>
#include <set>
#include <span>
#include <unordered_map>

#include <evmc/evmc.hpp>

#include "evmzero_uint256.h"

namespace tosca::evmzero {

class DummyHost {
 public:
  struct LogEntry {
    std::vector<uint8_t> data;
    std::vector<evmc::bytes32> topics;

    friend bool operator==(const LogEntry&, const LogEntry&) = default;
  };

  struct AccountData {
    bool dead = false;
    evmc::uint256be balance;
    std::vector<uint8_t> code;
    std::unordered_map<evmc::bytes32, evmc::bytes32> storage;
    std::vector<LogEntry> logs;

    friend bool operator==(const AccountData&, const AccountData&) = default;
  };

  ////////////////////////////////////////////////////////////

  DummyHost() = default;
  DummyHost(const std::unordered_map<evmc::address, AccountData>& accounts) : accounts_(accounts) {}

  ////////////////////////////////////////////////////////////
  // EVMC Interface
  static evmc_host_interface GetEvmcHostInterface() {
    return {
        .account_exists = [](evmc_host_context* host, const evmc_address* address) -> bool {
          return reinterpret_cast<DummyHost*>(host)->AccountExists(*address);
        },

        .get_storage = [](evmc_host_context* host, const evmc_address* address, const evmc_bytes32* key)
            -> evmc_bytes32 { return reinterpret_cast<DummyHost*>(host)->GetStorage(*address, *key); },

        .set_storage =
            [](evmc_host_context* host, const evmc_address* address, const evmc_bytes32* key,
               const evmc_bytes32* value) {
              return reinterpret_cast<DummyHost*>(host)->SetStorage(*address, *key, *value);
            },

        .get_balance = [](evmc_host_context* host, const evmc_address* address) -> evmc_uint256be {
          return reinterpret_cast<DummyHost*>(host)->GetBalance(*address);
        },

        .get_code_size = [](evmc_host_context* host, const evmc_address* address) -> size_t {
          return reinterpret_cast<DummyHost*>(host)->GetCodeSize(*address);
        },

        .get_code_hash = [](evmc_host_context* host, const evmc_address* address) -> evmc_bytes32 {
          return reinterpret_cast<DummyHost*>(host)->GetCodeHash(*address);
        },

        .copy_code = [](evmc_host_context* host, const evmc_address* address,  //
                        size_t offset, uint8_t* dst, size_t dst_size) -> size_t {
          return reinterpret_cast<DummyHost*>(host)->CopyCode(*address, offset, {dst, dst_size});
        },

        .selfdestruct = [](evmc_host_context* host, const evmc_address* address, const evmc_address* beneficiary)
            -> bool { return reinterpret_cast<DummyHost*>(host)->SelfDestruct(*address, *beneficiary); },

        .call = [](evmc_host_context* host, const evmc_message* message) -> evmc_result {
          return reinterpret_cast<DummyHost*>(host)->Call(*message);
        },

        .get_tx_context = [](evmc_host_context* host) -> evmc_tx_context {
          return reinterpret_cast<DummyHost*>(host)->GetTxContext();
        },

        .get_block_hash = [](evmc_host_context* host, int64_t number) -> evmc_bytes32 {
          return reinterpret_cast<DummyHost*>(host)->GetBlockHash(number);
        },

        .emit_log =
            [](evmc_host_context* host, const evmc_address* address, const uint8_t* data, size_t data_size,
               const evmc_bytes32 topics[], size_t topics_count) {
              return reinterpret_cast<DummyHost*>(host)->EmitLog(*address, {data, data_size}, {topics, topics_count});
            },

        .access_account = [](evmc_host_context* host, const evmc_address* address) -> evmc_access_status {
          return reinterpret_cast<DummyHost*>(host)->GetAccountAccess(*address);
        },

        .access_storage = [](evmc_host_context* host, const evmc_address* address, const evmc_bytes32* key)
            -> evmc_access_status { return reinterpret_cast<DummyHost*>(host)->GetStorageAccess(*address, *key); },
    };
  }

  operator evmc_host_context*() { return reinterpret_cast<evmc_host_context*>(this); }

  ////////////////////////////////////////////////////////////
  // Host Interface

  bool AccountExists(const evmc::address& address) const { return accounts_.contains(address); }

  evmc::bytes32 GetStorage(const evmc::address& address, const evmc::bytes32& key) {
    if (const auto* account = GetAccountData(address)) {
      if (auto it = account->storage.find(key); it != account->storage.end()) {
        return it->second;
      }
    }
    return evmc::bytes32{0};
  }

  evmc_storage_status SetStorage(const evmc::address& address, const evmc::bytes32& key, const evmc::bytes32& value) {
    if (auto* account = GetAccountData(address)) {
      account->storage[key] = value;
    }

    // TODO: Return correct storage status.
    return EVMC_STORAGE_ASSIGNED;
  }

  evmc_access_status GetStorageAccess(const evmc::address&, const evmc::bytes32&) const {
    // TODO
    return EVMC_ACCESS_COLD;
  }

  evmc::uint256be GetBalance(const evmc::address& address) const {
    if (const auto* account = GetAccountData(address)) {
      return account->balance;
    } else {
      return evmc::uint256be{0};
    }
  }

  size_t GetCodeSize(const evmc::address& address) const {
    if (const auto* account = GetAccountData(address)) {
      return account->code.size();
    } else {
      return 0;
    }
  }

  evmc::bytes32 GetCodeHash(const evmc::address&) const {
    // TODO
    return evmc::bytes32{0};
  }

  size_t CopyCode(const evmc::address& address, size_t offset, std::span<uint8_t> dst) const {
    if (const auto* account = GetAccountData(address)) {
      if (offset >= account->code.size()) {
        return 0;
      }

      size_t bytes_to_copy = std::min(account->code.size() - offset, dst.size_bytes());
      std::copy_n(account->code.data() + offset, bytes_to_copy, dst.begin());
      return bytes_to_copy;
    } else {
      return 0;
    }
  }

  bool SelfDestruct(const evmc::address& address, const evmc::address& beneficiary) {
    bool was_alive = false;

    if (auto* account = GetAccountData(address)) {
      if (auto* beneficiary_account = GetAccountData(beneficiary)) {
        uint256_t beneficiary_balance = ToUint256(beneficiary_account->balance);
        uint256_t account_balance = ToUint256(account->balance);
        beneficiary_account->balance = ToEvmcBytes(beneficiary_balance + account_balance);
      }

      account->balance = evmc::uint256be{0};

      if (!account->dead) {
        was_alive = true;
      }
      account->dead = true;
    }

    return was_alive;
  }

  evmc_result Call(const evmc_message&) const {
    // TODO
    return {.status_code = EVMC_FAILURE};
  }

  const evmc_tx_context& GetTxContext() const { return tx_context_; }
  void SetTxContext(const evmc_tx_context& tx_context) { tx_context_ = tx_context; }

  evmc::bytes32 GetBlockHash(int64_t) const {
    // TODO
    return evmc::bytes32{0};
  }

  void EmitLog(const evmc::address& address, std::span<const uint8_t> data, std::span<const evmc_bytes32> topics) {
    if (auto* account = GetAccountData(address)) {
      account->logs.push_back({
          .data = {data.begin(), data.end()},
          .topics = {topics.begin(), topics.end()},
      });
    }
  }

  ////////////////////////////////////////////////////////////
  // Account Management

  // Returns a pointer to AccountData, or nullptr if the account does not exist.
  // Note that this pointer is *not* stable and may be invalidated when the
  // internal accounts data structure is updated!
  AccountData* GetAccountData(const evmc::address& address) {
    if (auto it = accounts_.find(address); it != accounts_.end()) {
      return &it->second;
    } else {
      return nullptr;
    }
  }
  const AccountData* GetAccountData(const evmc::address& address) const {
    return const_cast<DummyHost&>(*this).GetAccountData(address);
  }

  const std::unordered_map<evmc::address, AccountData>& GetAccounts() const { return accounts_; }

  void SetAccountData(const evmc::address& address, const AccountData& account) { accounts_[address] = account; }

  evmc_access_status GetAccountAccess(const evmc::address&) const {
    // TODO
    return EVMC_ACCESS_COLD;
  }

 private:
  std::unordered_map<evmc::address, AccountData> accounts_;
  evmc_tx_context tx_context_;
};

}  // namespace tosca::evmzero

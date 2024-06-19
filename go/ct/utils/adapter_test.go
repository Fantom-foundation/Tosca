// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package utils

import (
	"reflect"
	"testing"

	cc "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"golang.org/x/crypto/sha3"
)

func TestAdapter_ParameterConversion(t *testing.T) {

	properties := map[string]struct {
		set func(*st.State)
		get func(vm.Parameters) (want, got any)
	}{
		"revision": {
			func(s *st.State) { s.Revision = cc.R07_Istanbul },
			func(p vm.Parameters) (any, any) { return vm.R07_Istanbul, p.Revision },
		},
		"gas": {
			func(s *st.State) { s.Gas = 12 },
			func(p vm.Parameters) (any, any) { return vm.Gas(12), p.Gas },
		},
		"code": {
			func(s *st.State) { s.Code = st.NewCode(vm.Code{1, 2, 3}) },
			func(p vm.Parameters) (any, any) { return vm.Code{1, 2, 3}, p.Code },
		},
		"input": {
			func(s *st.State) { s.CallData = cc.NewBytes(vm.Data{1, 2, 3}) },
			func(p vm.Parameters) (any, any) { return vm.Data{1, 2, 3}, p.Input },
		},
		"read-only-true": {
			func(s *st.State) { s.ReadOnly = true },
			func(p vm.Parameters) (any, any) { return true, p.Static },
		},
		"read-only-false": {
			func(s *st.State) { s.ReadOnly = false },
			func(p vm.Parameters) (any, any) { return false, p.Static },
		},
		"recipient": {
			func(s *st.State) { s.CallContext.AccountAddress = vm.Address{1, 2, 3} },
			func(p vm.Parameters) (any, any) { return vm.Address{1, 2, 3}, p.Recipient },
		},
		"sender": {
			func(s *st.State) { s.CallContext.CallerAddress = vm.Address{1, 2, 3} },
			func(p vm.Parameters) (any, any) { return vm.Address{1, 2, 3}, p.Sender },
		},
		"origin": {
			func(s *st.State) {
				s.TransactionContext = st.NewTransactionContext()
				s.TransactionContext.OriginAddress = vm.Address{1, 2, 3}
			},
			func(p vm.Parameters) (any, any) { return vm.Address{1, 2, 3}, p.Origin },
		},
		"blob-hashes": {
			func(s *st.State) {
				s.TransactionContext = st.NewTransactionContext()
				s.TransactionContext.BlobHashes = []vm.Hash{{1, 2, 3}}
			},
			func(p vm.Parameters) (any, any) { return []vm.Hash{{1, 2, 3}}, p.BlobHashes },
		},
		"value": {
			func(s *st.State) { s.CallContext.Value = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) { return vm.Value(cc.NewU256(123).Bytes32be()), p.Value },
		},
		"gas-price": {
			func(s *st.State) { s.BlockContext.GasPrice = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Value(cc.NewU256(123).Bytes32be()), p.GasPrice
			},
		},
		"coinbase": {
			func(s *st.State) { s.BlockContext.CoinBase = vm.Address{1, 2, 3} },
			func(p vm.Parameters) (any, any) {
				return vm.Address{1, 2, 3}, p.Coinbase
			},
		},
		"block-number": {
			func(s *st.State) { s.BlockContext.BlockNumber = 123 },
			func(p vm.Parameters) (any, any) {
				return int64(123), p.BlockNumber
			},
		},
		"timestamp": {
			func(s *st.State) { s.BlockContext.TimeStamp = 123 },
			func(p vm.Parameters) (any, any) {
				return int64(123), p.Timestamp
			},
		},
		"gas-limit": {
			func(s *st.State) { s.BlockContext.GasLimit = 123 },
			func(p vm.Parameters) (any, any) {
				return vm.Gas(123), p.GasLimit
			},
		},
		"prev-randao": {
			func(s *st.State) { s.BlockContext.PrevRandao = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Hash(cc.NewU256(123).Bytes32be()), p.PrevRandao
			},
		},
		"chain-id": {
			func(s *st.State) { s.BlockContext.ChainID = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Word(cc.NewU256(123).Bytes32be()), p.ChainID
			},
		},
		"base-fee": {
			func(s *st.State) { s.BlockContext.BaseFee = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Value(cc.NewU256(123).Bytes32be()), p.BaseFee
			},
		},
		"blob-base-fee": {
			func(s *st.State) { s.BlockContext.BlobBaseFee = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Value(cc.NewU256(123).Bytes32be()), p.BlobBaseFee
			},
		},
		"storage-current-unspecified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetCurrent(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Word{}, ctxt.GetStorage(vm.Address{}, vm.Key{})
			},
		},
		"storage-current-specified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetCurrent(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				key1 := vm.Key(cc.NewU256(1).Bytes32be())
				val2 := vm.Word(cc.NewU256(2).Bytes32be())
				return val2, ctxt.GetStorage(vm.Address{}, key1)
			},
		},
		"storage-original-unspecified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetCurrent(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Word{}, ctxt.GetCommittedStorage(vm.Address{}, vm.Key{})
			},
		},
		"storage-original-specified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetOriginal(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				key1 := vm.Key(cc.NewU256(1).Bytes32be())
				val2 := vm.Word(cc.NewU256(2).Bytes32be())
				return val2, ctxt.GetCommittedStorage(vm.Address{}, key1)
			},
		},
		"cold-slot": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				_, res := ctxt.IsSlotInAccessList(vm.Address{}, vm.Key{})
				return false, res
			},
		},
		"warm-slot": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetWarm(cc.NewU256(), true).
					Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				_, res := ctxt.IsSlotInAccessList(vm.Address{}, vm.Key{})
				return true, res
			},
		},
		"balance-unspecified": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetBalance(vm.Address{1}, cc.NewU256(2))
				s.Accounts = ab.Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Value{}, ctxt.GetBalance(vm.Address{})
			},
		},
		"balance-specified": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetBalance(vm.Address{1}, cc.NewU256(2))
				s.Accounts = ab.Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Value(cc.NewU256(2).Bytes32be()), ctxt.GetBalance(vm.Address{1})
			},
		},
		"getCode": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetCode(vm.Address{1}, cc.NewBytes([]byte{byte(cc.ADD), byte(cc.SUB)}))
				s.Accounts = ab.Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Code{byte(cc.ADD), byte(cc.SUB)}, ctxt.GetCode(vm.Address{1})
			},
		},
		"getCodeHash-emptyHash": {
			func(s *st.State) {
				s.Accounts = st.NewAccountsBuilder().Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context

				var hash [32]byte
				hasher := sha3.NewLegacyKeccak256()
				hasher.Write([]byte{})
				hasher.Sum(hash[:])

				return vm.Hash(hash), ctxt.GetCodeHash(vm.Address{})
			},
		},
		"getCodeHash": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetCode(vm.Address{1}, cc.NewBytes([]byte{byte(cc.ADD), byte(cc.SUB)}))
				s.Accounts = ab.Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context

				var hash [32]byte
				hasher := sha3.NewLegacyKeccak256()
				hasher.Write([]byte{byte(cc.ADD), byte(cc.SUB)})
				hasher.Sum(hash[:])

				return vm.Hash(hash), ctxt.GetCodeHash(vm.Address{1})
			},
		},
		"getCodeSize-empty": {
			func(s *st.State) {
				s.Accounts = st.NewAccountsBuilder().Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return 0, ctxt.GetCodeSize(vm.Address{})
			},
		},
		"getCodeSize": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetCode(vm.Address{1}, cc.NewBytes([]byte{byte(cc.ADD), byte(cc.SUB)}))
				s.Accounts = ab.Build()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return 2, ctxt.GetCodeSize(vm.Address{1})
			},
		},
		"cold-account": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.ColdAccess, ctxt.AccessAccount(vm.Address{})
			},
		},
		"warm-account": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
				s.Accounts.MarkWarm(vm.Address{})
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.WarmAccess, ctxt.AccessAccount(vm.Address{})
			},
		},
		"cold-account-legacy": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return false, ctxt.IsAddressInAccessList(vm.Address{})
			},
		},
		"warm-account-legacy": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
				s.Accounts.MarkWarm(vm.Address{})
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return true, ctxt.IsAddressInAccessList(vm.Address{})
			},
		},
	}

	for name, property := range properties {
		t.Run(name, func(t *testing.T) {
			state := st.State{}
			property.set(&state)
			params := ToVmParameters(&state)
			want, got := property.get(params)
			if !reflect.DeepEqual(want, got) {
				t.Errorf("failed to verify property, wanted %v, got %v of type %v and %v", want, got, reflect.TypeOf(want), reflect.TypeOf(got))
			}
		})
	}
}

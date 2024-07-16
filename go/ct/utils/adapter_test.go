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
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
	"golang.org/x/crypto/sha3"
)

func TestAdapter_ParameterConversion(t *testing.T) {

	properties := map[string]struct {
		set func(*st.State)
		get func(tosca.Parameters) (want, got any)
	}{
		"revision": {
			func(s *st.State) { s.Revision = tosca.R07_Istanbul },
			func(p tosca.Parameters) (any, any) { return tosca.R07_Istanbul, p.Revision },
		},
		"gas": {
			func(s *st.State) { s.Gas = 12 },
			func(p tosca.Parameters) (any, any) { return tosca.Gas(12), p.Gas },
		},
		"code": {
			func(s *st.State) { s.Code = st.NewCode(tosca.Code{1, 2, 3}) },
			func(p tosca.Parameters) (any, any) { return tosca.Code{1, 2, 3}, p.Code },
		},
		"input": {
			func(s *st.State) { s.CallData = cc.NewBytes(tosca.Data{1, 2, 3}) },
			func(p tosca.Parameters) (any, any) { return tosca.Data{1, 2, 3}, p.Input },
		},
		"read-only-true": {
			func(s *st.State) { s.ReadOnly = true },
			func(p tosca.Parameters) (any, any) { return true, p.Static },
		},
		"read-only-false": {
			func(s *st.State) { s.ReadOnly = false },
			func(p tosca.Parameters) (any, any) { return false, p.Static },
		},
		"recipient": {
			func(s *st.State) { s.CallContext.AccountAddress = tosca.Address{1, 2, 3} },
			func(p tosca.Parameters) (any, any) { return tosca.Address{1, 2, 3}, p.Recipient },
		},
		"sender": {
			func(s *st.State) { s.CallContext.CallerAddress = tosca.Address{1, 2, 3} },
			func(p tosca.Parameters) (any, any) { return tosca.Address{1, 2, 3}, p.Sender },
		},
		"origin": {
			func(s *st.State) {
				s.TransactionContext = st.NewTransactionContext()
				s.TransactionContext.OriginAddress = tosca.Address{1, 2, 3}
			},
			func(p tosca.Parameters) (any, any) { return tosca.Address{1, 2, 3}, p.Origin },
		},
		"blob-hashes": {
			func(s *st.State) {
				s.TransactionContext = st.NewTransactionContext()
				s.TransactionContext.BlobHashes = []tosca.Hash{{1, 2, 3}}
			},
			func(p tosca.Parameters) (any, any) { return []tosca.Hash{{1, 2, 3}}, p.BlobHashes },
		},
		"value": {
			func(s *st.State) { s.CallContext.Value = cc.NewU256(123) },
			func(p tosca.Parameters) (any, any) { return tosca.Value(cc.NewU256(123).Bytes32be()), p.Value },
		},
		"gas-price": {
			func(s *st.State) { s.BlockContext.GasPrice = cc.NewU256(123) },
			func(p tosca.Parameters) (any, any) {
				return tosca.Value(cc.NewU256(123).Bytes32be()), p.GasPrice
			},
		},
		"coinbase": {
			func(s *st.State) { s.BlockContext.CoinBase = tosca.Address{1, 2, 3} },
			func(p tosca.Parameters) (any, any) {
				return tosca.Address{1, 2, 3}, p.Coinbase
			},
		},
		"block-number": {
			func(s *st.State) { s.BlockContext.BlockNumber = 123 },
			func(p tosca.Parameters) (any, any) {
				return int64(123), p.BlockNumber
			},
		},
		"timestamp": {
			func(s *st.State) { s.BlockContext.TimeStamp = 123 },
			func(p tosca.Parameters) (any, any) {
				return int64(123), p.Timestamp
			},
		},
		"gas-limit": {
			func(s *st.State) { s.BlockContext.GasLimit = 123 },
			func(p tosca.Parameters) (any, any) {
				return tosca.Gas(123), p.GasLimit
			},
		},
		"prev-randao": {
			func(s *st.State) { s.BlockContext.PrevRandao = cc.NewU256(123) },
			func(p tosca.Parameters) (any, any) {
				return tosca.Hash(cc.NewU256(123).Bytes32be()), p.PrevRandao
			},
		},
		"chain-id": {
			func(s *st.State) { s.BlockContext.ChainID = cc.NewU256(123) },
			func(p tosca.Parameters) (any, any) {
				return tosca.Word(cc.NewU256(123).Bytes32be()), p.ChainID
			},
		},
		"base-fee": {
			func(s *st.State) { s.BlockContext.BaseFee = cc.NewU256(123) },
			func(p tosca.Parameters) (any, any) {
				return tosca.Value(cc.NewU256(123).Bytes32be()), p.BaseFee
			},
		},
		"blob-base-fee": {
			func(s *st.State) { s.BlockContext.BlobBaseFee = cc.NewU256(123) },
			func(p tosca.Parameters) (any, any) {
				return tosca.Value(cc.NewU256(123).Bytes32be()), p.BlobBaseFee
			},
		},
		"storage-current-unspecified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetCurrent(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return tosca.Word{}, ctxt.GetStorage(tosca.Address{}, tosca.Key{})
			},
		},
		"storage-current-specified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetCurrent(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				key1 := tosca.Key(cc.NewU256(1).Bytes32be())
				val2 := tosca.Word(cc.NewU256(2).Bytes32be())
				return val2, ctxt.GetStorage(tosca.Address{}, key1)
			},
		},
		"storage-original-unspecified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetCurrent(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				//lint:ignore SA1019 deprecated functions to be migrated in #616
				return tosca.Word{}, ctxt.GetCommittedStorage(tosca.Address{}, tosca.Key{})
			},
		},
		"storage-original-specified": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetOriginal(cc.NewU256(1), cc.NewU256(2)).
					Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				key1 := tosca.Key(cc.NewU256(1).Bytes32be())
				val2 := tosca.Word(cc.NewU256(2).Bytes32be())
				//lint:ignore SA1019 deprecated functions to be migrated in #616
				return val2, ctxt.GetCommittedStorage(tosca.Address{}, key1)
			},
		},
		"cold-slot": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				//lint:ignore SA1019 deprecated functions to be migrated in #616
				_, res := ctxt.IsSlotInAccessList(tosca.Address{}, tosca.Key{})
				return false, res
			},
		},
		"warm-slot": {
			func(s *st.State) {
				s.Storage = st.NewStorageBuilder().
					SetWarm(cc.NewU256(), true).
					Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				//lint:ignore SA1019 deprecated functions to be migrated in #616
				_, res := ctxt.IsSlotInAccessList(tosca.Address{}, tosca.Key{})
				return true, res
			},
		},
		"balance-unspecified": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetBalance(tosca.Address{1}, cc.NewU256(2))
				s.Accounts = ab.Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return tosca.Value{}, ctxt.GetBalance(tosca.Address{})
			},
		},
		"balance-specified": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetBalance(tosca.Address{1}, cc.NewU256(2))
				s.Accounts = ab.Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return tosca.Value(cc.NewU256(2).Bytes32be()), ctxt.GetBalance(tosca.Address{1})
			},
		},
		"getCode": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetCode(tosca.Address{1}, cc.NewBytes([]byte{byte(vm.ADD), byte(vm.SUB)}))
				s.Accounts = ab.Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return tosca.Code{byte(vm.ADD), byte(vm.SUB)}, ctxt.GetCode(tosca.Address{1})
			},
		},
		"getCodeHash-emptyHash": {
			func(s *st.State) {
				s.Accounts = st.NewAccountsBuilder().Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context

				var hash [32]byte
				hasher := sha3.NewLegacyKeccak256()
				hasher.Write([]byte{})
				hasher.Sum(hash[:])

				return tosca.Hash(hash), ctxt.GetCodeHash(tosca.Address{})
			},
		},
		"getCodeHash": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetCode(tosca.Address{1}, cc.NewBytes([]byte{byte(vm.ADD), byte(vm.SUB)}))
				s.Accounts = ab.Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context

				var hash [32]byte
				hasher := sha3.NewLegacyKeccak256()
				hasher.Write([]byte{byte(vm.ADD), byte(vm.SUB)})
				hasher.Sum(hash[:])

				return tosca.Hash(hash), ctxt.GetCodeHash(tosca.Address{1})
			},
		},
		"getCodeSize-empty": {
			func(s *st.State) {
				s.Accounts = st.NewAccountsBuilder().Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return 0, ctxt.GetCodeSize(tosca.Address{})
			},
		},
		"getCodeSize": {
			func(s *st.State) {
				ab := st.NewAccountsBuilder()
				ab.SetCode(tosca.Address{1}, cc.NewBytes([]byte{byte(vm.ADD), byte(vm.SUB)}))
				s.Accounts = ab.Build()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return 2, ctxt.GetCodeSize(tosca.Address{1})
			},
		},
		"cold-account": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return tosca.ColdAccess, ctxt.AccessAccount(tosca.Address{})
			},
		},
		"warm-account": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
				s.Accounts.MarkWarm(tosca.Address{})
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				return tosca.WarmAccess, ctxt.AccessAccount(tosca.Address{})
			},
		},
		"cold-account-legacy": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				//lint:ignore SA1019 deprecated functions to be migrated in #616
				return false, ctxt.IsAddressInAccessList(tosca.Address{})
			},
		},
		"warm-account-legacy": {
			func(s *st.State) {
				s.Accounts = st.NewAccounts()
				s.Accounts.MarkWarm(tosca.Address{})
			},
			func(p tosca.Parameters) (any, any) {
				ctxt := p.Context
				//lint:ignore SA1019 deprecated functions to be migrated in #616
				return true, ctxt.IsAddressInAccessList(tosca.Address{})
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

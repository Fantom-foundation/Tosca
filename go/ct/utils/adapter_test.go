package utils

import (
	"reflect"
	"testing"

	cc "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
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
			func(s *st.State) { s.Code = st.NewCode([]byte{1, 2, 3}) },
			func(p vm.Parameters) (any, any) { return []byte{1, 2, 3}, p.Code },
		},
		"input": {
			func(s *st.State) { s.CallData = []byte{1, 2, 3} },
			func(p vm.Parameters) (any, any) { return []byte{1, 2, 3}, p.Input },
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
			func(s *st.State) { s.CallContext.AccountAddress = cc.Address{1, 2, 3} },
			func(p vm.Parameters) (any, any) { return vm.Address{1, 2, 3}, p.Recipient },
		},
		"sender": {
			func(s *st.State) { s.CallContext.CallerAddress = cc.Address{1, 2, 3} },
			func(p vm.Parameters) (any, any) { return vm.Address{1, 2, 3}, p.Sender },
		},
		"origin": {
			func(s *st.State) { s.CallContext.OriginAddress = cc.Address{1, 2, 3} },
			func(p vm.Parameters) (any, any) { return vm.Address{1, 2, 3}, p.Context.GetTransactionContext().Origin },
		},
		"value": {
			func(s *st.State) { s.CallContext.Value = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) { return vm.Value(cc.NewU256(123).Bytes32be()), p.Value },
		},
		"gas-price": {
			func(s *st.State) { s.BlockContext.GasPrice = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Value(cc.NewU256(123).Bytes32be()), p.Context.GetTransactionContext().GasPrice
			},
		},
		"coinbase": {
			func(s *st.State) { s.BlockContext.CoinBase = cc.Address{1, 2, 3} },
			func(p vm.Parameters) (any, any) {
				return vm.Address{1, 2, 3}, p.Context.GetTransactionContext().Coinbase
			},
		},
		"block-number": {
			func(s *st.State) { s.BlockContext.BlockNumber = 123 },
			func(p vm.Parameters) (any, any) {
				return int64(123), p.Context.GetTransactionContext().BlockNumber
			},
		},
		"timestamp": {
			func(s *st.State) { s.BlockContext.TimeStamp = 123 },
			func(p vm.Parameters) (any, any) {
				return int64(123), p.Context.GetTransactionContext().Timestamp
			},
		},
		"gas-limit": {
			func(s *st.State) { s.BlockContext.GasLimit = 123 },
			func(p vm.Parameters) (any, any) {
				return vm.Gas(123), p.Context.GetTransactionContext().GasLimit
			},
		},
		"prev-randao": {
			func(s *st.State) { s.BlockContext.Difficulty = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Hash(cc.NewU256(123).Bytes32be()), p.Context.GetTransactionContext().PrevRandao
			},
		},
		"chain-id": {
			func(s *st.State) { s.BlockContext.ChainID = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Word(cc.NewU256(123).Bytes32be()), p.Context.GetTransactionContext().ChainID
			},
		},
		"base-fee": {
			func(s *st.State) { s.BlockContext.BaseFee = cc.NewU256(123) },
			func(p vm.Parameters) (any, any) {
				return vm.Value(cc.NewU256(123).Bytes32be()), p.Context.GetTransactionContext().BaseFee
			},
		},
		"storage-current-unspecified": {
			func(s *st.State) {
				s.Storage = &st.Storage{}
				s.Storage.Current = map[cc.U256]cc.U256{
					cc.NewU256(1): cc.NewU256(2),
				}
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Word{}, ctxt.GetStorage(vm.Address{}, vm.Key{})
			},
		},
		"storage-current-specified": {
			func(s *st.State) {
				s.Storage = &st.Storage{}
				s.Storage.Current = map[cc.U256]cc.U256{
					cc.NewU256(1): cc.NewU256(2),
				}
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
				s.Storage = &st.Storage{}
				s.Storage.Original = map[cc.U256]cc.U256{
					cc.NewU256(1): cc.NewU256(2),
				}
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Word{}, ctxt.GetCommittedStorage(vm.Address{}, vm.Key{})
			},
		},
		"storage-original-specified": {
			func(s *st.State) {
				s.Storage = &st.Storage{}
				s.Storage.Original = map[cc.U256]cc.U256{
					cc.NewU256(1): cc.NewU256(2),
				}
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
				s.Storage = st.NewStorage()
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				_, res := ctxt.IsSlotInAccessList(vm.Address{}, vm.Key{})
				return false, res
			},
		},
		"warm-slot": {
			func(s *st.State) {
				s.Storage = st.NewStorage()
				s.Storage.MarkWarm(cc.NewU256())
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				_, res := ctxt.IsSlotInAccessList(vm.Address{}, vm.Key{})
				return true, res
			},
		},
		"balance-unspecified": {
			func(s *st.State) {
				s.Accounts = &st.Accounts{}
				s.Accounts.Balance = map[cc.Address]cc.U256{
					{1}: cc.NewU256(2),
				}
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Value{}, ctxt.GetBalance(vm.Address{})
			},
		},
		"balance-specified": {
			func(s *st.State) {
				s.Accounts = &st.Accounts{}
				s.Accounts.Balance = map[cc.Address]cc.U256{
					{1}: cc.NewU256(2),
				}
			},
			func(p vm.Parameters) (any, any) {
				ctxt := p.Context
				return vm.Value(cc.NewU256(2).Bytes32be()), ctxt.GetBalance(vm.Address{1})
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
				s.Accounts.MarkWarm(cc.Address{})
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
				s.Accounts.MarkWarm(cc.Address{})
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

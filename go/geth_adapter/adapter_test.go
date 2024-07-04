package geth_adapter

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -source adapter_test.go -destination adapter_test_mocks.go -package geth_adapter

type StateDb interface {
	geth.StateDB
}

func TestRunContextAdapter_SetBalanceHasCorrectEffect(t *testing.T) {
	tests := []struct {
		before tosca.Value
		after  tosca.Value
		add    tosca.Value
		sub    tosca.Value
	}{
		{},
		{
			before: tosca.ValueFromUint64(10),
			after:  tosca.ValueFromUint64(10),
		},
		{
			before: tosca.ValueFromUint64(0),
			after:  tosca.ValueFromUint64(1),
			add:    tosca.ValueFromUint64(1),
		},
		{
			before: tosca.ValueFromUint64(1),
			after:  tosca.ValueFromUint64(0),
			sub:    tosca.ValueFromUint64(1),
		},
		{
			before: tosca.ValueFromUint64(123),
			after:  tosca.ValueFromUint64(321),
			add:    tosca.ValueFromUint64(321 - 123),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v_to_%v", test.before, test.after), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			stateDb := NewMockStateDb(ctrl)
			stateDb.EXPECT().GetBalance(gomock.Any()).Return(test.before.ToUint256())
			if test.add != (tosca.Value{}) {
				diff := test.add.ToUint256()
				stateDb.EXPECT().AddBalance(gomock.Any(), diff, gomock.Any())
			}
			if test.sub != (tosca.Value{}) {
				diff := test.sub.ToUint256()
				stateDb.EXPECT().SubBalance(gomock.Any(), diff, gomock.Any())
			}

			adapter := &runContextAdapter{evm: &geth.EVM{StateDB: stateDb}}
			adapter.SetBalance(tosca.Address{}, test.after)
		})
	}
}

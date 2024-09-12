// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package geth_adapter

import (
	"fmt"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	geth "github.com/ethereum/go-ethereum/core/vm"
	"go.uber.org/mock/gomock"

	_ "github.com/Fantom-foundation/Tosca/go/interpreter/geth"
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
			before: tosca.NewValue(10),
			after:  tosca.NewValue(10),
		},
		{
			before: tosca.NewValue(0),
			after:  tosca.NewValue(1),
			add:    tosca.NewValue(1),
		},
		{
			before: tosca.NewValue(1),
			after:  tosca.NewValue(0),
			sub:    tosca.NewValue(1),
		},
		{
			before: tosca.NewValue(123),
			after:  tosca.NewValue(321),
			add:    tosca.NewValue(321 - 123),
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

func TestRunContextAdapter_ReferenceGethInterpreterIsNotExported(t *testing.T) {
	if res, err := tosca.NewInterpreter("geth", nil); res == nil || err != nil {
		t.Fatal("geth reference interpreter not available in Tosca")
	}
	evm := &geth.EVM{}
	interpreter := geth.NewInterpreter("geth", evm, geth.Config{})
	if interpreter == nil {
		t.Fatal("no interpreter registered for 'geth'")
	}
	if _, ok := interpreter.(*gethInterpreterAdapter); ok {
		t.Fatal("geth reference interpreter is exported")
	}
}

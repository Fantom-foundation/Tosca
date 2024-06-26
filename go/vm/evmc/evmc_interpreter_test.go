// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package evmc

import (
	"math"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/ethereum/evmc/v11/bindings/go/evmc"
	"go.uber.org/mock/gomock"
)

func TestEvmcInterpreter_RevisionConversion(t *testing.T) {
	tests := []struct {
		tosca vm.Revision
		evmc  evmc.Revision
	}{
		{vm.R07_Istanbul, evmc.Istanbul},
		{vm.R09_Berlin, evmc.Berlin},
		{vm.R10_London, evmc.London},
		{vm.R11_Paris, evmc.Paris},
		{vm.R12_Shanghai, evmc.Shanghai},
	}

	for _, test := range tests {
		want := test.evmc
		got, err := toEvmcRevision(test.tosca)
		if err != nil {
			t.Fatalf("unexpected error during conversion: %v", err)
		}
		if want != got {
			t.Errorf("unexpected conversion of %v, wanted %v, got %v", test.tosca, want, got)
		}
	}
}

func TestEvmcInterpreter_RevisionConversionFailsOnUnknownRevision(t *testing.T) {
	_, err := toEvmcRevision(vm.Revision(math.MaxInt))
	if err == nil {
		t.Errorf("expected a conversion failure, got nothing")
	}
}

func TestEvmcInterpreter_SelfdestructCallsTheRightFunction(t *testing.T) {
	tests := map[string]struct {
		setup    func(*vm.MockRunContext)
		revision vm.Revision
	}{
		"pre-Cancun": {
			setup: func(runContext *vm.MockRunContext) {
				runContext.EXPECT().SelfDestruct(gomock.Any(), gomock.Any()).Times(1)
			},
			revision: vm.R12_Shanghai,
		},
		"Cancun": {
			setup: func(runContext *vm.MockRunContext) {
				runContext.EXPECT().SelfDestruct6780(gomock.Any(), gomock.Any())
			},
			revision: vm.R13_Cancun,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runContextCtrl := gomock.NewController(t)
			runContext := vm.NewMockRunContext(runContextCtrl)
			test.setup(runContext)

			ctx := &hostContext{
				context: runContext,
				params: vm.Parameters{
					BlockParameters: vm.BlockParameters{Revision: test.revision},
				},
			}

			ctx.Selfdestruct(evmc.Address{}, evmc.Address{1})
		})
	}
}

package lfvm

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"github.com/holiman/uint256"
)

func TestGas_CallGasCalculation(t *testing.T) {
	tests := map[string]struct {
		available vm.Gas       // < the gas available in the current context
		baseCosts vm.Gas       // < the costs for setting up the call
		provided  *uint256.Int // < the gas to be provided to the nested call
		want      vm.Gas       // < the gas costs for the call
	}{
		"available_is_more_than_provided": {
			available: vm.Gas(200),
			baseCosts: vm.Gas(20),
			provided:  uint256.NewInt(30),
			want:      30, // < limited by gas to be provided to nested call
		},
		"available_is_less_than_provided": {
			available: vm.Gas(200),
			baseCosts: vm.Gas(20),
			provided:  uint256.NewInt(300),
			want:      (200 - 20) - (200-20)/64, //  < limited by 63/64 of the available gas after the base costs
		},
		"available_is_less_than_provided_exceeding_maxUint64": {
			available: vm.Gas(200),
			baseCosts: vm.Gas(20),
			provided:  new(uint256.Int).Lsh(uint256.NewInt(1), 64),
			want:      (200 - 20) - (200-20)/64, //  < limited by 63/64 of the available gas after the base costs
		},
		"base_costs_higher_than_available": {
			available: vm.Gas(20),
			baseCosts: vm.Gas(200),
			provided:  uint256.NewInt(300),
			want:      200, //  < the base costs
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := callGas(test.available, test.baseCosts, test.provided)
			if want := test.want; want != got {
				t.Errorf("unexpected result, wanted %d, got %d", want, got)
			}
		})
	}
}

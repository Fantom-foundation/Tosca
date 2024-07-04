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

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/evmc/v11/bindings/go/evmc"
)

func TestEvmcInterpreter_RevisionConversion(t *testing.T) {
	tests := []struct {
		tosca tosca.Revision
		evmc  evmc.Revision
	}{
		{tosca.R07_Istanbul, evmc.Istanbul},
		{tosca.R09_Berlin, evmc.Berlin},
		{tosca.R10_London, evmc.London},
		{tosca.R11_Paris, evmc.Paris},
		{tosca.R12_Shanghai, evmc.Shanghai},
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
	_, err := toEvmcRevision(tosca.Revision(math.MaxInt))
	if err == nil {
		t.Errorf("expected a conversion failure, got nothing")
	}
}

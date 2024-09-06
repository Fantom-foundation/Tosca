// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package floria

import (
	"strings"
	"testing"

	test_utils "github.com/Fantom-foundation/Tosca/go/processor"
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestPrecompiled_RightNumberOfContractsDependingOnRevision(t *testing.T) {
	tests := []struct {
		revision          tosca.Revision
		numberOfContracts int
	}{
		{tosca.R07_Istanbul, 9},
		{tosca.R09_Berlin, 9},
		{tosca.R10_London, 9},
		{tosca.R11_Paris, 9},
		{tosca.R12_Shanghai, 9},
		{tosca.R13_Cancun, 10},
	}

	for _, test := range tests {
		count := 0
		for i := byte(0x01); i < byte(0x42); i++ {
			address := test_utils.NewAddress(i)
			_, isPrecompiled := getPrecompiledContract(address, test.revision)
			if isPrecompiled {
				count++
			}
		}
		if count != test.numberOfContracts {
			t.Errorf("unexpected number of precompiled contracts for revision %v, want %v, got %v", test.revision, test.numberOfContracts, count)
		}
		if len(getPrecompiledAddresses(test.revision)) != test.numberOfContracts {
			t.Errorf("unexpected number of precompiled contracts for revision %v, want %v, got %v", test.revision, test.numberOfContracts, count)
		}
	}
}

func TestPrecompiled_AddressesAreHandledCorrectly(t *testing.T) {
	tests := map[string]struct {
		revision      tosca.Revision
		address       tosca.Address
		gas           tosca.Gas
		isPrecompiled bool
		success       bool
	}{
		"nonPrecompiled":            {tosca.R09_Berlin, test_utils.NewAddress(0x20), 3000, false, false},
		"ecrecover-success":         {tosca.R10_London, test_utils.NewAddress(0x01), 3000, true, true},
		"ecrecover-outOfGas":        {tosca.R10_London, test_utils.NewAddress(0x01), 1, true, false},
		"pointEvaluation-success":   {tosca.R13_Cancun, test_utils.NewAddress(0x0a), 55000, true, true},
		"pointEvaluation-outOfGas":  {tosca.R13_Cancun, test_utils.NewAddress(0x0a), 1, true, false},
		"pointEvaluation-preCancun": {tosca.R10_London, test_utils.NewAddress(0x0a), 3000, false, false},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			input := tosca.Data{}
			if strings.Contains(name, "pointEvaluation") {
				input = test_utils.ValidPointEvaluationInput
			}

			result, isPrecompiled := handlePrecompiledContract(test.revision, input, test.address, test.gas)
			if isPrecompiled != test.isPrecompiled {
				t.Errorf("unexpected precompiled, want %v, got %v", test.isPrecompiled, isPrecompiled)
			}
			if result.Success != test.success {
				t.Errorf("unexpected success, want %v, got %v", test.success, result.Success)
			}
		})
	}
}

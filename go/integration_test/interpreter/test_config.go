// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package interpreter_test

import (
	"slices"
	"strings"

	_ "github.com/Fantom-foundation/Tosca/go/interpreter/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/interpreter/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/interpreter/geth"
	"github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"golang.org/x/exp/maps"
)

func init() {
	// Experimental LFVM configurations should be covered by integration tests
	// as they might be used by down-stream tools and for debugging.
	lfvm.RegisterExperimentalInterpreterConfigurations()
}

// getAllInterpreterVariantsForTests returns all registered interpreter variants
// that should be covered in integration tests.
func getAllInterpreterVariantsForTests() []string {
	// TODO: re-add logging variants once logging is no longer writing everything to stdout
	return slices.DeleteFunc(
		maps.Keys(tosca.GetAllRegisteredInterpreters()),
		func(s string) bool { return strings.Contains(s, "logging") },
	)
}

// skipTestForVariant returns true, if test should be skipped for variant
func skipTestForVariant(testName string, variant string) bool {
	disabledTest := map[string]map[string]bool{
		"TestNoReturnDataForCreate": {
			"evmone":          true,
			"evmone-basic":    true,
			"evmone-advanced": true,
		},
	}
	if disabled, found := disabledTest[testName][variant]; found && disabled {
		return true
	}
	return false
}

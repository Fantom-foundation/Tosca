//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package vm_test

import (
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/geth"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

var (
	Variants = []string{
		"geth",
		"lfvm",
		"lfvm-si",
		"lfvm-no-sha-cache",
		"lfvm-no-code-cache",
		//"lfvm-logging",
		"evmone",
		"evmone-basic",
		"evmone-advanced",
		"evmzero",
		//"evmzero-logging",
		"evmzero-no-analysis-cache",
		"evmzero-no-sha3-cache",
		"evmzero-profiling",
		"evmzero-profiling-external",
	}

	DisabledTest = map[string]map[string]bool{
		"TestNoReturnDataForCreate": {
			"evmone":          true,
			"evmone-basic":    true,
			"evmone-advanced": true,
		},
	}
)

// skipTestForVariant returns true, if test should be skipped for variant
func skipTestForVariant(testName string, variant string) bool {
	if disabled, found := DisabledTest[testName][variant]; found && disabled {
		return true
	}
	return false
}

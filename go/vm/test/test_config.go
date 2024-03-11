package vm_test

import (
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"

	//_ "github.com/Fantom-foundation/Tosca/go/vm/geth"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

var (
	Variants = []string{
		//"geth", // TODO: reenable once the geth EVM dependency is resolved
		"lfvm",
		"lfvm-si",
		"lfvm-no-sha-cache",
		"lfvm-no-code-cache",
		"lfvm-logging",
		"evmone",
		"evmone-basic",
		"evmone-advanced",
		"evmzero",
		"evmzero-logging",
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

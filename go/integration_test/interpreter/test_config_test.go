package interpreter_test

import (
	"slices"
	"testing"
)

func TestCoveredVariants_ContainsMainConfigurations(t *testing.T) {
	all := getAllInterpreterVariantsForTests()
	wanted := []string{
		"geth", "lfvm", "lfvm-si", "evmzero", "evmone",
	}
	for _, n := range wanted {
		if !slices.Contains(all, n) {
			t.Errorf("Variant %q is not registered, got %v", n, all)
		}
	}
}

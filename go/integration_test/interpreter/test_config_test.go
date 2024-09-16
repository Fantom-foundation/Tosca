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

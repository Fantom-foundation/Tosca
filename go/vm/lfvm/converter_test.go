//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package lfvm

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
)

func TestConvertLongExampleCode(t *testing.T) {
	clearConversionCache()
	_, err := Convert(longExampleCode, false, false, false, vm.Hash{})
	if err != nil {
		t.Errorf("Failed to convert example code with error %v", err)
	}
}

func BenchmarkConvertLongExampleCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		clearConversionCache()
		_, err := Convert(longExampleCode, false, false, false, vm.Hash{byte(i)})
		if err != nil {
			b.Errorf("Failed to convert example code with error %v", err)
		}
	}
}

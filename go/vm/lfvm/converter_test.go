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

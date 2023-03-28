package lfvm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestConvertLongExampleCode(t *testing.T) {
	addr := common.Address{}
	clearConversionCache()
	_, err := Convert(addr, longExampleCode, false, 0, false)
	if err != nil {
		t.Errorf("Failed to convert example code with error %v", err)
	}
}

func BenchmarkConvertLongExampleCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		clearConversionCache()
		addr := common.Address{}
		_, err := Convert(addr, longExampleCode, false, 0, false)
		if err != nil {
			b.Errorf("Failed to convert example code with error %v", err)
		}
	}
}

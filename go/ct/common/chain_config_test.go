package common

import (
	"math/big"
	"testing"
)

func TestChainConfig(t *testing.T) {
	chainConfig := GetChainConfig(big.NewInt(7))

	if want, got := big.NewInt(7), chainConfig.ChainID; want.Cmp(got) != 0 {
		t.Errorf("Unexpected chain id. wanted: %v, got %v", want, got)
	}
}

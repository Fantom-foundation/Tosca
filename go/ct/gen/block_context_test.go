package gen

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestBlockContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	blockContextGenerator := NewBlockContextGenerator()
	newBC, err := blockContextGenerator.Generate(rnd)

	if err != nil {
		t.Errorf("Error generating block context: %v", err)
	}

	if newBC.BlockNumber == (uint64(0)) {
		t.Errorf("Generated block number has default value.")
	}

	if newBC.CoinBase == (common.Address{}) {
		t.Errorf("Generated coinbase has default value.")
	}

	if newBC.GasLimit == (common.NewU256()) {
		t.Errorf("Generated gas limit has default value.")
	}

	if newBC.GasPrice == (common.NewU256()) {
		t.Errorf("Generated gas price has default value.")
	}

	if newBC.PrevRandao == ([32]byte{}) {
		t.Errorf("Generated prev randao has default value.")
	}

	if newBC.TimeStamp == (time.Time{}) {
		t.Errorf("Generated timestamp has default value.")
	}
}

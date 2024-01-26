package gen

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestBlockContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	blockContextGenerator := NewBlockContextGenerator()
	newBC, err := blockContextGenerator.Generate(rnd, common.Revision(rnd.Int31n(int32(common.R99_UnknownNextRevision)+1)))

	if err != nil {
		t.Errorf("Error generating block context: %v", err)
	}

	if newBC.BlockNumber == (uint64(0)) {
		t.Errorf("Generated block number has default value.")
	}

	if newBC.CoinBase == (common.Address{}) {
		t.Errorf("Generated coinbase has default value.")
	}

	if newBC.GasLimit == (uint64(0)) {
		t.Errorf("Generated gas limit has default value.")
	}

	if newBC.GasPrice == (common.NewU256()) {
		t.Errorf("Generated gas price has default value.")
	}

	if newBC.PrevRandao == (common.NewU256()) {
		t.Errorf("Generated prev randao has default value.")
	}

	if newBC.TimeStamp == (uint64(0)) {
		t.Errorf("Generated timestamp has default value.")
	}
}

func TestBlockContextGen_BlockNumber(t *testing.T) {
	istanbulBase, err := common.GetForkBlock(common.R07_Istanbul)
	if err != nil {
		t.Errorf("Failed to get Istanbul fork block number. %v", err)
	}
	berlinBase, err := common.GetForkBlock(common.R09_Berlin)
	if err != nil {
		t.Errorf("Failed to get Berlin fork block number. %v", err)
	}
	londonBase, err := common.GetForkBlock(common.R10_London)
	if err != nil {
		t.Errorf("Failed to get London fork block number. %v", err)
	}
	unknownBase, err := common.GetForkBlock(common.R99_UnknownNextRevision)
	if err != nil {
		t.Errorf("Failed to get future fork block number. %v", err)
	}

	tests := map[string]struct {
		revision common.Revision
		min      uint64
		max      uint64
	}{
		"Istanbul": {common.R07_Istanbul, istanbulBase, berlinBase},
		"Berlin":   {common.R09_Berlin, berlinBase, londonBase},
		"London":   {common.R10_London, londonBase, unknownBase},
		"future":   {common.R99_UnknownNextRevision, 0, 0},
	}
	rnd := rand.New(0)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			blockContextGenerator := NewBlockContextGenerator()
			b, err := blockContextGenerator.Generate(rnd, test.revision)
			if err != nil {
				t.Errorf("Error generating block context: %v", err)
			}
			if test.max != 0 && (test.min > b.BlockNumber || b.BlockNumber >= test.max) {
				t.Errorf("Generated block number %v outside of revision range", b.BlockNumber)
			} else if test.max == 0 && b.BlockNumber < unknownBase {
				t.Errorf("Generated block number %v outside of future revision range", b.BlockNumber)
			}
		})
	}

}

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

package gen

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

func TestBlockContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	blockContextGenerator := NewBlockContextGenerator()
	blockCtx, err := blockContextGenerator.Generate(rnd, common.Revision(rnd.Int31n(int32(common.R99_UnknownNextRevision)+1)))

	if err != nil {
		t.Errorf("Error generating block context: %v", err)
	}
	if blockCtx.BaseFee == (common.NewU256()) {
		t.Errorf("Generated base fee has default value.")
	}
	if blockCtx.BlockNumber == (uint64(0)) {
		t.Errorf("Generated block number has default value.")
	}
	if blockCtx.ChainID == (common.NewU256()) {
		t.Errorf("Generated chainid has default value.")
	}
	if blockCtx.CoinBase == (vm.Address{}) {
		t.Errorf("Generated coinbase has default value.")
	}
	if blockCtx.GasLimit == (uint64(0)) {
		t.Errorf("Generated gas limit has default value.")
	}
	if blockCtx.GasPrice == (common.NewU256()) {
		t.Errorf("Generated gas price has default value.")
	}
	if blockCtx.Difficulty == (common.NewU256()) {
		t.Errorf("Generated difficulty has default value.")
	}
	if blockCtx.TimeStamp == (uint64(0)) {
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
		"Unknown":  {common.R99_UnknownNextRevision, 0, 0},
	}
	rnd := rand.New(0)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			blockContextGenerator := NewBlockContextGenerator()
			blockCtx, err := blockContextGenerator.Generate(rnd, test.revision)
			if err != nil {
				t.Errorf("Error generating block context: %v", err)
			}
			if test.max != 0 && (test.min > blockCtx.BlockNumber || blockCtx.BlockNumber >= test.max) {
				t.Errorf("Generated block number %v outside of revision range", blockCtx.BlockNumber)
			} else if test.max == 0 && blockCtx.BlockNumber < unknownBase {
				t.Errorf("Generated block number %v outside of future revision range", blockCtx.BlockNumber)
			}
		})
	}
}

func TestBlockContextGen_BlockNumberError(t *testing.T) {
	rnd := rand.New(0)
	blockContextGenerator := NewBlockContextGenerator()
	_, err := blockContextGenerator.Generate(rnd, common.R99_UnknownNextRevision+1)
	if err == nil {
		t.Errorf("Failed to produce error with invalid revision.")
	}
}

//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package gen

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

func TestBlockContextGen_Generate(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}

	rnd := rand.New(0)
	blockContextGenerator := NewBlockContextGenerator()
	blockContextGenerator.AddBlockNumberOffsetConstraintIn(v1)
	blockCtx, err := blockContextGenerator.Generate(assignment, rnd, common.Revision(rnd.Int31n(int32(common.R99_UnknownNextRevision)+1)))

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
	if blockCtx.PrevRandao == (common.NewU256()) {
		t.Errorf("Generated prevRandao has default value.")
	}
	if blockCtx.TimeStamp == (uint64(0)) {
		t.Errorf("Generated timestamp has default value.")
	}
	if _, isAssigned := assignment[v1]; !isAssigned {
		t.Errorf("variable should have been assigned.")
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

	assignment := Assignment{}

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
			blockCtx, err := blockContextGenerator.Generate(assignment, rnd, test.revision)
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
	assignment := Assignment{}

	rnd := rand.New(0)
	blockContextGenerator := NewBlockContextGenerator()
	_, err := blockContextGenerator.Generate(assignment, rnd, common.R99_UnknownNextRevision+1)
	if err == nil {
		t.Errorf("Failed to produce error with invalid revision.")
	}
}

func TestBlockContextGen_BlockNumberOffsetVariableUnbound(t *testing.T) {
	v1 := Variable("v1")
	rnd := rand.New()

	tests := map[string]struct {
		addConstraint func(*BlockContextGenerator)
		check         func(value, assignmentValue uint64) bool
	}{
		"WithinRange": {addConstraint: func(b *BlockContextGenerator) { b.AddBlockNumberOffsetConstraintIn("v1") },
			check: func(blockNumber, assignmentValue uint64) bool {
				return blockNumber > assignmentValue && assignmentValue >= blockNumber-256
			}},
		"FixedValue257": {addConstraint: func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("v1", 257) },
			check: func(blockNumber, assignmentValue uint64) bool {
				return assignmentValue == blockNumber-257
			}},
		"FixedValue256": {addConstraint: func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("v1", 256) },
			check: func(blockNumber, assignmentValue uint64) bool {
				return assignmentValue == blockNumber-256
			}},
		"FixedValue255": {addConstraint: func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("v1", 255) },
			check: func(blockNumber, assignmentValue uint64) bool {
				return assignmentValue == blockNumber-255
			}},
		"FixedValue1": {addConstraint: func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("v1", 1) },
			check: func(blockNumber, assignmentValue uint64) bool {
				return assignmentValue == blockNumber-1
			}},
		"FixedValue0": {addConstraint: func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("v1", 0) },
			check: func(blockNumber, assignmentValue uint64) bool {
				return assignmentValue == blockNumber-0
			}},
		"FixedValue-1": {addConstraint: func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("v1", -1) },
			check: func(blockNumber, assignmentValue uint64) bool {
				return assignmentValue == blockNumber+1
			}},
		"OutOfRange": {addConstraint: func(b *BlockContextGenerator) { b.AddBlockNumberOffsetConstraintOut("v1") },
			check: func(blockNumber, assignmentValue uint64) bool {
				return blockNumber <= assignmentValue || assignmentValue < blockNumber-256
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			assignment := Assignment{}
			blockContextGenerator := NewBlockContextGenerator()
			test.addConstraint(blockContextGenerator)
			blockCtx, err := blockContextGenerator.Generate(assignment, rnd, common.MaxRevision-1)
			if err != nil {
				t.Errorf("Error generating block context: %v", err)
			}
			assignmentValue := assignment[v1].Uint64()
			if !test.check(blockCtx.BlockNumber, assignmentValue) {
				t.Errorf("Generated variable %v not in desired distance from block number %v.", assignment[v1].Uint64(), blockCtx.BlockNumber)
			}
		})
	}
}

func TestBlockContextGen_BlockNumberOffsetError(t *testing.T) {
	rnd := rand.New(0)

	tests := map[string]struct {
		fn func(*BlockContextGenerator)
	}{
		"outFirst": {fn: func(b *BlockContextGenerator) {
			b.AddBlockNumberOffsetConstraintOut("v1")
			b.AddBlockNumberOffsetConstraintIn("v1")
		}},
		"inFirst": {fn: func(b *BlockContextGenerator) {
			b.AddBlockNumberOffsetConstraintIn("v1")
			b.AddBlockNumberOffsetConstraintOut("v1")
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assignment := Assignment{}
			blockContextGenerator := NewBlockContextGenerator()
			test.fn(blockContextGenerator)
			_, err := blockContextGenerator.Generate(assignment, rnd, common.R07_Istanbul)
			if err != ErrUnsatisfiable {
				t.Errorf("Failed to produce error with conflicting range constraints.")
			}
		})
	}
}

func TestBlockContextGen_BlockNumberOffsetVariableBound(t *testing.T) {
	v1 := Variable("v1")
	rnd := rand.New(uint64(time.Now().UnixNano()))
	assignment := Assignment{}

	assignmentValues := []common.U256{common.NewU256(512), common.NewU256(257),
		common.NewU256(256), common.NewU256(255), common.NewU256(1), common.NewU256(0)}

	tests := map[string]struct {
		fn    func(*BlockContextGenerator)
		check func(uint64, uint64) bool
	}{
		"inRange": {fn: func(b *BlockContextGenerator) { b.AddBlockNumberOffsetConstraintIn("v1") },
			check: func(blockNumber, generated uint64) bool {
				min := uint64(0)
				if blockNumber > 256 {
					min = blockNumber - 256
				}
				return blockNumber > generated && min <= generated
			},
		},
		"outRange": {fn: func(b *BlockContextGenerator) { b.AddBlockNumberOffsetConstraintOut("v1") },
			check: func(blockNumber, generated uint64) bool {
				return blockNumber < generated || blockNumber-256 >= generated
			},
		},
		"fixedValue": {fn: func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("v1", 256) },
			check: func(blockNumber, generated uint64) bool {
				return generated == 256 && blockNumber-generated < 257
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, value := range assignmentValues {
				assignment[v1] = value
				blockContextGenerator := NewBlockContextGenerator()
				test.fn(blockContextGenerator)
				blockContext, err := blockContextGenerator.Generate(assignment, rnd, common.R07_Istanbul)
				if err != nil {
					if value != common.NewU256(256) && err != ErrUnsatisfiable {
						t.Errorf("Error generating block context: %v", err)
					} else if value != common.NewU256(256) && err == ErrUnsatisfiable {
						continue
					}
				}
				blckNum := blockContext.BlockNumber
				if assignment[v1] != value {
					t.Error("assigned value should not have changed.")
				}
				if !test.check(blckNum, assignment[v1].Uint64()) {
					t.Errorf("Block number should be in the expected range. got %v. assigned %v.", blckNum, assignment[v1].Uint64())
				}
			}
		})
	}
}

func TestBlockContextGen_Clone(t *testing.T) {
	blockContextGenerator := NewBlockContextGenerator()
	blockContextGenerator.variablesOffsetConstraints = append(blockContextGenerator.variablesOffsetConstraints, constraintPair{
		variable: "v1", lowerOffset: 1, upperOffset: 2})

	clone := blockContextGenerator.Clone()
	clone.variablesOffsetConstraints[0].lowerOffset = 3

	if blockContextGenerator.variablesOffsetConstraints[0].lowerOffset != 1 {
		t.Errorf("Original generator should not be affected by clone.")
	}
}

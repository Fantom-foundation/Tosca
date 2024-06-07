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
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

func TestBlockContextGen_Generate(t *testing.T) {
	v1 := Variable("v1")
	assignment := Assignment{}

	rnd := rand.New(0)
	blockContextGenerator := NewBlockContextGenerator()
	blockCtx, err := blockContextGenerator.Generate(assignment, rnd)

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

func TestBlockContextGen_PrintProducesConstraintFormula(t *testing.T) {
	tests := map[string]struct {
		setup func(*BlockContextGenerator)
		want  string
	}{
		"empty": {
			setup: func(b *BlockContextGenerator) {},
			want:  "true",
		},
		"restrict-to-single-revision": {
			setup: func(b *BlockContextGenerator) {
				b.SetRevision(common.R07_Istanbul)
			},
			want: "0≤BlockNumber≤999",
		},
		"restrict-to-revision-range": {
			setup: func(b *BlockContextGenerator) {
				b.AddRevisionBounds(common.R09_Berlin, common.R11_Paris)
			},
			// TODO: clean up this expectation by using GetForkBlock function
			want: "1000≤BlockNumber≤3999",
		},
		"restrict-to-multiple-revisions": {
			setup: func(b *BlockContextGenerator) {
				b.SetRevision(common.R07_Istanbul)
				b.SetRevision(common.R09_Berlin)
			},
			want: "false",
		},
		"restrict-to-multiple-revisions-that-are-not-conflicting": {
			setup: func(b *BlockContextGenerator) {
				b.AddRevisionBounds(common.R07_Istanbul, common.R09_Berlin)
				b.SetRevision(common.R07_Istanbul)
			},
			want: "0≤BlockNumber≤999",
		},
		"one-variable-in-range": {
			setup: func(b *BlockContextGenerator) {
				b.RestrictVariableToOneOfTheLast256Blocks("a")
			},
			want: "$a ∈ [BlockNumber-256..BlockNumber-1]",
		},
		"one-variable-out-of-range": {
			setup: func(b *BlockContextGenerator) {
				b.RestrictVariableToNoneOfTheLast256Blocks("a")
			},
			want: "$a ∉ [BlockNumber-256..BlockNumber-1]",
		},
		"two-variables-in-range": {
			setup: func(b *BlockContextGenerator) {
				b.RestrictVariableToOneOfTheLast256Blocks("a")
				b.RestrictVariableToOneOfTheLast256Blocks("b")
			},
			want: "$a ∈ [BlockNumber-256..BlockNumber-1] Λ $b ∈ [BlockNumber-256..BlockNumber-1]",
		},
		"one-variable-with-fixed-offset": {
			setup: func(b *BlockContextGenerator) {
				b.SetBlockNumberOffsetValue("a", 44)
			},
			want: "$a = BlockNumber-44",
		},
		"one-variable-with-fixed-offset-in-the-future": {
			setup: func(b *BlockContextGenerator) {
				b.SetBlockNumberOffsetValue("a", -44)
			},
			want: "$a = BlockNumber+44",
		},
		"mix-of-multiple-constraints": {
			setup: func(b *BlockContextGenerator) {
				b.SetRevision(common.R07_Istanbul)
				b.SetBlockNumberOffsetValue("a", 44)
				b.RestrictVariableToOneOfTheLast256Blocks("b")
				b.RestrictVariableToNoneOfTheLast256Blocks("c")
			},
			want: "$a = BlockNumber-44 Λ $b ∈ [BlockNumber-256..BlockNumber-1] Λ $c ∉ [BlockNumber-256..BlockNumber-1] Λ 0≤BlockNumber≤999",
		},
		"multiple-constraints-for-single-variable": {
			setup: func(b *BlockContextGenerator) {
				b.SetBlockNumberOffsetValue("a", 44)
				b.RestrictVariableToOneOfTheLast256Blocks("a")
			},
			want: "$a = BlockNumber-44 Λ $a ∈ [BlockNumber-256..BlockNumber-1]",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gen := NewBlockContextGenerator()
			test.setup(gen)
			if got := gen.String(); test.want != got {
				t.Errorf("unexpected print, wanted %s, got %s", test.want, got)
			}
		})
	}
}

func TestBlockContextGenerator_CanProduceSatisfyingBlockNumbersForConstraints(t *testing.T) {
	tests := map[string]struct {
		setup func(*BlockContextGenerator, Assignment)
		check func(*testing.T, st.BlockContext, Assignment)
	}{
		"no-constraints": {
			setup: func(b *BlockContextGenerator, a Assignment) {},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if len(a) != 0 {
					t.Errorf("solver should have not added any assignments, got %d", len(a))
				}
			},
		},
		"fix-revision": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.SetRevision(common.R10_London)
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if want, got := common.R10_London, common.GetRevisionForBlock(res.BlockNumber); want != got {
					t.Errorf("unexpected revision, wanted %v, got %v", want, got)
				}
			},
		},
		"revision-range": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.AddRevisionBounds(common.R10_London, common.R11_Paris)
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				got := common.GetRevisionForBlock(res.BlockNumber)
				if got < common.R10_London || got > common.R11_Paris {
					t.Errorf("unexpected revision, got %v, wanted something between London and Paris", got)
				}
			},
		},
		"variable-with-fixed-offset": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.SetBlockNumberOffsetValue("a", 44)
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if len(a) != 1 {
					t.Errorf("expected only one variable to be assigned, got %v", a)
				}
				value, present := a["a"]
				if !present {
					t.Errorf("expected variable 'a' to be assigned, got %v", a)
				} else if !value.IsUint64() {
					t.Errorf("value assigned to 'a' is out of range: %v", value)
				} else {
					value := value.Uint64()
					offset := res.BlockNumber - value
					if offset != 44 {
						t.Errorf("wanted an offset of %d, got %d, block number %d", 44, offset, res.BlockNumber)
					}
				}
			},
		},
		"variable-with-fixed-positive-offset-and-predefined-offset": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.SetBlockNumberOffsetValue("a", 44)
				a["a"] = common.NewU256(100)
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if len(a) != 1 {
					t.Errorf("expected only one variable to be assigned, got %v", a)
				}
				value, present := a["a"]
				if !present {
					t.Errorf("expected variable 'a' to be assigned, got %v", a)
				} else if value != common.NewU256(100) {
					t.Errorf("solver should not alter assigned value, got %v", value)
				} else if res.BlockNumber != 144 {
					t.Errorf("expected block number to be 144, got %d", res.BlockNumber)
				}
			},
		},
		"variable-with-fixed-negative-offset-and-predefined-offset": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.SetBlockNumberOffsetValue("a", -44)
				a["a"] = common.NewU256(100)
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if len(a) != 1 {
					t.Errorf("expected only one variable to be assigned, got %v", a)
				}
				value, present := a["a"]
				if !present {
					t.Errorf("expected variable 'a' to be assigned, got %v", a)
				} else if value != common.NewU256(100) {
					t.Errorf("solver should not alter assigned value, got %v", value)
				} else if res.BlockNumber != 56 {
					t.Errorf("expected block number to be 56, got %d", res.BlockNumber)
				}
			},
		},
		"variable-with-fixed-offset-beyond-uint64-range": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.SetBlockNumberOffsetValue("a", -44)
				a["a"] = common.NewU256(1, 12) // 2^64+12
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if got, want := res.BlockNumber, common.NewU256(1, 12).Sub(common.NewU256(44)).Uint64(); got != want {
					t.Errorf("expected block number to be %d, got %d", want, got)
				}
			},
		},
		"variable-in-range": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.RestrictVariableToOneOfTheLast256Blocks("a")
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				value, present := a["a"]
				if !present {
					t.Errorf("expected variable 'a' to be assigned, got %v", a)
				} else if !value.IsUint64() {
					t.Errorf("value assigned to 'a' is out of range: %v", value)
				} else {
					value := value.Uint64()
					if res.BlockNumber-value > 256 || res.BlockNumber-value < 1 {
						t.Errorf("unexpected distance between variable 'a' and block number, got block number %d and assignment %d", res.BlockNumber, value)
					}
				}
			},
		},
		"variable-in-range-with-pre-assigned-offset": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.SetRevision(common.R07_Istanbul)
				b.RestrictVariableToOneOfTheLast256Blocks("a")
				a["a"] = common.NewU256(800)
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if !(800 < res.BlockNumber && res.BlockNumber < 1000) {
					t.Errorf("expected block number to be in range 801-1000, got %d", res.BlockNumber)
				}
			},
		},
		"variable-out-of-range": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.RestrictVariableToNoneOfTheLast256Blocks("a")
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				value, present := a["a"]
				if !present {
					t.Errorf("expected variable 'a' to be assigned, got %v", a)
				} else if !value.IsUint64() {
					// this is fine, the value is out of range
				} else {
					value := value.Uint64()
					if !(res.BlockNumber-value > 256 || res.BlockNumber-value < 1) {
						t.Errorf("unexpected distance between variable 'a' and block number, got block number %d and assignment %d", res.BlockNumber, value)
					}
				}
			},
		},
		"variable-out-of-range-with-pre-assigned-offset": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.SetRevision(common.R07_Istanbul)
				b.RestrictVariableToNoneOfTheLast256Blocks("a")
				a["a"] = common.NewU256(8000)
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if res.BlockNumber >= 1000 {
					t.Errorf("produced block number is not a valid Istanbul block, got %d", res.BlockNumber)
				}
			},
		},
		"tight-block-range-with-single-solution": {
			setup: func(b *BlockContextGenerator, a Assignment) {
				b.blockNumberSolver = NewRangeSolver[uint64](0, 1)
				b.RestrictVariableToOneOfTheLast256Blocks("a")
			},
			check: func(t *testing.T, res st.BlockContext, a Assignment) {
				if res.BlockNumber != 1 {
					t.Errorf("expected block number to be 1, got %d", res.BlockNumber)
				}
				if value := a["a"]; value != common.NewU256(0) {
					t.Errorf("expected variable 'a' to be assigned to 0, got %v", value)
				}
			},
		},
	}

	randomSource := rand.New()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			generator := NewBlockContextGenerator()
			assignment := Assignment{}
			test.setup(generator, assignment)
			res, err := generator.Generate(assignment, randomSource)
			if err != nil {
				t.Fatalf("Error generating result for constraints %v and assignment %v: %v", generator, assignment, err)
			}
			test.check(t, res, assignment)
		})
	}
}

func TestBlockContextGenerator_SignalsUnsatisfiableForUnsatisfiableConstraints(t *testing.T) {
	// TODO: add support for pre-assigned values
	tests := map[string]func(*BlockContextGenerator, Assignment){
		"conflicting-revisions": func(b *BlockContextGenerator, _ Assignment) {
			b.SetRevision(common.R07_Istanbul)
			b.SetRevision(common.R09_Berlin)
		},
		"conflicting-ranges-in-first": func(b *BlockContextGenerator, _ Assignment) {
			b.RestrictVariableToOneOfTheLast256Blocks("a")
			b.RestrictVariableToNoneOfTheLast256Blocks("a")
		},
		"conflicting-ranges-out-first": func(b *BlockContextGenerator, _ Assignment) {
			b.RestrictVariableToNoneOfTheLast256Blocks("a")
			b.RestrictVariableToOneOfTheLast256Blocks("a")
		},
		"conflicting-fixed-offsets": func(b *BlockContextGenerator, _ Assignment) {
			b.SetBlockNumberOffsetValue("a", 44)
			b.SetBlockNumberOffsetValue("a", 45)
		},
		"conflicting-fixed-offsets-with-out-of-range": func(b *BlockContextGenerator, _ Assignment) {
			b.SetBlockNumberOffsetValue("a", 44)
			b.RestrictVariableToNoneOfTheLast256Blocks("a")
		},
		"conflicting-fixed-offsets-with-in-range": func(b *BlockContextGenerator, _ Assignment) {
			b.SetBlockNumberOffsetValue("a", 400)
			b.RestrictVariableToOneOfTheLast256Blocks("a")
		},
		"block-number-overflow": func(b *BlockContextGenerator, a Assignment) {
			b.SetBlockNumberOffsetValue("a", 400)
			a["a"] = common.NewU256(1, 500) // 2^64+500
		},
		"block-number-underflow": func(b *BlockContextGenerator, a Assignment) {
			b.SetRevision(common.R07_Istanbul)
			b.SetBlockNumberOffsetValue("a", 1100)
		},
		"conflicting-revisions-with-in-range-and-predefined-assignment": func(b *BlockContextGenerator, a Assignment) {
			b.SetRevision(common.R07_Istanbul)
			b.RestrictVariableToOneOfTheLast256Blocks("a")
			a["a"] = common.NewU256(8000)
		},
		"conflicting-revisions-with-in-range-and-predefined-assignment-bigger-than-uint64": func(b *BlockContextGenerator, a Assignment) {
			b.SetRevision(common.R07_Istanbul)
			b.RestrictVariableToOneOfTheLast256Blocks("a")
			a["a"] = common.NewU256(1, 1)
		},
		"conflicting-revisions-with-in-range-and-predefined-assignment-max-uint64": func(b *BlockContextGenerator, a Assignment) {
			b.SetRevision(common.R07_Istanbul)
			b.RestrictVariableToOneOfTheLast256Blocks("a")
			a["a"] = common.NewU256(0xffffffffffffffff)
		},
		"no-valid-block-range": func(b *BlockContextGenerator, a Assignment) {
			b.blockNumberSolver = NewRangeSolver[uint64](0, 0)
			b.RestrictVariableToOneOfTheLast256Blocks("a")
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assignment := Assignment{}
			generator := NewBlockContextGenerator()
			test(generator, assignment)
			res, err := generator.Generate(assignment, rand.New())
			if err != ErrUnsatisfiable {
				t.Errorf("expected unsatisfiable error, got %v with block number %d and assignment %v", err, res.BlockNumber, assignment)
			}
		})
	}
}

func TestBlockContextGen_addOffsetPositive(t *testing.T) {

	tests := map[string]struct {
		base     common.U256
		offset   int64
		want     common.U256
		overflow bool
	}{
		"small-positive": {
			base:     common.NewU256(44),
			offset:   44,
			want:     common.NewU256(88),
			overflow: false,
		},
		"small-negative": {
			base:     common.NewU256(44),
			offset:   -44,
			want:     common.NewU256(0),
			overflow: false,
		},
		"big-positive": {
			base:     common.NewU256(0xffffffffffffffff),
			offset:   1,
			want:     common.NewU256(1, 0),
			overflow: false,
		},
		"big-negative": {
			base:     common.NewU256(1, 0),
			offset:   -1,
			want:     common.NewU256(0xffffffffffffffff),
			overflow: false,
		},
		"overflow": {
			base: common.NewU256(0xffffffffffffffff, 0xffffffffffffffff,
				0xffffffffffffffff, 0xffffffffffffffff),
			offset:   1,
			want:     common.NewU256(0),
			overflow: true,
		},
		"underflow": {
			base:   common.NewU256(0),
			offset: -1,
			want: common.NewU256(0xffffffffffffffff, 0xffffffffffffffff,
				0xffffffffffffffff, 0xffffffffffffffff),
			overflow: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, overflow := addOffset(test.base, test.offset)
			if overflow != test.overflow {
				t.Errorf("expected overflow %v, got %v", test.overflow, overflow)
			}
			if !result.Eq(test.want) {
				t.Errorf("expected %v, got %v", test.want, result)
			}
		})
	}
}

func TestBlockContextGen_addWithOverflowCheck(t *testing.T) {

	tests := map[string]struct {
		base     uint64
		offset   uint64
		want     uint64
		overflow bool
	}{
		"small-positive": {base: 44, offset: 44, want: 88, overflow: false},
		"overflow":       {base: 0xffffffffffffffff, offset: 1, want: 0, overflow: true},
		"max":            {base: 0xffffffffffffffff, offset: 0xffffffffffffffff, want: 0xfffffffffffffffe, overflow: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, overflow := addWithOverflowCheck(test.base, test.offset)
			if overflow != test.overflow {
				t.Errorf("expected overflow %v, got %v", test.overflow, overflow)
			}
			if result != test.want {
				t.Errorf("expected %v, got %v", test.want, result)
			}
		})
	}
}

func TestBlockContextGen_Restore(t *testing.T) {

	tests := map[string]BlockContextGenerator{
		"empty":     *NewBlockContextGenerator(),
		"non-empty": {blockNumberSolver: NewRangeSolver[uint64](0, 100)},
	}

	for name, blockContextGen := range tests {
		t.Run(name, func(t *testing.T) {
			restoreMe := NewBlockContextGenerator()
			restoreMe.Restore(&blockContextGen)
			if want, got := blockContextGen.String(), restoreMe.String(); want != got {
				t.Errorf("expected %v, got %v", want, got)
			}
		})
	}
}

func TestBlockContextGen_UnsatisfiableStateDoesNotChange(t *testing.T) {

	tests := map[string]func(*BlockContextGenerator){
		"fix-offset":   func(b *BlockContextGenerator) { b.SetBlockNumberOffsetValue("a", 44) },
		"in-range":     func(b *BlockContextGenerator) { b.RestrictVariableToOneOfTheLast256Blocks("a") },
		"out-of-range": func(b *BlockContextGenerator) { b.RestrictVariableToNoneOfTheLast256Blocks("a") },
		"revision":     func(b *BlockContextGenerator) { b.SetRevision(common.R07_Istanbul) },
	}

	b := NewBlockContextGenerator()
	b.unsatisfiable = true
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := b.Clone()
			test(c)
			if want, got := b.String(), c.String(); want != got {
				t.Errorf("unsatisfiable generator should not change, expected %v, got %v", want, got)
			}
		})
	}
}

func TestBlockContextGen_BlockNumberOffsetVariableUnbound(t *testing.T) {
	v1 := Variable("v1")
	rnd := rand.New()

	tests := map[string]struct {
		addConstraint func(*BlockContextGenerator)
		check         func(value, assignmentValue uint64) bool
	}{
		"WithinRange": {addConstraint: func(b *BlockContextGenerator) { b.RestrictVariableToOneOfTheLast256Blocks("v1") },
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
		"OutOfRange": {addConstraint: func(b *BlockContextGenerator) { b.RestrictVariableToNoneOfTheLast256Blocks("v1") },
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
			blockCtx, err := blockContextGenerator.Generate(assignment, rnd)
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
			b.RestrictVariableToNoneOfTheLast256Blocks("v1")
			b.RestrictVariableToOneOfTheLast256Blocks("v1")
		}},
		"inFirst": {fn: func(b *BlockContextGenerator) {
			b.RestrictVariableToOneOfTheLast256Blocks("v1")
			b.RestrictVariableToNoneOfTheLast256Blocks("v1")
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assignment := Assignment{}
			blockContextGenerator := NewBlockContextGenerator()
			test.fn(blockContextGenerator)
			_, err := blockContextGenerator.Generate(assignment, rnd)
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
		"inRange": {fn: func(b *BlockContextGenerator) { b.RestrictVariableToOneOfTheLast256Blocks("v1") },
			check: func(blockNumber, generated uint64) bool {
				min := uint64(0)
				if blockNumber > 256 {
					min = blockNumber - 256
				}
				return blockNumber > generated && min <= generated
			},
		},
		"outRange": {fn: func(b *BlockContextGenerator) { b.RestrictVariableToNoneOfTheLast256Blocks("v1") },
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
				blockContext, err := blockContextGenerator.Generate(assignment, rnd)
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

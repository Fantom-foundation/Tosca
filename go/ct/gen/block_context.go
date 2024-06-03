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
	"fmt"
	"maps"
	"math"
	"sort"
	"strings"

	"pgregory.net/rand"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// BlockContextGenerator is a generator for block contexts.
// It can take constraints on the block number, which come from
// restricting the revision, and it can also take constraints on
// the on different variables to generta in respect to the block number.
// The constraints can be added in the form of fixed-value constraints
// on variables, and constraints on the range of values that variables
// can take.
// The generator can be marked as unsatisfiable if the constraints
// added to it are conflicting.
type BlockContextGenerator struct {
	blockNumberSolver *RangeSolver[uint64]

	// This map is used to keep track of range constraints of variables.
	// variables limited to recent block constraints are mapped to
	// true, variables limited to be not in recent block constraints
	// are mapped to false.
	rangeConstraints map[Variable]bool

	// This map is used to keep track of fixed-value constraints of variables.
	valueConstraint map[Variable]int64

	// This flag is set to true if the contained set of constraints
	// is not satisfiable.
	unsatisfiable bool
}

func NewBlockContextGenerator() *BlockContextGenerator {
	return &BlockContextGenerator{}
}

// addOffset adds an offset to a U256 value, checking for overflow.
func addOffset(value U256, offset int64) (result U256, hasOverflown bool) {
	if offset < 0 {
		result := value.Sub(NewU256(uint64(-offset)))
		return result, result.Gt(value)
	}
	result = value.Add(NewU256(uint64(offset)))
	return result, result.Lt(value)
}

// addWithOverflowCheck adds two uint64 values, checking for overflow.
func addWithOverflowCheck(a, b uint64) (result uint64, hasOverflown bool) {
	result = a + b
	return result, result < a
}

// generateResultingBlockNumber generates a block number based on the constraints
// added to the generator, this could be regarding revision constraints,
// fixed-value constraints, range constraints, or previously assigned variables
// ( the latter only if they are also target variable for any relevant constraint )
func (b *BlockContextGenerator) generateResultingBlockNumber(assignment Assignment, rnd *rand.Rand) (uint64, error) {

	blockNumberSolver := NewIntervalSolver[uint64](0, math.MaxUint64)
	if b.blockNumberSolver != nil {
		// apply constraints on block number solver derived from revision constraints
		if b.blockNumberSolver.min > 0 {
			blockNumberSolver.Exclude(0, b.blockNumberSolver.min-1)
		}
		if b.blockNumberSolver.max < math.MaxUint64 {
			blockNumberSolver.Exclude(b.blockNumberSolver.max+1, math.MaxUint64)
		}
	}

	// apply constraints on block number solver derived from predefined assignments
	// 1) fixed offset constraints
	for variable, offset := range b.valueConstraint {
		if assignedValue, isBound := assignment[variable]; isBound {
			wantedBlock, overflow := addOffset(assignedValue, offset)
			if overflow || !wantedBlock.IsUint64() {
				return 0, ErrUnsatisfiable
			}
			blockNumberSolver.AddEqualityConstraint(wantedBlock.Uint64())
		}
	}

	// 2) add constraints on block number solver derived from range constraints
	for variable, inRange := range b.rangeConstraints {
		if assignedValue, isBound := assignment[variable]; isBound {
			if inRange {
				if !assignedValue.IsUint64() {
					return 0, ErrUnsatisfiable
				}
				value := assignedValue.Uint64()
				if lower, overflow := addWithOverflowCheck(value, 1); !overflow {
					blockNumberSolver.AddLowerBoundary(lower)
				} else {
					return 0, ErrUnsatisfiable
				}
				if upper, overflow := addWithOverflowCheck(value, 256); !overflow {
					blockNumberSolver.AddUpperBoundary(upper)
				}
			} else {
				if !assignedValue.Sub(NewU256(256)).IsUint64() {
					return 0, ErrUnsatisfiable
				}
				value := assignedValue.Uint64()
				blockNumberSolver.Exclude(value+1, value+256)
			}
		}
	}

	return blockNumberSolver.Generate(rnd)
}

// processValueConstraint processes the fixed-value constraints on variables.
func (b *BlockContextGenerator) processValueConstraint(assignment Assignment, resultingBlockNumber uint64) error {
	for variable, offset := range b.valueConstraint {
		requiredValue, underflow := addOffset(NewU256(resultingBlockNumber), -offset)
		if underflow {
			return ErrUnsatisfiable
		}
		if _, isBound := assignment[variable]; !isBound {
			assignment[variable] = requiredValue
		}
		// there is no else case needed because:
		// 1) if the variable was assigned before generating a block number, then resultingBlockNumber := assignedValue+offset
		//    which would result in requiresValue == assignedValue
		// 2) if the variable was not assigned before generating a block number, it means the variable was assigned by
		//    another valueConstraint, but if these two were conflicting, the generator would have been marked as
		//    unsatisfiable when the second constraint was added.
	}
	return nil
}

// processRangeConstraint processes the range constraints on variables.
func (b *BlockContextGenerator) processRangeConstraint(resultingBlockNumber uint64, assignment Assignment, rnd *rand.Rand) error {
	lowerBound := uint64(0)
	if resultingBlockNumber > 256 {
		lowerBound = resultingBlockNumber - 256
	}
	upperBound := resultingBlockNumber
	blockNumberRangeGenerator := NewRangeSolver(lowerBound, upperBound-1)

	for variable, inRange := range b.rangeConstraints {
		if currentValue, isBound := assignment[variable]; isBound {
			value := currentValue.Uint64()
			if inRange {
				if !isInRange(lowerBound, currentValue, upperBound) {
					return ErrUnsatisfiable
				}
			} else {
				if currentValue.IsUint64() {
					if lowerBound <= value && value < upperBound {
						return ErrUnsatisfiable
					}
				}
			}
		} else { // no value bound to variable yet
			if inRange {
				blockNumber, err := blockNumberRangeGenerator.Generate(rnd)
				if err != nil {
					return err
				}
				assignment[variable] = NewU256(blockNumber)
			} else {
				number := rnd.Uint64n(math.MaxUint64 - 256)
				if number >= lowerBound {
					number += 256
				}
				assignment[variable] = NewU256(number)
			}
		}
	}
	return nil
}

func (b *BlockContextGenerator) Generate(assignment Assignment, rnd *rand.Rand) (st.BlockContext, error) {
	if b.unsatisfiable {
		return st.BlockContext{}, ErrUnsatisfiable
	}

	// this call takes into account all preassigned values and revision constraints to generate a block number
	resultingBlockNumber, err := b.generateResultingBlockNumber(assignment, rnd)
	if err != nil {
		return st.BlockContext{}, err
	}

	// for all non bound relevant variables, assign them a value based on the constraints.
	// 1) fixed offset constraints
	if err := b.processValueConstraint(assignment, resultingBlockNumber); err != nil {
		return st.BlockContext{}, err
	}

	// 2) range constraints
	if err := b.processRangeConstraint(resultingBlockNumber, assignment, rnd); err != nil {
		return st.BlockContext{}, err
	}

	chainId := RandU256(rnd)
	coinbase, err := RandAddress(rnd)
	if err != nil {
		return st.BlockContext{}, err
	}

	baseFee := RandU256(rnd)
	gasLimit := rnd.Uint64()
	gasPrice := RandU256(rnd)

	prevRandao := RandU256(rnd)

	revision := GetRevisionForBlock(resultingBlockNumber)
	time := GetForkTime(revision)
	nextTime := GetForkTime(revision + 1)
	timestamp := rnd.Uint64n(nextTime-time) + time

	return st.BlockContext{
		BaseFee:     baseFee,
		BlockNumber: resultingBlockNumber,
		ChainID:     chainId,
		CoinBase:    coinbase,
		GasLimit:    gasLimit,
		GasPrice:    gasPrice,
		PrevRandao:  prevRandao,
		TimeStamp:   timestamp,
	}, nil
}

func isInRange(lowerBound uint64, currentValue U256, upperBound uint64) bool {
	value := currentValue.Uint64()
	return currentValue.IsUint64() && lowerBound <= value && value <= upperBound
}

func (b *BlockContextGenerator) Clone() *BlockContextGenerator {
	var blockNumberSolverCopy *RangeSolver[uint64]
	if b.blockNumberSolver != nil {
		blockNumberSolverCopy = b.blockNumberSolver
	}
	return &BlockContextGenerator{
		unsatisfiable:     b.unsatisfiable,
		blockNumberSolver: blockNumberSolverCopy,
		rangeConstraints:  maps.Clone(b.rangeConstraints),
		valueConstraint:   maps.Clone(b.valueConstraint),
	}
}

func (b *BlockContextGenerator) Restore(o *BlockContextGenerator) {
	b.unsatisfiable = o.unsatisfiable
	if o.blockNumberSolver != nil {
		b.blockNumberSolver = o.blockNumberSolver.Clone()
	}
	b.rangeConstraints = maps.Clone(o.rangeConstraints)
	b.valueConstraint = maps.Clone(o.valueConstraint)
}

func (b *BlockContextGenerator) String() string {
	if b.unsatisfiable {
		return "false"
	}
	if b.blockNumberSolver == nil && b.rangeConstraints == nil && b.valueConstraint == nil {
		return "true"
	}

	clauses := []string{}
	if b.blockNumberSolver != nil {
		if b.blockNumberSolver.IsSatisfiable() {
			clauses = append(clauses, fmt.Sprintf(
				"BlockNumber ∈ [%d..%d]",
				b.blockNumberSolver.min,
				b.blockNumberSolver.max,
			))
		}
	}

	if b.rangeConstraints != nil {
		for variable, inRange := range b.rangeConstraints {
			if inRange {
				clauses = append(clauses, fmt.Sprintf(
					"%v ∈ [BlockNumber-256..BlockNumber-1]",
					variable,
				))
			} else {
				clauses = append(clauses, fmt.Sprintf(
					"%v ∉ [BlockNumber-256..BlockNumber-1]",
					variable,
				))
			}
		}
	}

	if b.valueConstraint != nil {
		for variable, value := range b.valueConstraint {
			op := "+"
			if value > 0 {
				op = "-"
			} else {
				value = -value
			}
			clauses = append(clauses, fmt.Sprintf(
				"%v = BlockNumber%s%d",
				variable,
				op,
				value,
			))
		}
	}

	sort.Slice(clauses, func(i, j int) bool {
		return clauses[i] < clauses[j]
	})

	return strings.Join(clauses, " Λ ")
}

// RestricVariableToOneOfTheLast256Blocks adds a constraint on the variable
// so that it is generated with a value within the last 256 blocks.'
// If the generator is already marked as unsatisfiable or if the variable
// is already constrained to be outside the last 256 blocks, the generator
// remains unsatisfiable.
func (b *BlockContextGenerator) RestrictVariableToOneOfTheLast256Blocks(variable Variable) {
	if b.unsatisfiable {
		return
	}
	if b.rangeConstraints == nil {
		b.rangeConstraints = make(map[Variable]bool)
	}
	if inRange, ok := b.rangeConstraints[variable]; ok {
		if !inRange {
			b.markUnsatisfiable()
		}
	} else {
		b.rangeConstraints[variable] = true
	}
}

// RestrictVariableToNoneOfTheLast256Blocks adds a constraint on the variable
// so that it is generated with a value outside the last 256 blocks.'
// If the generator is already marked as unsatisfiable or if the variable
// is already constrained to be within the last 256 blocks, the generator
// remains unsatisfiable.
func (b *BlockContextGenerator) RestrictVariableToNoneOfTheLast256Blocks(variable Variable) {
	if b.unsatisfiable {
		return
	}
	if b.rangeConstraints == nil {
		b.rangeConstraints = make(map[Variable]bool)
	}
	if inRange, ok := b.rangeConstraints[variable]; ok {
		if inRange {
			b.markUnsatisfiable()
		}
	} else {
		b.rangeConstraints[variable] = false
	}
}

// SetBlockNumberOffsetValue adds a constraint on the variable so that it is
// generated with a value that is offset from the block number.
// If the generator is already marked as unsatisfiable or if the variable
// is already constrained to a different offset, the generator remains
// unsatisfiable.
func (b *BlockContextGenerator) SetBlockNumberOffsetValue(variable Variable, value int64) {
	if b.unsatisfiable {
		return
	}
	if b.valueConstraint == nil {
		b.valueConstraint = make(map[Variable]int64)
	}
	if existingValue, ok := b.valueConstraint[variable]; ok {
		if existingValue != value {
			b.markUnsatisfiable()
		}
	} else {
		b.valueConstraint[variable] = value
	}
}

// SetRevision adds a constraint on the State's revision.
func (b *BlockContextGenerator) SetRevision(revision Revision) {
	b.AddRevisionBounds(revision, revision)
}

// AddRevisionBounds adds a constraint on the State's revision.
func (b *BlockContextGenerator) AddRevisionBounds(lower, upper Revision) {
	if b.unsatisfiable {
		return
	}
	if lower > upper || lower < 0 || upper < 0 {
		b.markUnsatisfiable()
		return
	}
	min, _ := GetForkBlock(lower)
	max, _ := GetForkBlock(upper)
	len, _ := GetBlockRangeLengthFor(upper)
	if len == math.MaxUint64 {
		max = math.MaxUint64
	} else {
		max += len - 1
	}
	if b.blockNumberSolver == nil {
		b.blockNumberSolver = NewRangeSolver[uint64](min, max)
	} else {
		b.blockNumberSolver.AddLowerBoundary(min)
		b.blockNumberSolver.AddUpperBoundary(max)
	}
	if !b.blockNumberSolver.IsSatisfiable() {
		b.markUnsatisfiable()
	}
}

func (b *BlockContextGenerator) markUnsatisfiable() {
	b.unsatisfiable = true
	b.blockNumberSolver = nil
	b.rangeConstraints = nil
	b.valueConstraint = nil
}

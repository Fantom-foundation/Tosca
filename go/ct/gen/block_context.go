// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

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
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// BlockContextGenerator is a generator for block contexts.
// It can take constraints on the block number, which come from
// restricting the revision, and it can also take constraints on
// different variables in respect to the block number to be solved for.
// The constraints can be added in the form of fixed-value constraints
// on variables, and constraints on the range of values that variables
// can take.
// if conflicting constraints are added the generator will turn
// unsatisfiable, and return an error when trying to generate a block context.
type BlockContextGenerator struct {
	blockNumberSolver *RangeSolver[uint64]

	// This map is used to keep track of range constraints of variables.
	// Variables limited to recent block number range are mapped to
	// true, variables required to be not in the recent block number range
	// are mapped to false.
	rangeConstraints map[Variable]bool

	// This map is used to keep track of fixed-value constraints of variables.
	// the values we keep here are the offset from the block number, not the
	// actual value, because of this, the value can be negative as well,
	// meaning we could want to generate a value bigger than the block number.
	valueConstraint map[Variable]int64

	// This flag is set to true if the contained set of constraints
	// is not satisfiable.
	unsatisfiable bool
}

func NewBlockContextGenerator() *BlockContextGenerator {
	return &BlockContextGenerator{}
}

// generateBlockNumber generates a block number based on the constraints
// added to the generator. These could be revision constraints,
// fixed-value constraints, range constraints, or previously bound variables
// in the given assignment.
func (b *BlockContextGenerator) generateBlockNumber(assignment Assignment, rnd *rand.Rand) (uint64, error) {

	blockNumberSolver := NewIntervalSolver[uint64](0, math.MaxUint64)
	if b.blockNumberSolver != nil {
		blockNumberSolver.AddLowerBoundary(b.blockNumberSolver.min)
		blockNumberSolver.AddUpperBoundary(b.blockNumberSolver.max)
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
				max := uint64(math.MaxUint64)
				value := assignedValue.Uint64()
				if assignedValue.IsUint64() && value < max {
					min := assignedValue.Uint64() + 1
					if assignedValue.Uint64() < (max - 256) {
						max = assignedValue.Uint64() + 256
					}
					blockNumberSolver.Exclude(min, max)
				}
			}
		} else {
			if inRange {
				// if we have a condition for a variable in range, then we can not generate the first block number
				blockNumberSolver.AddLowerBoundary(1)
			}
		}
	}

	return blockNumberSolver.Generate(rnd)
}

// bindVariablesInValueConstraints processes the fixed-value constraints on variables.
func (b *BlockContextGenerator) bindVariablesInValueConstraints(assignment Assignment, resultingBlockNumber uint64) error {
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

// bindVariablesInRangeConstraints processes the range constraints on variables.
func (b *BlockContextGenerator) bindVariablesInRangeConstraints(resultingBlockNumber uint64, assignment Assignment, rnd *rand.Rand) error {
	lowerBound := uint64(0)
	if resultingBlockNumber > 256 {
		lowerBound = resultingBlockNumber - 256
	}
	upperBound := resultingBlockNumber
	blockNumberRangeGenerator := NewRangeSolver(lowerBound, upperBound-1)

	for variable, inRange := range b.rangeConstraints {
		if currentValue, isBound := assignment[variable]; isBound {
			if inRange != isInRange(lowerBound, currentValue, upperBound) {
				return ErrUnsatisfiable
			}
		} else { // no value bound to variable yet
			if inRange {
				blockNumber, err := blockNumberRangeGenerator.Generate(rnd)
				if err != nil {
					return err
				}
				assignment[variable] = NewU256(blockNumber)
			} else {
				numberOutOfRangeGenerator := NewIntervalSolver[uint64](0, math.MaxUint64)
				numberOutOfRangeGenerator.Exclude(resultingBlockNumber-256, resultingBlockNumber-1)
				number, err := numberOutOfRangeGenerator.Generate(rnd)
				if err != nil {
					return err
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
	blockNumber, err := b.generateBlockNumber(assignment, rnd)
	if err != nil {
		return st.BlockContext{}, err
	}

	// for all non bound relevant variables, assign them a value based on the constraints.
	// 1) fixed offset constraints
	if err := b.bindVariablesInValueConstraints(assignment, blockNumber); err != nil {
		return st.BlockContext{}, err
	}

	// 2) range constraints
	if err := b.bindVariablesInRangeConstraints(blockNumber, assignment, rnd); err != nil {
		return st.BlockContext{}, err
	}

	chainId := RandU256(rnd)
	coinbase := RandomAddress(rnd)

	baseFee := RandU256(rnd)
	blobBaseFee := RandU256(rnd)
	gasLimit := rnd.Uint64()
	gasPrice := RandU256(rnd)

	prevRandao := RandU256(rnd)

	revision := GetRevisionForBlock(blockNumber)
	time := GetForkTime(revision)
	nextTime := GetForkTime(revision + 1)
	timestamp := rnd.Uint64n(nextTime-time) + time

	return st.BlockContext{
		BaseFee:     baseFee,
		BlobBaseFee: blobBaseFee,
		BlockNumber: blockNumber,
		ChainID:     chainId,
		CoinBase:    coinbase,
		GasLimit:    gasLimit,
		GasPrice:    gasPrice,
		PrevRandao:  prevRandao,
		TimeStamp:   timestamp,
	}, nil
}

func (b *BlockContextGenerator) Clone() *BlockContextGenerator {
	if b.unsatisfiable {
		return &BlockContextGenerator{unsatisfiable: true}
	}
	var blockNumberSolverCopy *RangeSolver[uint64]
	if b.blockNumberSolver != nil {
		blockNumberSolverCopy = b.blockNumberSolver.Clone()
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
	} else {
		b.blockNumberSolver = nil
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
			clauses = append(clauses, b.blockNumberSolver.Print("BlockNumber"))
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

// RestrictVariableToOneOfTheLast256Blocks adds a constraint on the variable
// so that this generator assigns a value to it referencing one of the last 256 blocks.
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
// so that this generator assigns a value to it referencing none of the last 256 blocks.
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
// assigned with a value that is offseted from the block number.
func (b *BlockContextGenerator) SetBlockNumberOffsetValue(variable Variable, offset int64) {
	if b.unsatisfiable {
		return
	}
	if b.valueConstraint == nil {
		b.valueConstraint = make(map[Variable]int64)
	}
	if existingValue, ok := b.valueConstraint[variable]; ok {
		if existingValue != offset {
			b.markUnsatisfiable()
		}
	} else {
		b.valueConstraint[variable] = offset
	}
}

// SetRevision adds a constraint on the State's revision.
func (b *BlockContextGenerator) SetRevision(revision tosca.Revision) {
	b.AddRevisionBounds(revision, revision)
}

// AddRevisionBounds adds a constraint on the State's revision.
func (b *BlockContextGenerator) AddRevisionBounds(lower, upper tosca.Revision) {
	if b.unsatisfiable {
		return
	}
	if lower > upper || lower < 0 || upper < 0 {
		b.markUnsatisfiable()
		return
	}

	min := GetForkBlock(lower)
	max := GetForkBlock(upper)
	len, err := GetBlockRangeLengthFor(upper)
	if err != nil {
		b.markUnsatisfiable()
		return
	}
	if len >= math.MaxUint64-max {
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

func isInRange(lowerBound uint64, currentValue U256, upperBound uint64) bool {
	value := currentValue.Uint64()
	return currentValue.IsUint64() && lowerBound <= value && value <= upperBound
}

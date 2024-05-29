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
	"math"
	"sort"
	"strings"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// TODO: document me
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

// TODO: to be tested
func addOffset(value U256, offset int64) (result U256, hasOverflown bool) {
	if offset < 0 {
		result := value.Sub(common.NewU256(uint64(-offset)))
		return result, result.Gt(value)
	}
	result = value.Add(common.NewU256(uint64(offset)))
	return result, result.Lt(value)
}

func addWithOverflowCheck(a, b uint64) (result uint64, hasOverflown bool) {
	result = a + b
	return result, result < a
}

func (b *BlockContextGenerator) Generate(assignment Assignment, rnd *rand.Rand) (st.BlockContext, error) {
	if b.unsatisfiable {
		return st.BlockContext{}, ErrUnsatisfiable
	}

	var blockNumberSolver *RangeSolver[uint64]
	if b.blockNumberSolver != nil {
		blockNumberSolver = b.blockNumberSolver.Clone()
	} else {
		blockNumberSolver = NewRangeSolver[uint64](0, math.MaxUint64)
	}

	// apply constraints on block number derived from predefined assignments
	// 1) fixed offset constraints
	for variable, offset := range b.valueConstraint {
		if assignedValue, isBound := assignment[variable]; isBound {
			wantedBlock, overflow := addOffset(assignedValue, offset)
			if overflow || !wantedBlock.IsUint64() {
				return st.BlockContext{}, ErrUnsatisfiable
			}
			blockNumberSolver.AddEqualityConstraint(wantedBlock.Uint64())
		}
	}

	// 2) add constraints on block number derived from range constraints
	for variable, inRange := range b.rangeConstraints {
		if assignedValue, isBound := assignment[variable]; isBound {
			if inRange {
				if !assignedValue.IsUint64() {
					return st.BlockContext{}, ErrUnsatisfiable
				}
				value := assignedValue.Uint64()
				if lower, overflow := addWithOverflowCheck(value, 1); !overflow {
					blockNumberSolver.AddLowerBoundary(lower)
				} else {
					return st.BlockContext{}, ErrUnsatisfiable
				}
				if upper, overflow := addWithOverflowCheck(value, 256); !overflow {
					blockNumberSolver.AddUpperBoundary(upper)
				}
			} else {
				// 500 \notin [BN-256..BN-1]
				// needed: BN \in [0..500-1] || BN \in [500+256..math.MaxUint64]
				panic("not implemented")
			}
		}
	}

	resultingBlockNumber, err := blockNumberSolver.Generate(rnd)
	if err != nil {
		return st.BlockContext{}, err
	}

	// for all non bound relevant variables, assign them a value based on the constraints.
	// 1) fixed offset constraints
	for variable, offset := range b.valueConstraint {
		requiredValue, underflow := addOffset(common.NewU256(resultingBlockNumber), -offset)
		if underflow {
			return st.BlockContext{}, ErrUnsatisfiable
		}
		if currentValue, isBound := assignment[variable]; isBound && currentValue != requiredValue {
			return st.BlockContext{}, ErrUnsatisfiable
		} else if !isBound {
			assignment[variable] = requiredValue
		}
	}

	// 2) range constraints
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
				if !currentValue.IsUint64() {
					return st.BlockContext{}, ErrUnsatisfiable
				}
				if value < lowerBound || value >= upperBound {
					return st.BlockContext{}, ErrUnsatisfiable
				}
			} else {
				if currentValue.IsUint64() {
					if lowerBound <= value && value < upperBound {
						return st.BlockContext{}, ErrUnsatisfiable
					}
				}
			}
		} else { // no value bound to variable yet
			if inRange {
				blockNumber, err := blockNumberRangeGenerator.Generate(rnd)
				if err != nil {
					return st.BlockContext{}, err
				}
				assignment[variable] = common.NewU256(blockNumber)
			} else {
				number := rnd.Uint64n(math.MaxUint64 - 256)
				if number >= lowerBound {
					number += 256
				}
				assignment[variable] = common.NewU256(number)
			}
		}
	}

	return st.BlockContext{
		BlockNumber: resultingBlockNumber,
	}, nil

	/*
		baseFee := common.RandU256(rnd)
		blockNumberRangeSolver := NewRangeSolver(uint64(0), math.MaxUint64)

		revisionNumber, err := common.GetForkBlock(revision)
		if err != nil {
			return st.BlockContext{}, err
		}

		blockNumberRangeSolver.AddLowerBoundary(revisionNumber)
		revisionNumberLength, err := common.GetBlockRangeLengthFor(revision)
		if err != nil {
			return st.BlockContext{}, err
		}
		var blockNumber uint64
		if revisionNumberLength != 0 {
			blockNumberRangeSolver.AddUpperBoundary(revisionNumber + revisionNumberLength)
		}

		// if any relevant variable is already bound, then we need to constraint the block number to be generated.
		for _, offsetConstraint := range b.variablesOffsetConstraints {
			if assignedValue, isBound := assignment[offsetConstraint.variable]; isBound {

				if !assignedValue.IsUint64() {
					return st.BlockContext{}, fmt.Errorf("assigned value to variable %v is not a uint64", offsetConstraint.variable)
				}

				assignedValueI64 := int64(assignedValue.Uint64())
				lowerBound := assignedValueI64 - offsetConstraint.upperOffset + 2
				upperBound := assignedValueI64 + offsetConstraint.lowerOffset

				// if the range is only one number, then the assigned value must be within the range.
				isFixedValue := offsetConstraint.upperOffset-offsetConstraint.lowerOffset == 0
				isSameAsfixed := offsetConstraint.lowerOffset == assignedValueI64
				if isFixedValue && !isSameAsfixed {
					return st.BlockContext{}, ErrUnsatisfiable
				} else if isFixedValue && isSameAsfixed {
					blockNumberRangeSolver.AddEqualityConstraint(uint64(upperBound))
				}

				// in case of desired out of range and bound value too small.
				if lowerBound < 0 {
					lowerBound = 0
				}

				blockNumberRangeSolver.AddLowerBoundary(uint64(lowerBound))
				blockNumberRangeSolver.AddUpperBoundary(uint64(upperBound))

				// if the range solver is now unsatisfiable, it is most likely because the assigned value is out of the range
				// of the first number of the current revision - 256.
				if !blockNumberRangeSolver.IsSatisfiable() {
					return st.BlockContext{}, ErrUnsatisfiable
				}
			}
		}

		// generate block number
		blockNumber, err = blockNumberRangeSolver.Generate(rnd)
		if err != nil {
			return st.BlockContext{}, err
		}

		// for all non bound relevante variables, assign them a value based on the constraints.
		for _, offsetConstraint := range b.variablesOffsetConstraints {
			if _, isBound := assignment[offsetConstraint.variable]; !isBound {
				variableRangeSolver := NewRangeSolver[int64](math.MinInt64, math.MaxInt64)

				if difference := offsetConstraint.lowerOffset - offsetConstraint.upperOffset; difference == 0 {
					// if the difference between the two offsets is 0, then we can only generate a fix value.
					variableRangeSolver.AddEqualityConstraint(offsetConstraint.lowerOffset)
				} else if offsetConstraint.lowerOffset > 0 && offsetConstraint.upperOffset < 257 {
					// if lower offset is greater than 0 and upper is less than 257 we generate IN RANGE
					variableRangeSolver.AddLowerBoundary(offsetConstraint.upperOffset)
					variableRangeSolver.AddUpperBoundary(offsetConstraint.lowerOffset)
				} else {
					// else have to generate OUT OF RANGE
					// if blockNumber is too small, then we can ONLY generate OVER the current block number
					if blockNumber < 256 {
						variableRangeSolver.AddLowerBoundary(offsetConstraint.upperOffset + int64(blockNumber))
						variableRangeSolver.AddUpperBoundary(math.MaxInt64)

					} else {
						// if blockNumber is large enough, we can randomly generate under or over the range.
						// note that we can only generate under the range if the block number is larger than the upper offset.
						if rand.Intn(2) == 0 && blockNumber < 256 {
							// generate under the range
							variableRangeSolver.AddLowerBoundary(0)
							variableRangeSolver.AddUpperBoundary(int64(blockNumber))

						} else {
							// generate over the range
							variableRangeSolver.AddLowerBoundary(math.MinInt64)
							variableRangeSolver.AddUpperBoundary(0)
						}
					}
				}
				newValue, err := variableRangeSolver.Generate(rnd)
				if err != nil {
					return st.BlockContext{}, err
				}
				assignment[offsetConstraint.variable] = common.NewU256(uint64(int64(blockNumber) - newValue))
			} else {
				// if the variable is bound, then we need to check that the current constraint holds

				wantsInRange := func(c constraintPair) bool { return c.lowerOffset >= c.upperOffset }
				wantsFixValue := func(c constraintPair) bool { return c.lowerOffset == c.upperOffset }
				isValueInRange := func(value int64) bool { return 256 >= value && value >= 1 }
				if !assignment[offsetConstraint.variable].IsUint64() {
					return st.BlockContext{}, fmt.Errorf("assigned value to variable %v is not a uint64", offsetConstraint.variable)
				}
				boundedValueOffset := int64(blockNumber) - int64(assignment[offsetConstraint.variable].Uint64())

				// if we want fixed value and the fixed value is different
				if (wantsFixValue(offsetConstraint) && offsetConstraint.lowerOffset != boundedValueOffset) ||
					// or if we want the value to be in range but is it not
					(wantsInRange(offsetConstraint) && !isValueInRange(boundedValueOffset)) ||
					// or if we want it to be out of range and it is in range
					(!wantsInRange(offsetConstraint) && isValueInRange(boundedValueOffset)) {
					return st.BlockContext{}, ErrUnsatisfiable
				}

			}
		}

		chainId := common.RandU256(rnd)
		coinbase := common.RandomAddress(rnd)
		gasLimit := rnd.Uint64()
		gasPrice := common.RandU256(rnd)

		prevRandao := common.RandU256(rnd)
		timestamp := rnd.Uint64()

		return st.BlockContext{
			BaseFee:     baseFee,
			BlockNumber: blockNumber,
			ChainID:     chainId,
			CoinBase:    coinbase,
			GasLimit:    gasLimit,
			GasPrice:    gasPrice,
			PrevRandao:  prevRandao,
			TimeStamp:   timestamp,
		}, nil
	*/
	panic("not implemented")
}

func (b *BlockContextGenerator) Clone() *BlockContextGenerator {
	panic("not implemented")
	//return &BlockContextGenerator{variablesOffsetConstraints: slices.Clone(b.variablesOffsetConstraints)}
}

func (b *BlockContextGenerator) Restore(other *BlockContextGenerator) {
	panic("not implemented")
	/*
		if b == other {
			return
		}
		b.variablesOffsetConstraints = slices.Clone(other.variablesOffsetConstraints)
	*/
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

func (b *BlockContextGenerator) RestrictVariableToOneOfTheLast256Blocks(variable Variable) {
	if b.rangeConstraints == nil {
		b.rangeConstraints = make(map[Variable]bool)
	}
	if inRange, ok := b.rangeConstraints[variable]; ok {
		if !inRange {
			b.unsatisfiable = true
		}
	} else {
		b.rangeConstraints[variable] = true
	}
}

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

func (b *BlockContextGenerator) AddRevisionBounds(lower, upper Revision) {
	if b.unsatisfiable {
		return
	}
	min, _ := GetForkBlock(lower)
	max, _ := GetForkBlock(upper)
	len, _ := GetBlockRangeLengthFor(upper)
	max += len - 1
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

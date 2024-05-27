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
	"slices"

	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

// constraintPair represent a constraint on the variable, such that is in a range
// `(currentBlock - lower) < variable < (current block - upper)`
// or that the variable should be out of the range such as
// `variable < (currentBlock - lower) or variable > (currentBlock - upper)`
// or that the variable should be a fixed value
type constraintPair struct {
	variable    Variable
	lowerOffset int64
	upperOffset int64
}

type BlockContextGenerator struct {
	variablesOffsetConstraints []constraintPair
}

func NewBlockContextGenerator() *BlockContextGenerator {
	return &BlockContextGenerator{variablesOffsetConstraints: []constraintPair{}}
}

func (b *BlockContextGenerator) Generate(assignment Assignment, rnd *rand.Rand, revision common.Revision) (st.BlockContext, error) {
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
}

func (b *BlockContextGenerator) Clone() *BlockContextGenerator {
	return &BlockContextGenerator{variablesOffsetConstraints: slices.Clone(b.variablesOffsetConstraints)}
}

func (b *BlockContextGenerator) Restore(other *BlockContextGenerator) {
	if b == other {
		return
	}
	b.variablesOffsetConstraints = slices.Clone(other.variablesOffsetConstraints)
}

func (b *BlockContextGenerator) String() string {
	defineOperatorString := func(val int64) (int64, string) {
		if val >= 0 {
			return val, "-"
		} else {
			return -val, "+"
		}
	}

	var variablesOffsetConstraints string
	for _, v := range b.variablesOffsetConstraints {
		lower := v.lowerOffset
		upper := v.upperOffset
		lowerOffset, lowerOperator := defineOperatorString(lower)
		upperOffset, upperOperator := defineOperatorString(upper)
		variablesOffsetConstraints += fmt.Sprintf("[%v => BlockNumber %v %v Λ %v <= BlockNumber %v %v]", v.variable, upperOperator, upperOffset, v.variable, lowerOperator, lowerOffset) + " "
	}
	if len(variablesOffsetConstraints) != 0 {
		variablesOffsetConstraints = variablesOffsetConstraints[:len(variablesOffsetConstraints)-1]
	}

	return "{variablesOffsetConstraints: " + variablesOffsetConstraints + "}"
}

func (b *BlockContextGenerator) AddBlockNumberOffsetConstraintIn(variable Variable) {
	constraintIn := constraintPair{variable: variable, lowerOffset: 256, upperOffset: 1}
	if !slices.Contains(b.variablesOffsetConstraints, constraintIn) {
		b.variablesOffsetConstraints = append(b.variablesOffsetConstraints, constraintIn)
	}
}

func (b *BlockContextGenerator) AddBlockNumberOffsetConstraintOut(variable Variable) {
	constraintOut := constraintPair{variable: variable, lowerOffset: 0, upperOffset: 257}
	if !slices.Contains(b.variablesOffsetConstraints, constraintOut) {
		b.variablesOffsetConstraints = append(b.variablesOffsetConstraints, constraintOut)
	}
}

func (b *BlockContextGenerator) SetBlockNumberOffsetValue(variable Variable, value int64) {
	constraintValue := constraintPair{variable: variable, lowerOffset: value, upperOffset: value}
	if !slices.Contains(b.variablesOffsetConstraints, constraintValue) {
		b.variablesOffsetConstraints = append(b.variablesOffsetConstraints, constraintValue)
	}
}

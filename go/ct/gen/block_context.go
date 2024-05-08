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

type inequalityType int

const (
	offsetLessThan inequalityType = iota
	offsetGreaterThan
)

// variableInequality represents an inequality regarding the block number to be generated.
// it should be interpreted as
// `variable < (currentBlock - offset)` or `variable > (currentBlock - offset)`
type variableInequality struct {
	variable Variable
	kind     inequalityType
	offset   int64
}

// constraintPari is a pair of inequalities that represent a constraint that can either be that the variable is in a range such as
// `(currentBlock - ofset) < variable < current block`
// or that the variable should be out of the range such as
// `variable < (currentBlock - offset) or variable > currentBlock`
// it is assumed that both inequalities are regarding the same variable.
type constraintPair struct {
	lower variableInequality
	upper variableInequality
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
		if assignedValue, isBound := assignment[offsetConstraint.lower.variable]; isBound {

			assignedValueI64 := int64(assignedValue.Uint64())
			lowerBound := assignedValueI64 - offsetConstraint.upper.offset
			upperBound := assignedValueI64 + offsetConstraint.lower.offset - 1

			// if the range is only one number ( aka diff 2 ), then the assigned value must be within the range.
			diff := offsetConstraint.upper.offset - offsetConstraint.lower.offset
			isFixedValue := diff == 2 || diff == -2
			isSameAsfixed := offsetConstraint.lower.offset-1 == assignedValueI64
			if isFixedValue && !isSameAsfixed {
				return st.BlockContext{}, ErrUnsatisfiable
			} else if isFixedValue && isSameAsfixed {
				blockNumberRangeSolver.AddEqualityConstraint(uint64(upperBound - 1))
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

	variableRangeSolver := NewRangeSolver(math.MinInt64, int64(blockNumber))
	baseSolver := NewRangeSolver(math.MinInt64, int64(blockNumber))
	// for all non bound relevante variables, assign them a value based on the constraints.
	for i, offsetConstraint := range b.variablesOffsetConstraints {
		if _, isBound := assignment[offsetConstraint.lower.variable]; !isBound {
			variableRangeSolver.Restore(baseSolver)
			if difference := offsetConstraint.lower.offset - offsetConstraint.upper.offset; difference == 2 || difference == -2 {
				// if the difference between the two offsets is 2, then we can only generate a fix value.
				variableRangeSolver.AddEqualityConstraint(offsetConstraint.lower.offset - 1)
			} else if offsetConstraint.lower.offset > 1 && offsetConstraint.upper.offset < 256 {
				// if lower offset is greater than 1 and upper is less than 256 we generate in range.
				variableRangeSolver.AddLowerBoundary(offsetConstraint.upper.offset)
				variableRangeSolver.AddUpperBoundary(offsetConstraint.lower.offset)
			} else {
				// else have to generate out of range
				// if blockNumber is too small, then we can only generate over the range.
				if blockNumber < 256 {
					variableRangeSolver.AddLowerBoundary(int64(blockNumber))
					variableRangeSolver.AddUpperBoundary(math.MaxInt64)
				} else {
					// if blockNumber is large enough, we can generate under the range.
					variableRangeSolver.AddLowerBoundary(256)
					variableRangeSolver.AddUpperBoundary(int64(blockNumber) - 256)
				}
			}
			newValue, err := variableRangeSolver.Generate(rnd)
			if err != nil {
				return st.BlockContext{}, err
			}
			assignment[offsetConstraint.lower.variable] = common.NewU256(uint64(int64(blockNumber) - newValue))
		} else {
			// if the variable is bound, then we need to check that it does not clash with any other constraint.
			for _, previousConstraint := range b.variablesOffsetConstraints[:i] {
				if previousConstraint.lower.variable == offsetConstraint.lower.variable {
					if previousConstraint.lower.offset > offsetConstraint.upper.offset ||
						previousConstraint.upper.offset < offsetConstraint.lower.offset {
						return st.BlockContext{}, ErrUnsatisfiable
					}
				}
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
		lower := v.lower
		upper := v.upper
		lowerOffset, lowerOperator := defineOperatorString(lower.offset)
		upperOffset, upperOperator := defineOperatorString(upper.offset)
		variablesOffsetConstraints += fmt.Sprintf("[%v > BlockNumber %v %v Λ %v < BlockNumber %v %v]", upper.variable, upperOperator, upperOffset, lower.variable, lowerOperator, lowerOffset) + " "
	}
	if len(variablesOffsetConstraints) != 0 {
		variablesOffsetConstraints = variablesOffsetConstraints[:len(variablesOffsetConstraints)-1]
	}

	return "{variablesOffsetConstraints: " + variablesOffsetConstraints + "}"
}

func (b *BlockContextGenerator) AddBlockNumberOffsetConstraintIn(variable Variable) {
	offsetLessThan := variableInequality{variable, offsetLessThan, 257}
	offsetGreaterThan := variableInequality{variable, offsetGreaterThan, 0}
	constraintIn := constraintPair{offsetLessThan, offsetGreaterThan}
	if !slices.Contains(b.variablesOffsetConstraints, constraintIn) {
		b.variablesOffsetConstraints = append(b.variablesOffsetConstraints, constraintIn)
	}
}

func (b *BlockContextGenerator) AddBlockNumberOffsetConstraintOut(variable Variable) {
	offsetLessThan := variableInequality{variable, offsetLessThan, 1}
	offsetGreaterThan := variableInequality{variable, offsetGreaterThan, 256}
	constraintOut := constraintPair{offsetLessThan, offsetGreaterThan}
	if !slices.Contains(b.variablesOffsetConstraints, constraintOut) {
		b.variablesOffsetConstraints = append(b.variablesOffsetConstraints, constraintOut)
	}
}

func (b *BlockContextGenerator) SetBlockNumberOffsetValue(variable Variable, value int64) {
	offsetLessThan := variableInequality{variable, offsetLessThan, value + 1}
	offsetGreaterThan := variableInequality{variable, offsetGreaterThan, value - 1}
	constraintValue := constraintPair{offsetLessThan, offsetGreaterThan}
	if !slices.Contains(b.variablesOffsetConstraints, constraintValue) {
		b.variablesOffsetConstraints = append(b.variablesOffsetConstraints, constraintValue)
	}
}

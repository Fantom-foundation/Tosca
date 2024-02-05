package gen

import (
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type BlockContextGenerator struct {
}

func NewBlockContextGenerator() *BlockContextGenerator {
	return &BlockContextGenerator{}
}

func (*BlockContextGenerator) Generate(rnd *rand.Rand, revision common.Revision) (st.BlockContext, error) {

	baseFee := common.RandU256(rnd)

	revisionNumber, err := common.GetForkBlock(revision)
	if err != nil {
		return st.NewBlockContext(), err
	}
	revisionNumberRange, err := common.GetBlockRangeLengthFor(revision)
	if err != nil {
		return st.NewBlockContext(), err
	}
	var randomOffset uint64
	if revisionNumberRange != 0 {
		randomOffset = rnd.Uint64n(revisionNumberRange)
	} else {
		randomOffset = rnd.Uint64()
	}
	blockNumber := revisionNumber + randomOffset

	chainId := common.RandU256(rnd)
	coinbase, err := common.RandAddress(rnd)
	if err != nil {
		return st.NewBlockContext(), err
	}
	gasLimit := rnd.Uint64()
	gasPrice := common.RandU256(rnd)

	difficulty := common.RandU256(rnd)
	timestamp := rnd.Uint64()

	newBC := st.NewBlockContext()
	newBC.BaseFee = baseFee
	newBC.BlockNumber = blockNumber
	newBC.ChainID = chainId
	newBC.CoinBase = coinbase
	newBC.GasLimit = gasLimit
	newBC.GasPrice = gasPrice
	newBC.Difficulty = difficulty
	newBC.TimeStamp = timestamp

	return newBC, nil
}

func (*BlockContextGenerator) Clone() *BlockContextGenerator {
	return &BlockContextGenerator{}
}

func (*BlockContextGenerator) Restore(*BlockContextGenerator) {
}

func (*BlockContextGenerator) String() string {
	return "{}"
}

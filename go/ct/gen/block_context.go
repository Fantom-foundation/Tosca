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

func (*BlockContextGenerator) Generate(rnd *rand.Rand) (st.BlockContext, error) {

	blockNumber := rnd.Uint64()
	coinbase, err := common.RandAddress(rnd)
	if err != nil {
		return nil, err
	}
	gasLimit := rnd.Uint64()
	gasPrice := common.RandU256(rnd)

	prevRandao := [32]byte{}
	_, err = rnd.Read(prevRandao[:])
	if err != nil {
		return nil, err
	}

	timestamp := rnd.Uint64()

	newBC := st.NewBlockContext()
	newBC.BlockNumber = blockNumber
	newBC.CoinBase = coinbase
	newBC.GasLimit = gasLimit
	newBC.GasPrice = gasPrice
	newBC.PrevRandao = prevRandao
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

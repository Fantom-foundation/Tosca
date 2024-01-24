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

	revisionNumber, err := common.GetForkBlock(revision)
	if err != nil {
		return st.NewBlockContext(), err
	}

	// if it's the last supported revision, the blockNumber range has no limit.
	// if it's not, we want to limit this range to the firt block number of next revision.
	revisionNumberRange := uint64(0)
	if revision != (common.R99_UnknownNextRevision - 1) {
		nextRevisionNumber, err := common.GetForkBlock(revision + 1)
		if err != nil {
			return st.NewBlockContext(), err
		}
		// since we know both numbers are positive, and nextRevisionNumber is bigger,
		// we can safely converet them to uint64
		revisionNumberRange = uint64(nextRevisionNumber - revisionNumber)
	} else {
		revisionNumberRange = rnd.Uint64()
	}

	blockNumber := uint64(revisionNumber) + rnd.Uint64n(revisionNumberRange)
	coinbase, err := common.RandAddress(rnd)
	if err != nil {
		return st.NewBlockContext(), err
	}
	gasLimit := rnd.Uint64()
	gasPrice := common.RandU256(rnd)

	prevRandao := [32]byte{}
	_, err = rnd.Read(prevRandao[:])
	if err != nil {
		return st.NewBlockContext(), err
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

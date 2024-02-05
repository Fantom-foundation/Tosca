package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// BlockContext holds the block environment information
type BlockContext struct {
	BaseFee     U256    // Base fee in wei
	BlockNumber uint64  // Block's number
	ChainID     U256    // Chain id of the network
	CoinBase    Address // Address of the block's benficiary
	GasLimit    uint64  // Block's gas limit
	GasPrice    U256    // Price of gas in current environment
	Difficulty  U256    // Current block's difficulty
	TimeStamp   uint64  // Block's timestamp in unix time in seconds
}

// NewBlockContext returns a newly created instance with all default values.
func NewBlockContext() BlockContext {
	return BlockContext{}
}

// Diff returns a list of differences between the two contexts
func (b *BlockContext) Diff(other *BlockContext) []string {
	ret := []string{}
	blockDifference := "Different block context "

	if !b.BaseFee.Eq(other.BaseFee) {
		ret = append(ret, blockDifference+fmt.Sprintf("base fee: %v vs %v\n", b.BaseFee, other.BaseFee))
	}

	if b.BlockNumber != other.BlockNumber {
		ret = append(ret, blockDifference+fmt.Sprintf("block number: %v vs %v\n", b.BlockNumber, other.BlockNumber))
	}

	if !b.ChainID.Eq(other.ChainID) {
		ret = append(ret, blockDifference+fmt.Sprintf("chain id: %v vs %v\n", b.ChainID, other.ChainID))
	}

	if b.CoinBase != other.CoinBase {
		ret = append(ret, blockDifference+fmt.Sprintf("coinbase address: %v vs. %v\n", b.CoinBase, other.CoinBase))
	}

	if b.GasLimit != other.GasLimit {
		ret = append(ret, blockDifference+fmt.Sprintf("gas limit: %v vs %v\n", b.GasLimit, other.GasLimit))
	}

	if !b.GasPrice.Eq(other.GasPrice) {
		ret = append(ret, blockDifference+fmt.Sprintf("gas price: %v vs %v\n", b.GasPrice, other.GasPrice))
	}

	if b.Difficulty != other.Difficulty {
		ret = append(ret, blockDifference+fmt.Sprintf("difficulty: %v vs %v\n", b.Difficulty, other.Difficulty))
	}

	if b.TimeStamp != other.TimeStamp {
		ret = append(ret, blockDifference+fmt.Sprintf("timestamp: %v vs %v\n", b.TimeStamp, other.TimeStamp))
	}

	return ret
}

func (b *BlockContext) String() string {
	return fmt.Sprintf(
		"Block Context:"+
			"\n\t    Base Fee: %v,"+
			"\n\t    Block Number: %v,"+
			"\n\t    ChainID: %v,"+
			"\n\t    CoinBase: %v,"+
			"\n\t    Gas Limit: %v,"+
			"\n\t    Gas Price: %v,"+
			"\n\t    Difficulty: %v,"+
			"\n\t    Timestamp: %v\n",
		b.BaseFee, b.BlockNumber, b.ChainID, b.CoinBase, b.GasLimit, b.GasPrice,
		b.Difficulty, b.TimeStamp)
}

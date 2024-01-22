package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// BlockContext holds the block environment information
type BlockContext struct {
	BlockNumber uint64   // Block's number
	CoinBase    Address  // Address of the block's benficiary
	GasLimit    uint64   // Block's gas limit
	GasPrice    U256     // Price of gas in current environment
	PrevRandao  [32]byte // Previous block's RANDAO mix
	TimeStamp   uint64   // Block's timestamp
}

// NewBlockContext returns a newly created instance with all default values.
func NewBlockContext() BlockContext {
	return BlockContext{}
}

// Diff returns a list of differences between the two contexts
func (b *BlockContext) Diff(other *BlockContext) []string {
	ret := []string{}
	blockDifference := "Different block context "
	if b.BlockNumber != other.BlockNumber {
		ret = append(ret, blockDifference+fmt.Sprintf("block number: %v vs %v", b.BlockNumber, other.BlockNumber))
	}

	if b.CoinBase != other.CoinBase {
		ret = append(ret, blockDifference+fmt.Sprintf("coinbase address: %v vs. %v", b.CoinBase, other.CoinBase))
	}

	if b.GasLimit != other.GasLimit {
		ret = append(ret, blockDifference+fmt.Sprintf("gas limit: %v vs %v", b.GasLimit, other.GasLimit))
	}

	if !b.GasPrice.Eq(other.GasPrice) {
		ret = append(ret, blockDifference+fmt.Sprintf("gas price: %v vs %v", b.GasPrice, other.GasPrice))
	}

	if b.PrevRandao != other.PrevRandao {
		ret = append(ret, blockDifference+fmt.Sprintf("prev randao mix: %v vs %v", b.PrevRandao, other.PrevRandao))
	}

	if b.TimeStamp != other.TimeStamp {
		ret = append(ret, blockDifference+fmt.Sprintf("timestamp: %v vs %v", b.TimeStamp, other.TimeStamp))
	}

	return ret
}

func (b *BlockContext) String() string {
	return fmt.Sprintf("Block Context: ( Block Number: %v, CoinBase: %v,"+
		" Gas Limit: %v, Gas Price: %v, Prev Randao: %v, Timestamp: %v)",
		b.BlockNumber, b.CoinBase, b.GasLimit, b.GasPrice, b.PrevRandao, b.TimeStamp)
}

package st

import (
	"fmt"
	"time"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// BlockContext holds the block environment information
type BlockContext struct {
	BlockNumber int       // Block's number
	CoinBase    Address   // Address of the block's benficiary
	GasLimit    U256      // Block's gas limit
	GasPrice    U256      // Price of gas in current environment
	PrevRandao  [32]byte  // Previous block's RANDAO mix
	TimeStamp   time.Time // Block's timestamp, it should be returned in format.
}

// NewBlockContext returns a newly created instance with all default values.
func NewBlockContext() *BlockContext {
	return &BlockContext{}
}

// Clone creates an independent copy of the block context
func (b *BlockContext) Clone() *BlockContext {
	ret := NewBlockContext()
	ret.BlockNumber = b.BlockNumber
	ret.CoinBase = b.CoinBase
	ret.GasLimit = b.GasLimit
	ret.GasPrice = b.GasPrice
	ret.PrevRandao = b.PrevRandao
	ret.TimeStamp = b.TimeStamp
	return ret
}

// Eq compares all fiels of the block context
func (b *BlockContext) Eq(other *BlockContext) bool {
	return b.BlockNumber == other.BlockNumber &&
		b.CoinBase == other.CoinBase &&
		b.GasLimit.Eq(other.GasLimit) &&
		b.GasPrice.Eq(other.GasPrice) &&
		b.PrevRandao == other.PrevRandao &&
		b.TimeStamp == other.TimeStamp
}

// Diff returns a list of differences between the two contexts
func (b *BlockContext) Diff(other *BlockContext) []string {
	ret := []string{}

	if b.BlockNumber != other.BlockNumber {
		ret = append(ret, fmt.Sprintf("Different block number: %v vs %v", b.BlockNumber, other.BlockNumber))
	}

	if b.CoinBase != other.CoinBase {
		str := "Different coinbase address: "
		for _, dif := range b.CoinBase.Diff(other.CoinBase) {
			str += dif
		}
		ret = append(ret, str)
	}

	if !b.GasLimit.Eq(other.GasLimit) {
		ret = append(ret, fmt.Sprintf("Different gas limit: %v vs %v", b.GasLimit, other.GasLimit))
	}

	if !b.GasPrice.Eq(other.GasPrice) {
		ret = append(ret, fmt.Sprintf("Different gas price: %v vs %v", b.GasPrice, other.GasPrice))
	}

	if b.PrevRandao != other.PrevRandao {
		ret = append(ret, fmt.Sprintf("Different prev randao mix: %v vs %v", b.PrevRandao, other.PrevRandao))
	}

	if b.TimeStamp != other.TimeStamp {
		ret = append(ret, fmt.Sprintf("Different timestamp: %v vs %v", b.TimeStamp, other.TimeStamp))
	}

	return ret
}

func (b *BlockContext) String() string {
	return fmt.Sprintf("Block Context: ( Block Number: %v, CoinBase: %v,"+
		" Gas Limit: %v, Gas Price: %v, Prev Randao: %v, Timestamp: %v)",
		b.BlockNumber, b.CoinBase, b.GasLimit, b.GasPrice, b.PrevRandao, b.TimeStamp)
}

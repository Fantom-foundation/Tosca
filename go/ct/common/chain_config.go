package common

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

func GetChainConfig(chainId *big.Int) *params.ChainConfig {
	istanbulBlock, err := GetForkBlock(R07_Istanbul)
	if err != nil {
		panic(fmt.Sprintf("could not get Istanbul fork block: %v", err))
	}
	berlinBlock, err := GetForkBlock(R09_Berlin)
	if err != nil {
		panic(fmt.Sprintf("could not get Berlin fork block: %v", err))
	}
	londonBlock, err := GetForkBlock(R10_London)
	if err != nil {
		panic(fmt.Sprintf("could not get London fork block: %v", err))
	}

	chainConfig := &params.ChainConfig{}
	chainConfig.ChainID = chainId
	chainConfig.ByzantiumBlock = big.NewInt(0)
	chainConfig.IstanbulBlock = big.NewInt(0).SetUint64(istanbulBlock)
	chainConfig.BerlinBlock = big.NewInt(0).SetUint64(berlinBlock)
	chainConfig.LondonBlock = big.NewInt(0).SetUint64(londonBlock)
	chainConfig.Ethash = new(params.EthashConfig)

	return chainConfig
}

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

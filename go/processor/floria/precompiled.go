// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package floria

import (
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/ethereum/go-ethereum/common"
	geth "github.com/ethereum/go-ethereum/core/vm"
)

func handlePrecompiled(revision tosca.Revision, input tosca.Data, address tosca.Address, gas tosca.Gas) (tosca.CallResult, bool) {
	contract, ok := precompiledContract(address, revision)
	if !ok {
		return tosca.CallResult{}, false
	}
	gasCost := contract.RequiredGas(input)
	if gas < tosca.Gas(gasCost) {
		return tosca.CallResult{}, true
	}
	gas -= tosca.Gas(gasCost)
	output, err := contract.Run(input)

	return tosca.CallResult{
		Success: err == nil, // precompiled contracts only return errors on invalid input
		Output:  output,
		GasLeft: gas,
	}, true
}

func precompiledContract(address tosca.Address, revision tosca.Revision) (geth.PrecompiledContract, bool) {
	var precompiles map[common.Address]geth.PrecompiledContract
	switch revision {
	case tosca.R13_Cancun:
		precompiles = geth.PrecompiledContractsCancun
	case tosca.R12_Shanghai, tosca.R11_Paris, tosca.R10_London, tosca.R09_Berlin:
		precompiles = geth.PrecompiledContractsBerlin
	default: // Istanbul is the oldest supported revision supported by Sonic
		precompiles = geth.PrecompiledContractsIstanbul
	}
	contract, ok := precompiles[common.Address(address)]
	return contract, ok
}

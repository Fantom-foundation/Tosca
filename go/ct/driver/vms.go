// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package main

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/ct"
	"github.com/Fantom-foundation/Tosca/go/interpreter/evmzero"
	"github.com/Fantom-foundation/Tosca/go/interpreter/geth"
	"github.com/Fantom-foundation/Tosca/go/interpreter/lfvm"
	"golang.org/x/exp/maps"
)

func getVm(evmIdentifier string) (ct.Evm, error) {

	var allowedEVMs = map[string]func() ct.Evm{
		"lfvm":    func() ct.Evm { return lfvm.NewConformanceTestingTarget() },
		"geth":    func() ct.Evm { return geth.NewConformanceTestingTarget() },
		"evmzero": func() ct.Evm { return evmzero.NewConformanceTestingTarget() },
	}

	if f, ok := allowedEVMs[evmIdentifier]; ok {
		return f(), nil
	}

	return nil, fmt.Errorf("invalid EVM identifier, use one of: %v", maps.Keys(allowedEVMs))
}

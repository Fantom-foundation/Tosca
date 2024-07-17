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

// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// Registers the long-form EVM as a possible interpreter implementation.
func init() {
	// TODO: split into release-version and experimental versions
	for _, si := range []string{"", "-si"} {
		for _, stats := range []string{"", "-stats"} {
			for _, shaCache := range []string{"", "-no-sha-cache"} {
				for _, logging := range []string{"", "-logging"} {

					vm, err := NewVm(Config{
						ConversionConfig: ConversionConfig{
							WithSuperInstructions: si == "-si",
						},
						WithStatistics: stats == "-stats",
						NoShaCache:     shaCache == "-no-sha-cache",
						Logging:        logging == "-logging",
					})
					name := "lfvm" + si + stats + shaCache + logging
					if err != nil {
						panic(fmt.Sprintf("failed to create %s: %v", name, err))
					}

					tosca.RegisterInterpreter(name, vm)
				}
			}
		}
	}
	vm, err := NewVm(Config{
		ConversionConfig: ConversionConfig{
			CacheSize: -1,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create no-code-cache instance: %v", err))
	}
	tosca.RegisterInterpreter("lfvm-no-code-cache", vm)
}

type Config struct {
	ConversionConfig
	WithStatistics bool
	NoShaCache     bool
	Logging        bool
}

type VM struct {
	config    Config
	converter *Converter
}

func NewVm(config Config) (*VM, error) {
	converter, err := NewConverter(config.ConversionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create converter: %v", err)
	}
	return &VM{config: config, converter: converter}, nil
}

// Defines the newest supported revision for this interpreter implementation
const newestSupportedRevision = tosca.R13_Cancun

func (v *VM) Run(params tosca.Parameters) (tosca.Result, error) {
	if params.Revision > newestSupportedRevision {
		return tosca.Result{}, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}

	converted := v.converter.Convert(
		params.Code,
		params.CodeHash,
	)

	return Run(params, converted, v.config.WithStatistics, v.config.NoShaCache, v.config.Logging)
}

func (e *VM) DumpProfile() {
	if e.config.WithStatistics {
		printCollectedInstructionStatistics()
	}
}

func (e *VM) ResetProfile() {
	if e.config.WithStatistics {
		resetCollectedInstructionStatistics()
	}
}

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

	configs := map[string]Config{
		// This is the officially supported LFVM interpreter configuration to be
		// used for production purposes.
		"lfvm": {
			ConversionConfig: ConversionConfig{
				WithSuperInstructions: false,
			},
			WithShaCache: true,
		},

		// This is an unofficial LFVM interpreter configuration that uses super
		// instructions. It is currently exported by default since Aida's nightly
		// tests are depending on it and Aida is not yet importing experimental
		// configurations explicitly. Once Aida has been updated to import
		// experimental configurations explicitly, this configuration should be
		// removed from the default exports.
		//
		// TODO(#763): remove once Aida has been updated to import experimental
		// configurations explicitly.
		"lfvm-si": {
			ConversionConfig: ConversionConfig{
				WithSuperInstructions: true,
			},
			WithShaCache: true,
		},
	}

	for name, config := range configs {
		vm, err := NewVm(config)
		if err != nil {
			panic(fmt.Sprintf("failed to create %s: %v", name, err))
		}
		tosca.RegisterInterpreter(name, vm)
	}
}

// RegisterExperimentalInterpreterConfigurations registers all experimental
// LFVM interpreter configurations to Tosca's interpreter registry. This
// function should not be called in production code, as the resulting VMs are
// not officially supported.
func RegisterExperimentalInterpreterConfigurations() {
	for _, si := range []string{"", "-si"} {
		for _, shaCache := range []string{"", "-no-sha-cache"} {
			for _, mode := range []string{"", "-stats", "-logging"} {

				config := Config{
					ConversionConfig: ConversionConfig{
						WithSuperInstructions: si == "-si",
					},
					WithShaCache: shaCache != "-no-sha-cache",
				}

				if mode == "-stats" {
					config.runner = &statisticRunner{
						stats: newStatistics(),
					}
				} else if mode == "-logging" {
					config.runner = loggingRunner{}
				}

				vm, err := NewVm(config)
				name := "lfvm" + si + shaCache + mode
				if err != nil {
					panic(fmt.Sprintf("failed to create %s: %v", name, err))
				}

				if name != "lfvm" && name != "lfvm-si" {
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
	WithShaCache bool
	runner       runner
}

type lfvm struct {
	config    Config
	converter *Converter
}

func NewVm(config Config) (*lfvm, error) {
	converter, err := NewConverter(config.ConversionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create converter: %v", err)
	}
	return &lfvm{config: config, converter: converter}, nil
}

// Defines the newest supported revision for this interpreter implementation
const newestSupportedRevision = tosca.R13_Cancun

func (v *lfvm) Run(params tosca.Parameters) (tosca.Result, error) {
	if params.Revision > newestSupportedRevision {
		return tosca.Result{}, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}

	converted := v.converter.Convert(
		params.Code,
		params.CodeHash,
	)

	config := interpreterConfig{
		withShaCache: v.config.WithShaCache,
		runner:       v.config.runner,
	}

	return run(config, params, converted)
}

func (e *lfvm) DumpProfile() {
	if statsRunner, ok := e.config.runner.(*statisticRunner); ok {
		fmt.Print(statsRunner.getSummary())
	}
}

func (e *lfvm) ResetProfile() {
	if statsRunner, ok := e.config.runner.(*statisticRunner); ok {
		statsRunner.reset()
	}
}

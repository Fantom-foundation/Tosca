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
	"os"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// Config provides a set of user-definable options for the LFVM interpreter.
type Config struct {
}

// NewInterpreter creates a new LFVM interpreter instance with the official
// configuration for production purposes.
func NewInterpreter(Config) (*lfvm, error) {
	return newVm(config{
		ConversionConfig: ConversionConfig{
			WithSuperInstructions: false,
		},
		WithShaCache: true,
	})
}

// Registers the long-form EVM as a possible interpreter implementation.
func init() {
	tosca.MustRegisterInterpreterFactory("lfvm", func(any) (tosca.Interpreter, error) {
		return NewInterpreter(Config{})
	})
}

// RegisterExperimentalInterpreterConfigurations registers all experimental
// LFVM interpreter configurations to Tosca's interpreter registry. This
// function should not be called in production code, as the resulting VMs are
// not officially supported.
func RegisterExperimentalInterpreterConfigurations() error {

	configs := map[string]config{}

	for _, si := range []string{"", "-si"} {
		for _, shaCache := range []string{"", "-no-sha-cache"} {
			for _, mode := range []string{"", "-stats", "-logging"} {

				config := config{
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
					config.runner = loggingRunner{
						log: os.Stdout,
					}
				}

				name := "lfvm" + si + shaCache + mode
				if name == "lfvm" {
					continue
				}

				configs[name] = config
			}
		}
	}

	configs["lfvm-no-code-cache"] = config{
		ConversionConfig: ConversionConfig{CacheSize: -1},
	}

	for name, config := range configs {
		err := tosca.RegisterInterpreterFactory(
			name,
			func(any) (tosca.Interpreter, error) {
				return newVm(config)
			},
		)
		if err != nil {
			return fmt.Errorf("failed to register interpreter %q: %v", name, err)
		}
	}

	return nil
}

type config struct {
	ConversionConfig
	WithShaCache bool
	runner       runner
}

type lfvm struct {
	config    config
	converter *Converter
}

func newVm(config config) (*lfvm, error) {
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

	return run(v.config, params, converted)
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

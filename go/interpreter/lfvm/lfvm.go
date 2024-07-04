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

import "github.com/Fantom-foundation/Tosca/go/tosca"

// Registers the long-form EVM as a possible interpreter implementation.
func init() {
	tosca.RegisterInterpreter("lfvm", &VM{})
	tosca.RegisterInterpreter("lfvm-no-sha-cache", &VM{no_shaCache: true})
	tosca.RegisterInterpreter("lfvm-si", &VM{with_super_instructions: true})
	tosca.RegisterInterpreter("lfvm-si-no-sha-cache", &VM{with_super_instructions: true, no_shaCache: true})
	tosca.RegisterInterpreter("lfvm-stats", &VM{with_statistics: true})
	tosca.RegisterInterpreter("lfvm-si-stats", &VM{with_super_instructions: true, with_statistics: true})
	tosca.RegisterInterpreter("lfvm-no-code-cache", &VM{no_code_cache: true})
	tosca.RegisterInterpreter("lfvm-logging", &VM{logging: true})
}

type VM struct {
	with_super_instructions bool
	with_statistics         bool
	no_shaCache             bool
	no_code_cache           bool
	logging                 bool
}

// Defines the newest supported revision for this interpreter implementation
const newestSupportedRevision = tosca.R13_Cancun

func (v *VM) Run(params tosca.Parameters) (tosca.Result, error) {
	var codeHash tosca.Hash
	if params.CodeHash != nil {
		codeHash = *params.CodeHash
	}

	if params.Revision > newestSupportedRevision {
		return tosca.Result{}, &tosca.ErrUnsupportedRevision{Revision: params.Revision}
	}

	converted, err := Convert(
		params.Code,
		v.with_super_instructions,
		params.CodeHash == nil,
		v.no_code_cache,
		codeHash,
	)
	if err != nil {
		return tosca.Result{}, err
	}

	return Run(params, converted, v.with_statistics, v.no_shaCache, v.logging)
}

func (e *VM) DumpProfile() {
	if e.with_statistics {
		printCollectedInstructionStatistics()
	}
}

func (e *VM) ResetProfile() {
	if e.with_statistics {
		resetCollectedInstructionStatistics()
	}
}

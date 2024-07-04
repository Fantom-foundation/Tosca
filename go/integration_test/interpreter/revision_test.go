// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package interpreter_test

import (
	"errors"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"go.uber.org/mock/gomock"

	// This is only imported to get the EVM opcode definitions.
	// TODO: write up our own op-code definition and remove this dependency.
	geth "github.com/ethereum/go-ethereum/core/vm"
)

func TestUnsupportedRevision_KnownRevisions(t *testing.T) {
	knownRevisions := []tosca.Revision{tosca.Revision(0), tosca.Revision(1), tosca.Revision(2)}
	unknownRevisions := []tosca.Revision{tosca.Revision(90), tosca.Revision(91), tosca.Revision(92)}

	ctrl := gomock.NewController(t)
	mockStateDB := NewMockStateDB(ctrl)
	mockStateDB.EXPECT().GetStorage(gomock.Any(), gomock.Any()).AnyTimes().Return(tosca.Word{})
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(tosca.Value{})
	mockStateDB.EXPECT().GetCodeSize(gomock.Any()).AnyTimes().Return(0)
	mockStateDB.EXPECT().AccountExists(gomock.Any()).AnyTimes().Return(true)
	mockStateDB.EXPECT().GetCodeHash(gomock.Any()).AnyTimes().Return(tosca.Hash{})
	mockStateDB.EXPECT().GetBlockHash(gomock.Any()).AnyTimes().Return(tosca.Hash{})

	code := []byte{byte(geth.PUSH2), byte(5), byte(2), byte(geth.SUB)}

	for _, variant := range Variants {
		for _, revision := range knownRevisions {
			evm := TestEVM{
				interpreter: tosca.GetInterpreter(variant),
				revision:    revision,
				state:       mockStateDB,
			}
			_, err := evm.Run(code, []byte{})
			if err != nil {
				t.Errorf("unexpected error during evm run")
			}
		}

		for _, revision := range unknownRevisions {
			evm := TestEVM{
				interpreter: tosca.GetInterpreter(variant),
				revision:    revision,
				state:       mockStateDB,
			}
			_, err := evm.Run(code, []byte{})
			targetError := &tosca.ErrUnsupportedRevision{}
			if !errors.As(err, &targetError) {
				t.Errorf("Running on %s: expected unsupported revision error but got %v", variant, err)
			}
		}
	}
}

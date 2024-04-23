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

package vm_test

import (
	"errors"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"go.uber.org/mock/gomock"

	// This is only imported to get the EVM opcode definitions.
	// TODO: write up our own op-code definition and remove this dependency.
	geth "github.com/ethereum/go-ethereum/core/vm"
)

func TestUnsupportedRevision_KnownRevisions(t *testing.T) {
	knownRevisions := []vm.Revision{vm.Revision(0), vm.Revision(1), vm.Revision(2)}
	unknownRevisions := []vm.Revision{vm.Revision(90), vm.Revision(91), vm.Revision(92)}

	ctrl := gomock.NewController(t)
	mockStateDB := NewMockStateDB(ctrl)
	mockStateDB.EXPECT().GetStorage(gomock.Any(), gomock.Any()).AnyTimes().Return(vm.Word{})
	mockStateDB.EXPECT().GetBalance(gomock.Any()).AnyTimes().Return(vm.Value{})
	mockStateDB.EXPECT().GetCodeSize(gomock.Any()).AnyTimes().Return(0)
	mockStateDB.EXPECT().AccountExists(gomock.Any()).AnyTimes().Return(true)
	mockStateDB.EXPECT().GetCodeHash(gomock.Any()).AnyTimes().Return(vm.Hash{})
	mockStateDB.EXPECT().GetBlockHash(gomock.Any()).AnyTimes().Return(vm.Hash{})

	code := []byte{byte(geth.PUSH2), byte(5), byte(2), byte(geth.SUB)}

	for _, variant := range Variants {
		for _, revision := range knownRevisions {
			evm := TestEVM{
				interpreter: vm.GetInterpreter(variant),
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
				interpreter: vm.GetInterpreter(variant),
				revision:    revision,
				state:       mockStateDB,
			}
			_, err := evm.Run(code, []byte{})
			targetError := &vm.ErrUnsupportedRevision{}
			if !errors.As(err, &targetError) {
				t.Errorf("Running on %s: expected unsupported revision error but got %v", variant, err)
			}
		}
	}
}

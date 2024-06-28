// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vm_test

import "github.com/Fantom-foundation/Tosca/go/vm"

type WorldState map[vm.Address]Account

//var _ vm.WorldState = &WorldState{}

type Account struct {
	Nonce   uint64
	Balance vm.Value
	Code    []byte
	Storage map[vm.Key]vm.Word
}

func (ws *WorldState) AccountExists(addr vm.Address) bool {
	_, ok := (*ws)[addr]
	return ok
}

func (ws *WorldState) GetNonce(addr vm.Address) uint64 {
	return (*ws)[addr].Nonce
}

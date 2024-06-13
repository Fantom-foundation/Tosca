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

package gen

import (
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type TransientGenerator struct {
}

func NewTransientGenerator() *TransientGenerator {
	return &TransientGenerator{}
}

func (t *TransientGenerator) Generate(rnd *rand.Rand) (*st.Transient, error) {
	transient := &st.Transient{}

	// Some entries with keys returned by the parameter samples function
	transient.SetStorage(common.NewU256(0), common.NewU256(1))
	transient.SetStorage(common.NewU256(1), common.NewU256(1))
	transient.SetStorage(common.NewU256(1<<8), common.NewU256(1<<8))
	transient.SetStorage(common.NewU256(1<<16), common.NewU256(1<<16))
	transient.SetStorage(common.NewU256(1<<32), common.NewU256(1<<32))

	// Random entries
	for i := 0; i < rnd.Intn(42); i++ {
		key := common.RandU256(rnd)
		value := common.RandU256(rnd)

		transient.SetStorage(key, value)
	}

	return transient, nil
}

func (t *TransientGenerator) Clone() *TransientGenerator {
	return &TransientGenerator{}
}

func (*TransientGenerator) Restore(*TransientGenerator) {
}

func (t *TransientGenerator) String() string {
	return "{}"
}

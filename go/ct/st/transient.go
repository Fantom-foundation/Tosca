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

package st

import (
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"golang.org/x/exp/maps"
)

type Transient struct {
	storage map[U256]U256
}

func (t *Transient) SetStorage(key U256, value U256) {
	if t.storage == nil {
		t.storage = make(map[U256]U256)
		t.storage[key] = value
	} else {
		t.storage[key] = value
	}
}

func (t *Transient) GetStorage(key U256) U256 {
	return t.storage[key]
}

func (t *Transient) DeleteStorage(key U256) {
	delete(t.storage, key)
}

func (t *Transient) Clone() *Transient {
	return &Transient{maps.Clone(t.storage)}
}

func (t *Transient) Eq(other *Transient) bool {
	return mapEqualIgnoringZeroValues(t.storage, other.storage)
}

func (t *Transient) Diff(other *Transient) (res []string) {
	res = append(res, mapDiffIgnoringZeroValues(t.storage, other.storage, "storage")...)
	return
}

// For testing purposes only
func (t *Transient) GetStorageKeys() []U256 {
	return maps.Keys(t.storage)
}

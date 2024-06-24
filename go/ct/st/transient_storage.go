// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	. "github.com/Fantom-foundation/Tosca/go/ct/common"
	"golang.org/x/exp/maps"
)

type TransientStorage struct {
	storage map[U256]U256
}

func (t *TransientStorage) Set(key U256, value U256) {
	if value.IsZero() {
		delete(t.storage, key)
		return
	}

	if t.storage == nil {
		t.storage = make(map[U256]U256)
	}
	t.storage[key] = value
}

func (t *TransientStorage) Get(key U256) U256 {
	return t.storage[key]
}

func (t *TransientStorage) IsZero(key U256) bool {
	return t.storage[key].IsZero()
}

func (t *TransientStorage) Clone() *TransientStorage {
	return &TransientStorage{maps.Clone(t.storage)}
}

func (t *TransientStorage) Eq(other *TransientStorage) bool {
	return mapEqualIgnoringZeroValues(t.storage, other.storage)
}

func (t *TransientStorage) Diff(other *TransientStorage) (res []string) {
	return mapDiffIgnoringZeroValues(t.storage, other.storage, "transient storage")
}

// IsAllZero is used for testing purposes only
func (t *TransientStorage) IsAllZero() bool {
	return len(t.storage) == 0
}

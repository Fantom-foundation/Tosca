// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package common

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
	"pgregory.net/rand"
)

func TestHash_GetRandomHash(t *testing.T) {
	rnd := rand.New()
	hashes := []tosca.Hash{}
	for i := 0; i < 10; i++ {
		hashes = append(hashes, GetRandomHash(rnd))
		for j := 0; j < i; j++ {
			if hashes[i] == hashes[j] {
				t.Errorf("random hashes are not random, got %v and %v", hashes[i], hashes[j])
			}
		}
	}
}

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
	"testing"

	"pgregory.net/rand"
)

func TestTransient_GenerateNonEmptyStorage(t *testing.T) {
	rnd := rand.New(0)
	generator := NewTransientGenerator()
	transient, err := generator.Generate(rnd)
	if err != nil {
		t.Fatalf("failed to generate transient storage, err: %v", err)
	}
	if transient.GetStorageKeys() == nil {
		t.Error("generated transient storage is empty")
	}
	if len(transient.GetStorageKeys()) == 0 {
		t.Error("generated transient storage is empty")
	}

}

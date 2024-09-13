// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package tosca

import "testing"

func TestInterpreterRegistry_NameCollisionsAreDetected(t *testing.T) {
	const name = "something-just-for-this-test"
	factory := func(any) (Interpreter, error) {
		return nil, nil
	}
	if err := RegisterInterpreterFactory(name, factory); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := RegisterInterpreterFactory(name, factory); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestInterpreterRegistry_NilFactoriesAreRejected(t *testing.T) {
	const name = "something"
	if err := RegisterInterpreterFactory(name, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

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

import (
	"errors"
	"testing"
)

func TestConstError_Error(t *testing.T) {
	// Define a constant error
	const myError = ConstError("this is a constant error")

	// Test the Error() method
	if myError.Error() != "this is a constant error" {
		t.Errorf("expected 'this is a constant error', got '%s'", myError.Error())
	}

	// tests error.Is
	if !errors.Is(myError, ConstError("this is a constant error")) {
		t.Errorf("expected true, got false")
	}
}

func TestConstError_Empty(t *testing.T) {
	// Define an empty constant error
	const emptyError ConstError = ""

	// Test the Error() method
	if emptyError.Error() != "" {
		t.Errorf("expected empty string, got '%s'", emptyError.Error())
	}
}

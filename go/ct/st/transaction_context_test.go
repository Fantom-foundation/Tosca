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
	"fmt"
	"strings"
	"testing"
)

func TestTransactionContext_Diff(t *testing.T) {
	tests := map[string]struct {
		change func(*TransactionContext)
	}{
		"Origin Address": {func(t *TransactionContext) { t.OriginAddress[0]++ }},
	}

	transactionContext := TransactionContext{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t2 := TransactionContext{}
			test.change(&t2)
			if diffs := transactionContext.Diff(&t2); len(diffs) == 0 {
				t.Errorf("No difference found in modified %v", name)
			}
		})
	}
}

func TestTransactionContext_String(t *testing.T) {
	tests := map[string]struct {
		change func(*TransactionContext) any
	}{
		"Origin Address": {func(t *TransactionContext) any {
			t.OriginAddress[19] = 0xfe
			return t.OriginAddress
		}},
	}

	c := TransactionContext{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			v := test.change(&c)
			str := c.String()
			if !strings.Contains(str, fmt.Sprintf("%v: %v", name, v)) {
				t.Errorf("Did not find %v string", name)
			}
		})
	}
}

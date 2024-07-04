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
	"reflect"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestTransactionContext_Diff(t *testing.T) {
	tests := map[string]struct {
		change func(*TransactionContext)
	}{
		"Origin Address": {func(t *TransactionContext) { t.OriginAddress[0]++ }},
		"blobHashes":     {func(t *TransactionContext) { t.BlobHashes = []tosca.Hash{{1}} }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			transactionContext := TransactionContext{}
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
		"Blob Hashes": {func(t *TransactionContext) any {
			t.BlobHashes = []tosca.Hash{{1, 2, 3, 4}}
			return t.BlobHashes
		}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := TransactionContext{}
			v := test.change(&c)
			got := c.String()
			want := fmt.Sprintf("%v: %v", name, v)
			if !strings.Contains(got, want) {
				t.Errorf("Did not find %v string", name)
			}
		})
	}
}

func TestTransactionContext_Clone(t *testing.T) {

	t1 := TransactionContext{
		OriginAddress: tosca.Address{1, 2, 3, 4},
		BlobHashes:    []tosca.Hash{{1, 2, 3, 4}},
	}

	t2 := t1.Clone()

	if !reflect.DeepEqual(t1.BlobHashes, t2.BlobHashes) || !reflect.DeepEqual(t1.OriginAddress, t2.OriginAddress) {
		t.Errorf("Cloned transaction context is not equal to original")
	}

	t2.OriginAddress[0] = 0xff
	t2.BlobHashes[0] = tosca.Hash{0x00}

	if reflect.DeepEqual(t1.BlobHashes, t2.BlobHashes) || reflect.DeepEqual(t1.OriginAddress, t2.OriginAddress) {
		t.Errorf("Cloned transaction context is not independent to original")
	}
}

func TestTransactionContext_Equal(t *testing.T) {

	tests := map[string]struct {
		a, b   *TransactionContext
		wanted bool
	}{
		"match": {
			a:      &TransactionContext{},
			b:      &TransactionContext{},
			wanted: true,
		},
		"mismatch": {
			a:      &TransactionContext{BlobHashes: []tosca.Hash{{1}}},
			b:      &TransactionContext{BlobHashes: []tosca.Hash{{4}}},
			wanted: false,
		},
		"mismatch-different-blobhash-length": {
			a:      &TransactionContext{BlobHashes: []tosca.Hash{{1}, {2}}},
			b:      &TransactionContext{BlobHashes: []tosca.Hash{{4}}},
			wanted: false},
		"mismatch-different-origin-address": {
			a:      &TransactionContext{OriginAddress: tosca.Address{1, 2, 3, 4}},
			b:      &TransactionContext{OriginAddress: tosca.Address{4, 3, 2, 1}},
			wanted: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			t1 := test.a
			t2 := test.b

			if t1.Eq(t2) != test.wanted {
				t.Error("transaction context equal does not behave as expected")
			}
		})
	}
}

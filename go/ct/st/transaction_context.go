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
	"slices"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

// TransactionContext holds all transaction data
type TransactionContext struct {
	OriginAddress tosca.Address // Address of execution origination
	BlobHashes    []tosca.Hash  // List of blob hashes
}

func NewTransactionContext() *TransactionContext {
	return &TransactionContext{}
}

func (t *TransactionContext) Clone() *TransactionContext {
	return &TransactionContext{
		OriginAddress: t.OriginAddress,
		BlobHashes:    slices.Clone(t.BlobHashes),
	}
}

func (t *TransactionContext) Eq(other *TransactionContext) bool {
	isEqual := true

	isEqual = isEqual && (t.OriginAddress == other.OriginAddress)
	isEqual = isEqual && slices.Equal(t.BlobHashes, other.BlobHashes)

	return isEqual
}

// Diff returns a list of differences between the two transaction contexts.
func (t *TransactionContext) Diff(other *TransactionContext) []string {
	ret := []string{}
	transactionContextDiff := "Different transaction context "

	if t.OriginAddress != other.OriginAddress {
		ret = append(ret, transactionContextDiff+fmt.Sprintf("origin address: %v vs. %v\n", t.OriginAddress,
			other.OriginAddress))
	}

	if len(t.BlobHashes) != len(other.BlobHashes) {
		ret = append(ret, transactionContextDiff+fmt.Sprintf("blob hashes: %v vs %v\n", t.BlobHashes, other.BlobHashes))
	} else {
		for i, hash := range t.BlobHashes {
			if hash != other.BlobHashes[i] {
				ret = append(ret, transactionContextDiff+fmt.Sprintf("blob hash at location %v: %v vs %v\n", i, hash, other.BlobHashes[i]))
			}
		}
	}

	return ret
}

func (c *TransactionContext) String() string {
	const maxHashesToPrint = 5

	BlobHashString := "Blob Hashes: "
	if len(c.BlobHashes) > maxHashesToPrint {
		BlobHashString += fmt.Sprintf("%v, ...", c.BlobHashes[:maxHashesToPrint])
	} else {
		BlobHashString += fmt.Sprintf("%v", c.BlobHashes)
	}

	return fmt.Sprintf(
		"Transaction Context:"+
			"\n\t    Origin Address: %v\n"+
			"\n\t    %v\n",
		c.OriginAddress, BlobHashString)
}

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

	"github.com/Fantom-foundation/Tosca/go/vm"
)

// TransactionContext holds all transaction data
type TransactionContext struct {
	OriginAddress vm.Address // Address of execution origination
}

// Diff returns a list of differences between the two transaction contexts.
func (c *TransactionContext) Diff(other *TransactionContext) []string {
	ret := []string{}
	transactionContextDiff := "Different transaction context "

	if c.OriginAddress != other.OriginAddress {
		ret = append(ret, transactionContextDiff+fmt.Sprintf("origin address: %v vs. %v\n", c.OriginAddress,
			other.OriginAddress))
	}

	return ret
}

func (c *TransactionContext) String() string {
	return fmt.Sprintf(
		"Transaction Context:\n\t    Origin Address: %v\n",
		c.OriginAddress)
}

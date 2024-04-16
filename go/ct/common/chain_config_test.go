//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package common

import (
	"math/big"
	"testing"
)

func TestChainConfig(t *testing.T) {
	chainConfig := GetChainConfig(big.NewInt(7))

	if want, got := big.NewInt(7), chainConfig.ChainID; want.Cmp(got) != 0 {
		t.Errorf("Unexpected chain id. wanted: %v, got %v", want, got)
	}
}

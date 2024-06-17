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

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestBlockContext_Diff(t *testing.T) {
	tests := map[string]struct {
		change func(*BlockContext)
	}{
		"basefee":     {func(b *BlockContext) { b.BaseFee = NewU256(1) }},
		"blobBaseFee": {func(b *BlockContext) { b.BlobBaseFee = NewU256(1) }},
		"blockNumber": {func(b *BlockContext) { b.BlockNumber++ }},
		"chainid":     {func(b *BlockContext) { b.ChainID = NewU256(1) }},
		"coinbase":    {func(b *BlockContext) { b.CoinBase[0]++ }},
		"gasLimit":    {func(b *BlockContext) { b.GasLimit++ }},
		"gasPrice":    {func(b *BlockContext) { b.GasPrice = NewU256(1) }},
		"prevRandao":  {func(b *BlockContext) { b.PrevRandao = NewU256(1) }},
		"timestamp":   {func(b *BlockContext) { b.TimeStamp++ }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b1 := BlockContext{}
			b2 := BlockContext{}
			test.change(&b2)
			if diffs := b1.Diff(&b2); len(diffs) == 0 {
				t.Error("No difference found in modified context")
			}
		})
	}
}

func TestBlockContext_String(t *testing.T) {
	tests := map[string]struct {
		change func(*BlockContext) any
	}{
		"Base Fee":      {func(b *BlockContext) any { b.BaseFee = NewU256(1); return b.BaseFee }},
		"Blob Base Fee": {func(b *BlockContext) any { b.BlobBaseFee = NewU256(1); return b.BlobBaseFee }},
		"Block Number":  {func(b *BlockContext) any { b.BlockNumber++; return b.BlockNumber }},
		"ChainID":       {func(b *BlockContext) any { b.ChainID = NewU256(1); return b.ChainID }},
		"CoinBase":      {func(b *BlockContext) any { b.CoinBase[0]++; return b.CoinBase }},
		"Gas Limit":     {func(b *BlockContext) any { b.GasLimit++; return b.GasLimit }},
		"Gas Price":     {func(b *BlockContext) any { b.GasPrice = NewU256(1); return b.GasPrice }},
		"PrevRandao":    {func(b *BlockContext) any { b.PrevRandao = NewU256(1); return b.PrevRandao }},
		"Timestamp":     {func(b *BlockContext) any { b.TimeStamp++; return b.TimeStamp }},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b := BlockContext{}
			v := test.change(&b)
			str := b.String()
			want := fmt.Sprintf("%v: %v", name, v)
			if !strings.Contains(str, want) {
				t.Errorf("Did not find %v string", name)
			}
		})
	}
}

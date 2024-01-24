package st

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestBlockContext_NewBlockContext(t *testing.T) {
	tests := map[string]struct {
		equal func(*BlockContext) bool
	}{
		"blockNumber": {func(b *BlockContext) bool { want, got := uint64(0), b.BlockNumber; return want == got }},
		"coinbase":    {func(b *BlockContext) bool { want, got := (Address{}), b.CoinBase; return want == got }},
		"gasLimit":    {func(b *BlockContext) bool { want, got := uint64(0), b.GasLimit; return want == got }},
		"gasPrice":    {func(b *BlockContext) bool { want, got := NewU256(0), b.GasPrice; return want.Eq(got) }},
		"prevRandao":  {func(b *BlockContext) bool { want, got := [32]byte{}, b.PrevRandao; return want == got }},
		"timestamp":   {func(b *BlockContext) bool { want, got := (uint64(0)), b.TimeStamp; return want == got }},
	}

	blockContext := NewBlockContext()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if !test.equal(&blockContext) {
				t.Error("Unexpected value in new context")
			}
		})
	}
}

func TestBlockContext_Diff(t *testing.T) {
	tests := map[string]struct {
		change func(*BlockContext)
	}{
		"blockNumber": {func(b *BlockContext) { b.BlockNumber++ }},
		"coinbase":    {func(b *BlockContext) { b.CoinBase[0]++ }},
		"gasLimit":    {func(b *BlockContext) { b.GasLimit++ }},
		"gasPrice":    {func(b *BlockContext) { b.GasPrice = NewU256(1) }},
		"prevRandao":  {func(b *BlockContext) { b.PrevRandao[0]++ }},
		"timestamp":   {func(b *BlockContext) { b.TimeStamp++ }},
	}

	b1 := NewBlockContext()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b2 := NewBlockContext()
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
		"Block Number": {func(b *BlockContext) any { b.BlockNumber++; return b.BlockNumber }},
		"CoinBase":     {func(b *BlockContext) any { b.CoinBase[0]++; return b.CoinBase }},
		"Gas Limit":    {func(b *BlockContext) any { b.GasLimit++; return b.GasLimit }},
		"Gas Price":    {func(b *BlockContext) any { b.GasPrice = NewU256(1); return b.GasPrice }},
		"Prev Randao":  {func(b *BlockContext) any { b.PrevRandao[0]++; return b.PrevRandao }},
		"Timestamp":    {func(b *BlockContext) any { b.TimeStamp++; return b.TimeStamp }},
	}

	b := NewBlockContext()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			v := test.change(&b)
			str := b.String()
			want := fmt.Sprintf("%v: %v", name, v)
			if !strings.Contains(str, want) {
				t.Errorf("Did not find %v string", name)
			}
		})
	}
}

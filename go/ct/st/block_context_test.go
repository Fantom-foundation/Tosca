package st

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestBlockContext_NewBlockContext(t *testing.T) {
	blockContext := NewBlockContext()

	if want, got := uint64(0), blockContext.BlockNumber; want != got {
		t.Errorf("Unexpected block number, want %v, got %v", want, got)
	}

	if want, got := (Address{}), blockContext.CoinBase; want != got {
		t.Errorf("Unexpected codebase, want %v, got %v", want, got)
	}

	if want, got := NewU256(0), blockContext.GasLimit; !want.Eq(got) {
		t.Errorf("Unexpected gas limit, want %v, got %v", want, got)
	}

	if want, got := NewU256(0), blockContext.GasPrice; !want.Eq(got) {
		t.Errorf("Unexpected gas price, want %v, got %v", want, got)
	}

	if want, got := [32]byte{}, blockContext.PrevRandao; want != got {
		t.Errorf("Unexpected prev randao, want %v, got %v", want, got)
	}

	if want, got := (time.Time{}), blockContext.TimeStamp; want != got {
		t.Errorf("Unexpected timestamp, want %v, got %v", want, got)
	}

}

func TestBlockContext_Clone(t *testing.T) {
	b1 := NewBlockContext()
	b2 := b1.Clone()

	if !b1.Eq(b2) {
		t.Error("Clone is different from original")
	}

	b2.BlockNumber++
	b2.CoinBase[0] = 0xff
	b2.GasLimit = NewU256(1)
	b2.GasPrice = NewU256(1)
	b2.PrevRandao[0] = 0xff
	b2.TimeStamp = time.Now()

	if b1.BlockNumber == b2.BlockNumber ||
		b1.CoinBase == b2.CoinBase ||
		b1.GasLimit == b2.GasLimit ||
		b1.GasPrice.Eq(b2.GasPrice) ||
		b1.PrevRandao == b2.PrevRandao ||
		b1.TimeStamp == b2.TimeStamp {
		t.Error("Clone is not independent from original")
	}

}

func TestBlockContext_Eq(t *testing.T) {
	b1 := NewBlockContext()
	b2 := b1.Clone()

	if !b1.Eq(&b1) {
		t.Error("Self-comparison is broken")
	}

	if !b1.Eq(b2) {
		t.Error("Clones are not equal")
	}

	b2.BlockNumber++
	if b1.Eq(b2) {
		t.Error("Different block number is considered the same")
	}
	b2.BlockNumber--

	b2.CoinBase = Address{0xff}
	if b1.Eq(b2) {
		t.Error("Different coinbase is considered the same")
	}
	b2.CoinBase = Address{}

	b2.GasLimit = NewU256(1)
	if b1.Eq(b2) {
		t.Error("Different gas limit is considered the same")
	}
	b2.GasLimit = NewU256(0)

	b2.GasPrice = NewU256(1)
	if b1.Eq(b2) {
		t.Error("Different gas price is considered the same")
	}
	b2.GasPrice = NewU256(0)

	b2.PrevRandao[0] = 0xff
	if b1.Eq(b2) {
		t.Error("Different prev randao is considered the same")
	}
	b2.PrevRandao[0] = 0x00

	b2.TimeStamp = time.Now()
	if b1.Eq(b2) {
		t.Error("Different timestamp is considered the same")
	}
	b2.TimeStamp = time.Time{}
}

func TestBlockContext_Diff(t *testing.T) {
	b1 := NewBlockContext()
	b2 := NewBlockContext()
	diffs := []string{}

	if diffs = b1.Diff(&b2); len(diffs) != 0 {
		t.Error("Found differencees in two new block contexts.")
	}

	b2.BlockNumber++
	if diffs = b1.Diff(&b2); len(diffs) == 0 {
		t.Error("No difference found in different block numbers")
	}

	b2.CoinBase[0] = 0xff
	if diffs = b1.Diff(&b2); len(diffs) == 0 {
		t.Error("No difference found in different coinbase")
	}

	b2.GasLimit = NewU256(1)
	if diffs = b1.Diff(&b2); len(diffs) == 0 {
		t.Error("No difference found in different gas limit")
	}

	b2.GasPrice = NewU256(1)
	if diffs = b1.Diff(&b2); len(diffs) == 0 {
		t.Error("No difference found in different gas price")
	}

	b2.PrevRandao[0] = 0xff
	if diffs = b1.Diff(&b2); len(diffs) == 0 {
		t.Error("No difference found in different prev randao")
	}

	b2.TimeStamp = time.Now()
	if diffs = b1.Diff(&b2); len(diffs) == 0 {
		t.Error("No difference found in different timestamp")
	}

}

func TestBlockContext_String(t *testing.T) {
	b := NewBlockContext()
	b.BlockNumber++
	b.CoinBase[0] = 0xff
	b.GasLimit = NewU256(1)
	b.GasPrice = NewU256(1)
	b.PrevRandao[0] = 0xff
	b.TimeStamp = time.Now()
	str := b.String()

	if !strings.Contains(str, fmt.Sprintf("Block Number: %v", b.BlockNumber)) {
		t.Errorf("Did not find block number string.")
	}
	if !strings.Contains(str, fmt.Sprintf("CoinBase: %v", b.CoinBase)) {
		t.Errorf("Did not find coinbase string.")
	}
	if !strings.Contains(str, fmt.Sprintf("Gas Limit: %v", b.GasLimit)) {
		t.Errorf("Did not find gas limit string.")
	}
	if !strings.Contains(str, fmt.Sprintf("Gas Price: %v", b.GasPrice)) {
		t.Errorf("Did not find gas price string.")
	}
	if !strings.Contains(str, fmt.Sprintf("Prev Randao: %v", b.PrevRandao)) {
		t.Errorf("Did not find prev randao string.")
	}
	if !strings.Contains(str, fmt.Sprintf("Timestamp: %v", b.TimeStamp)) {
		t.Errorf("Did not find timestamp string.")
	}
}

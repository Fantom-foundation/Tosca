package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// Analogous to evmc_message
// The message describing an EVM call
type MessageCtx struct {
	AccountAddr *Address
}

func NewMsgCtx() *MessageCtx {
	return &MessageCtx{NewAddress()}
}

// create a copy and return it.
func (mc *MessageCtx) Clone() *MessageCtx {
	ret := MessageCtx{}
	ret.AccountAddr = mc.AccountAddr.Clone()
	return &ret
}

func (mc *MessageCtx) Eq(other *MessageCtx) bool {
	return mc.AccountAddr.Eq(other.AccountAddr)
}

// Diff returns a list of differences between the two addresses.
func (mc *MessageCtx) Diff(other *MessageCtx) []string {
	ret := []string{}

	accountName := fmt.Sprintf("Account Address")
	differences := mc.AccountAddr.Diff(other.AccountAddr, accountName)
	if len(differences) != 0 {
		ret = append(ret, differences...)
	}

	return ret
}

func (mc *MessageCtx) String() string {
	return fmt.Sprintf("AccountAddr: %v", mc.AccountAddr.String())
}

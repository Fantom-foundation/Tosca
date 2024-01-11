package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// CallCtx holds all data needed for the call-group of instructions
type CallCtx struct {
	AccountAddr *Address
}

func NewCallCtx() *CallCtx {
	return &CallCtx{NewAddress()}
}

// Clone creates an independent copy of the call context.
func (mc *CallCtx) Clone() *CallCtx {
	ret := CallCtx{}
	ret.AccountAddr = mc.AccountAddr.Clone()
	return &ret
}

func (mc *CallCtx) Eq(other *CallCtx) bool {
	return mc.AccountAddr.Eq(other.AccountAddr)
}

// Diff returns a list of differences between the two call contexts.
func (mc *CallCtx) Diff(other *CallCtx) []string {
	ret := []string{}

	differences := mc.AccountAddr.Diff(other.AccountAddr)
	if len(differences) != 0 {
		str := fmt.Sprintf("Different account address: ")
		for _, dif := range differences {
			str += dif
		}

		ret = append(ret, str)
	}

	return ret
}

func (mc *CallCtx) String() string {
	return fmt.Sprintf("Call Context: (Account Address: %v)", mc.AccountAddr.String())
}

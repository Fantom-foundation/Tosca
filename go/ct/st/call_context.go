package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// CallContext holds all data needed for the call-group of instructions
type CallContext struct {
	AccountAddress Address // Address of currently executing account
}

func NewCallContext() *CallContext {
	return &CallContext{NewAddress()}
}

// Clone creates an independent copy of the call context.
func (mc *CallContext) Clone() *CallContext {
	ret := CallContext{}
	ret.AccountAddress = *mc.AccountAddress.Clone()
	return &ret
}

func (mc *CallContext) Eq(other *CallContext) bool {
	return mc.AccountAddress == other.AccountAddress
}

// Diff returns a list of differences between the two call contexts.
func (mc *CallContext) Diff(other *CallContext) []string {
	ret := []string{}

	differences := mc.AccountAddress.Diff(&other.AccountAddress)
	if len(differences) != 0 {
		str := fmt.Sprintf("Different account address: ")
		for _, dif := range differences {
			str += dif
		}

		ret = append(ret, str)
	}

	return ret
}

func (mc *CallContext) String() string {
	return fmt.Sprintf("Call Context: (Account Address: %v)", mc.AccountAddress)
}

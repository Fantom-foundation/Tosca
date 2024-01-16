package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// CallContext holds all data needed for the call-group of instructions
type CallContext struct {
	AccountAddress Address // Address of currently executing account
	OriginAddress  Address // Address of execution origination
	CallerAddress  Address // Address of the caller
	Value          U256    // Deposited value by the instruction/transaction responsible for this execution
}

func NewCallContext() CallContext {
	return CallContext{}
}

// Diff returns a list of differences between the two call contexts.
func (c *CallContext) Diff(other *CallContext) []string {
	ret := []string{}
	callContextDiff := "Different call context "

	if c.AccountAddress != other.AccountAddress {
		ret = append(ret, callContextDiff+fmt.Sprintf("account address: %v vs. %v\n", c.AccountAddress, other.AccountAddress))
	}

	if c.OriginAddress != other.OriginAddress {
		ret = append(ret, callContextDiff+fmt.Sprintf("origin address: %v vs. %v\n", c.OriginAddress, other.OriginAddress))
	}

	if c.CallerAddress != other.CallerAddress {
		ret = append(ret, callContextDiff+fmt.Sprintf("caller address: %v vs. %v\n", c.CallerAddress, other.CallerAddress))
	}

	if !c.Value.Eq(other.Value) {
		ret = append(ret, callContextDiff+fmt.Sprintf("call value %v vs %v\n", c.Value, other.Value))
	}

	if !c.Value.Eq(other.Value) {
		ret = append(ret, fmt.Sprintf("Different call value %v vs %v.", c.Value, other.Value))
	}

	return ret
}

func (c *CallContext) String() string {
	return fmt.Sprintf(
		"Call Context:\n\t    Account Address: %v,\n\t    Origin Address: %v,\n\t    Caller Address: %v,\n"+
			"\t    Call Value: %v\n",
		c.AccountAddress, c.OriginAddress, c.CallerAddress, c.Value)
}

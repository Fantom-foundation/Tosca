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

func NewCallContext() *CallContext {
	return &CallContext{}
}

// Clone creates an independent copy of the call context.
func (c *CallContext) Clone() *CallContext {
	ret := *c
	return &ret
}

func (c *CallContext) Eq(other *CallContext) bool {
	return c.AccountAddress == other.AccountAddress &&
		c.OriginAddress == other.OriginAddress &&
		c.CallerAddress == other.CallerAddress &&
		c.Value.Eq(other.Value)
}

// Diff returns a list of differences between the two call contexts.
func (c *CallContext) Diff(other *CallContext) []string {
	ret := []string{}

	addressDifferences := func(diffs []string, name string) []string {
		ret := []string{}
		if len(diffs) != 0 {
			str := fmt.Sprintf("Different %v address: ", name)
			for _, dif := range diffs {
				str += dif
			}
			ret = append(ret, str)
		}
		return ret
	}

	differences := c.AccountAddress.Diff(other.AccountAddress)
	ret = append(ret, addressDifferences(differences, "account")...)

	differences = c.OriginAddress.Diff(other.OriginAddress)
	ret = append(ret, addressDifferences(differences, "origin")...)

	differences = c.CallerAddress.Diff(other.CallerAddress)
	ret = append(ret, addressDifferences(differences, "caller")...)

	if !c.Value.Eq(other.Value) {
		ret = append(ret, fmt.Sprintf("Different call value %v vs %v.", c.Value, other.Value))
	}

	return ret
}

func (c *CallContext) String() string {
	return fmt.Sprintf(
		"Call Context: (Account Address: %v, Origin Address: %v, Caller Address: %v, Call Value: %v)",
		c.AccountAddress, c.OriginAddress, c.CallerAddress, c.Value)
}

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
}

func NewCallContext() *CallContext {
	return &CallContext{Address{}, Address{}, Address{}}
}

// Clone creates an independent copy of the call context.
func (c *CallContext) Clone() *CallContext {
	ret := CallContext{}
	ret.AccountAddress = c.AccountAddress.Clone()
	ret.OriginAddress = c.OriginAddress.Clone()
	ret.CallerAddress = c.CallerAddress.Clone()
	return &ret
}

func (c *CallContext) Eq(other *CallContext) bool {
	return c.AccountAddress == other.AccountAddress &&
		c.OriginAddress == other.OriginAddress &&
		c.CallerAddress == other.CallerAddress
}

func addr_differences(diffs []string, name string) []string {
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

// Diff returns a list of differences between the two call contexts.
func (c *CallContext) Diff(other *CallContext) []string {
	ret := []string{}

	differences := c.AccountAddress.Diff(other.AccountAddress)
	ret = append(ret, addr_differences(differences, "account")...)

	differences = c.OriginAddress.Diff(other.OriginAddress)
	ret = append(ret, addr_differences(differences, "origin")...)

	differences = c.CallerAddress.Diff(other.CallerAddress)
	ret = append(ret, addr_differences(differences, "caller")...)

	return ret
}

func (c *CallContext) String() string {
	return fmt.Sprintf("Call Context: (Account Address: %v, Origin Address: %v, Caller Address: %v)", c.AccountAddress, c.OriginAddress, c.CallerAddress)
}

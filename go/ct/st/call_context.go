package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// CallContext holds all data needed for the call-group of instructions
type CallContext struct {
	AccountAddress Address // Address of currently executing account
	OriginAddress  Address // Address of execution origination
}

func NewCallContext() *CallContext {
	return &CallContext{Address{}, Address{}}
}

// Clone creates an independent copy of the call context.
func (c *CallContext) Clone() *CallContext {
	ret := CallContext{}
	ret.AccountAddress = c.AccountAddress.Clone()
	ret.OriginAddress = c.OriginAddress.Clone()
	return &ret
}

func (c *CallContext) Eq(other *CallContext) bool {
	return c.AccountAddress == other.AccountAddress &&
		c.OriginAddress == other.OriginAddress
}

// Diff returns a list of differences between the two call contexts.
func (c *CallContext) Diff(other *CallContext) []string {
	ret := []string{}

	differences := c.AccountAddress.Diff(other.AccountAddress)
	if len(differences) != 0 {
		str := "Different account address: "
		for _, dif := range differences {
			str += dif
		}
		ret = append(ret, str)
	}

	differences = c.OriginAddress.Diff(other.OriginAddress)
	if len(differences) != 0 {
		str := fmt.Sprintf("Different origin address: ")
		for _, dif := range differences {
			str += dif
		}
		ret = append(ret, str)
	}

	return ret
}

func (c *CallContext) String() string {
	return fmt.Sprintf("Call Context: (Account Address: %v, Origin Address: %v)", c.AccountAddress, c.OriginAddress)
}

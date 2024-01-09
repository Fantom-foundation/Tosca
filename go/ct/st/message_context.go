package st

import (
	"fmt"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// Analogous to evmc_message
// The message describing an EVM call
type MessageCtx struct {
	maxGas       uint64
	recipient    *Address
	sender       *Address
	codeAddress  *Address
	contractAddr *Address
}

func NewMsgCtx() *MessageCtx {
	return &MessageCtx{0, NewAddress(), NewAddress(), NewAddress(), NewAddress()}
}

// create a copy and return it.
func (mc *MessageCtx) Clone() *MessageCtx {
	ret := MessageCtx{}
	ret.maxGas = mc.maxGas
	ret.recipient = mc.recipient
	ret.sender = mc.sender
	ret.codeAddress = mc.codeAddress
	ret.contractAddr = mc.contractAddr
	return &ret
}

func (mc *MessageCtx) Eq(other *MessageCtx) bool {
	return mc.maxGas == other.maxGas &&
		mc.recipient.Eq(other.recipient) &&
		mc.sender.Eq(other.sender) &&
		mc.codeAddress.Eq(other.codeAddress) &&
		mc.contractAddr.Eq(other.contractAddr)
}

// Diff returns a list of differences between the two addresses.
func (mc *MessageCtx) Diff(other *MessageCtx) []string {
	ret := []string{}

	if mc.maxGas != other.maxGas {
		ret = append(ret, fmt.Sprintf("Msg Ctx with different maxGas %v vs %v", mc.maxGas, other.maxGas))
	}

	differences := mc.sender.Diff(other.sender)
	if len(differences) != 0 {
		ret = append(ret, "Different caller addreess:")
		ret = append(ret, differences...)
	}

	differences = mc.recipient.Diff(other.recipient)
	if len(differences) != 0 {
		ret = append(ret, "Different reciver addreess:")
		ret = append(ret, differences...)
	}

	differences = mc.codeAddress.Diff(other.codeAddress)
	if len(differences) != 0 {
		ret = append(ret, "Different code addreess:")
		ret = append(ret, differences...)
	}

	differences = mc.contractAddr.Diff(other.contractAddr)
	if len(differences) != 0 {
		ret = append(ret, "Different current addreess:")
		ret = append(ret, differences...)
	}

	return ret
}

func (mc *MessageCtx) String() string {
	return fmt.Sprintf("gas: %v rec: %v sender: %v, codeAddr: %v, contractAddr: %v", mc.maxGas, mc.recipient, mc.sender, mc.codeAddress, mc.contractAddr)
}

// Address of the currently executing code
func (mc *MessageCtx) GetContractAddr() *Address {
	return mc.contractAddr
}

// Address of the
func (mc *MessageCtx) GetSenderAddress() *Address {
	return mc.sender
}

func (mc *MessageCtx) CloneContractAddr(addr *Address) {
	mc.contractAddr = addr
}

func (mc *MessageCtx) CloneSenderAddr(addr *Address) {
	mc.sender = addr
}

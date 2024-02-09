package gen

import (
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type CallContextGenerator struct {
}

func NewCallContextGenerator() *CallContextGenerator {
	return &CallContextGenerator{}
}

func (*CallContextGenerator) Generate(rnd *rand.Rand) (st.CallContext, error) {
	accountAddress, err := common.RandAddress(rnd)
	if err != nil {
		return st.CallContext{}, err
	}

	originAddress, err := common.RandAddress(rnd)
	if err != nil {
		return st.CallContext{}, err
	}

	callerAddress, err := common.RandAddress(rnd)
	if err != nil {
		return st.CallContext{}, err
	}

	return st.CallContext{
		AccountAddress: accountAddress,
		OriginAddress:  originAddress,
		CallerAddress:  callerAddress,
		Value:          common.RandU256(rnd),
	}, nil
}

func (*CallContextGenerator) Clone() *CallContextGenerator {
	return &CallContextGenerator{}
}

func (*CallContextGenerator) Restore(*CallContextGenerator) {
}

func (*CallContextGenerator) String() string {
	return "{}"
}

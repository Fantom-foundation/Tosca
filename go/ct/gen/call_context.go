package gen

import (
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type CallCtxGenerator struct {
}

func NewCallCtxGenerator() *CallCtxGenerator {
	return &CallCtxGenerator{}
}

func (ccg *CallCtxGenerator) Generate(rnd *rand.Rand) (*st.CallCtx, error) {
	accountAddr, err := common.RandAddress(rnd)
	if err != nil {
		return nil, err
	}

	newCC := st.NewCallCtx()
	newCC.AccountAddr = accountAddr

	return newCC, nil
}

func (ccg *CallCtxGenerator) Clone() *CallCtxGenerator {
	return &CallCtxGenerator{}
}

func (*CallCtxGenerator) Restore(*CallCtxGenerator) {
}

func (ccg *CallCtxGenerator) String() string {
	return "{}"
}

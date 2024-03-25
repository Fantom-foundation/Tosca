package gen

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/vm"
	"pgregory.net/rand"
)

func TestCallContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	callCtxGen := NewCallContextGenerator()
	accountAddress, err := common.RandAddress(rnd)
	if err != nil {
		t.Errorf("Unexpected random address generation error: %v", err)
	}
	callCtx, err := callCtxGen.Generate(rnd, accountAddress)
	if err != nil {
		t.Errorf("Error generating call context: %v", err)
	}

	if callCtx.AccountAddress == (vm.Address{}) {
		t.Errorf("Generated account address has default value.")
	}
	if callCtx.OriginAddress == (vm.Address{}) {
		t.Errorf("Generated origin address has default value.")
	}
	if callCtx.CallerAddress == (vm.Address{}) {
		t.Errorf("Generated caller address has default value.")
	}
	if callCtx.Value.Eq(common.NewU256(0)) {
		t.Errorf("Generated call value has default value.")
	}
}

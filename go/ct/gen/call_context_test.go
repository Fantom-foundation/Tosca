package gen

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestCallContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	callCtxGen := NewCallContextGenerator()
	callCtx, err := callCtxGen.Generate(rnd)
	if err != nil {
		t.Errorf("Error generating call context: %v", err)
	}

	if callCtx.AccountAddress == (common.Address{}) {
		t.Errorf("Generated account address has default value.")
	}
	if callCtx.OriginAddress == (common.Address{}) {
		t.Errorf("Generated origin address has default value.")
	}
	if callCtx.CallerAddress == (common.Address{}) {
		t.Errorf("Generated caller address has default value.")
	}
	if callCtx.Value.Eq(common.NewU256(0)) {
		t.Errorf("Generated call value has default value.")
	}
}

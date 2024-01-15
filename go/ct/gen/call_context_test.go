package gen

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestCallContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	callctxGen := NewCallContextGenerator()
	newCC, err := callctxGen.Generate(rnd)
	if err != nil {
		t.Errorf("Error generating call context: %v", err)
	}

	if newCC.AccountAddress == (common.Address{}) {
		t.Errorf("Generated account address has default value.")
	}

	if newCC.OriginAddress == (common.Address{}) {
		t.Errorf("Generated origin address has default value.")
	}

	if newCC.CallerAddress == (common.Address{}) {
		t.Errorf("Generated caller address has default value.")
	}

	if newCC.Value == nil {
		t.Errorf("Generated context does not generate call value")
	}
	if newCC.Value.Cmp(big.NewInt(0)) == 0 {
		t.Errorf("Generated call value has default value.")
	}
}

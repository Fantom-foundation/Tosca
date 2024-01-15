package gen

import (
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
}

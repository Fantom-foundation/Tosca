package gen

import (
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func testAddr(t *testing.T, address *common.Address, name string) {
	if *address == (common.Address{}) {
		t.Errorf("Generated %v address has default value.", name)
	}
}

func TestCallContextGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	callctxGen := NewCallContextGenerator()
	newCC, err := callctxGen.Generate(rnd)
	if err != nil {
		t.Errorf("Error generating call context: %v", err)
	}

	testAddr(t, &newCC.AccountAddress, "account")
	testAddr(t, &newCC.OriginAddress, "origin")
	testAddr(t, &newCC.CallerAddress, "caller")

	if newCC.Value == nil {
		t.Errorf("Generated context does not generate call value")
	}
	if newCC.Value.Cmp(big.NewInt(0)) == 0 {
		t.Errorf("Generated call value has default value.")
	}
}

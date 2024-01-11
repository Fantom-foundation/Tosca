package gen

import (
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
}

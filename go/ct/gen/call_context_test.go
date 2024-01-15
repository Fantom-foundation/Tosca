package gen

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func testAddress(t *testing.T, address *common.Address, name string) {
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

	if newCC.AccountAddress == (common.Address{}) {
		t.Errorf("Generated account address has default value.")
	}

	if newCC.OriginAddress == (common.Address{}) {
		t.Errorf("Generated origin address has default value.")
	}

	if newCC.CallerAddress == (common.Address{}) {
		t.Errorf("Generated caller address has default value.")
	}

	if newCC.Value.Eq(common.NewU256(0)) {
		t.Errorf("Generated call value has default value.")
	}
}

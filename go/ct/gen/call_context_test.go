package gen

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"pgregory.net/rand"
)

func TestCallCtxGen_Generate(t *testing.T) {
	rnd := rand.New(0)
	callctxGen := NewCallCtxGenerator()
	newCC, err := callctxGen.Generate(rnd)
	if err != nil {
		t.Errorf("Error generating call context: %v", err)
	}
	if newCC.AccountAddr == nil {
		t.Errorf("Generated context does not generate account address")
	}
	if newCC.AccountAddr.Eq(common.NewAddress()) {
		t.Errorf("Generated account address has default value.")
	}
}

package gen

import (
	"pgregory.net/rand"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type TransientGenerator struct {
}

func NewTransientGenerator() *TransientGenerator {
	return &TransientGenerator{}
}

func (t *TransientGenerator) Generate(rnd *rand.Rand) (*st.Transient, error) {
	transient := &st.Transient{}

	// Some entries with keys returned by the parameter samples function
	transient.SetStorage(common.NewU256(0), common.NewU256(5))
	transient.SetStorage(common.NewU256(1), common.NewU256(10))
	transient.SetStorage(common.NewU256(1<<8), common.NewU256(15<<8))
	transient.SetStorage(common.NewU256(1<<16), common.NewU256(20<<16))
	transient.SetStorage(common.NewU256(1<<32), common.NewU256(25<<32))

	// Random entries
	for i := 0; i < rnd.Intn(42); i++ {
		key := common.RandU256(rnd)
		value := common.RandU256(rnd)

		transient.SetStorage(key, value)
	}

	return transient, nil
}

func (t *TransientGenerator) Clone() *TransientGenerator {
	return &TransientGenerator{}
}

func (*TransientGenerator) Restore(*TransientGenerator) {
}

func (t *TransientGenerator) String() string {
	return "{}"
}

package floria

import (
	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func init() {
	tosca.RegisterProcessorFactory("floria", newProcessor)
}

func newProcessor(interpreter tosca.Interpreter) tosca.Processor {
	if interpreter == nil {
		panic("interpreter is required")
	}

	return &processor{
		interpreter: interpreter,
	}
}

type processor struct {
	interpreter tosca.Interpreter
}

func (p *processor) Run(
	blockParams tosca.BlockParameters,
	transaction tosca.Transaction,
	context tosca.TransactionContext,
) (tosca.Receipt, error) {
	return tosca.Receipt{}, nil
}

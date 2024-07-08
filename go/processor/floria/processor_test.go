package floria

import (
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestProcessorRegistry_InitProcessor(t *testing.T) {
	processorFactories := tosca.GetAllRegisteredProcessorFactories()

	if len(processorFactories) == 0 {
		t.Errorf("No processor factories found")
	}

	processor := tosca.GetProcessorFactory("floria")
	if processor == nil {
		t.Errorf("Floria processor factory not found")
	}

}

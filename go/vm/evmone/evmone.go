package evmone

/*
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../third_party/evmone/build/lib
*/
import "C"

import (
	"fmt"

	"github.com/Fantom-foundation/Tosca/go/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

var evmone *common.EvmcVM

func init() {
	// In the CGO instructions at the top of this file the build directory
	// of the evmone project is added to the rpath of the resulting library.
	// This way, the libevmone.so file can be found during runtime, even if
	// the LD_LIBRARY_PATH is not set accordingly.
	vm, err := common.LoadEvmcVM("libevmone.so")
	if err != nil {
		panic(fmt.Errorf("failed to load evmone library: %s", err))
	}
	evmone = vm
}

func NewInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEvmcInterpreter(evmone, evm, cfg)
}

func init() {
	vm.RegisterInterpreterFactory("evmone", NewInterpreter)
}

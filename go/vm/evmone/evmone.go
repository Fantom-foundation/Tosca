package evmone

/*
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../third_party/evmone/build/lib
*/
import "C"

import (
	"github.com/Fantom-foundation/Tosca/go/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func NewInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	return common.NewEVMCInterpreter("libevmone.so", evm, cfg)
}

func init() {
	vm.RegisterInterpreterFactory("evmone", NewInterpreter)
}

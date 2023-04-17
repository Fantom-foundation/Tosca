package evmone

import (
	"github.com/Fantom-foundation/Tosca/go/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func NewInterpreter(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
	path := "../../../third_party/evmone/build/lib/libevmone.so"
	return common.NewEVMCInterpreter(path, evm, cfg)
}

func init() {
	vm.RegisterInterpreterFactory("evmone", NewInterpreter)
}

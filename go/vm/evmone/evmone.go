package evmone

//#cgo CFLAGS: -I${SRCDIR}/../../../cpp/vm/evmone/include
//#cgo CFLAGS: -I${SRCDIR}/../../../cpp/vm/evmone/evmc/include
//#cgo LDFLAGS: -L${SRCDIR}/../../../cpp/vm/evmone/build/lib -levmone
//#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/../../../cpp/vm/evmone/build/lib
//#include <evmone/evmone.h>
import "C"
import (
	"log"

	"github.com/ethereum/go-ethereum/core/vm"
)

type EVMInterpreter struct {
	evm *vm.EVM
	cfg vm.Config
}

func (e *EVMInterpreter) Run(contract *vm.Contract, input []byte, readOnly bool) (ret []byte, err error) {
	log.Fatalln("emvone run not implemented")
	return
}

func init() {
	vm.RegisterInterpreterFactory("evmone", func(evm *vm.EVM, cfg vm.Config) vm.EVMInterpreter {
		return &EVMInterpreter{evm: evm, cfg: cfg}
	})
}

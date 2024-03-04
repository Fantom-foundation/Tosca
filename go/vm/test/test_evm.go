package vm_test

import (
	"github.com/Fantom-foundation/Tosca/go/vm"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/evmzero"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

//go:generate mockgen -source test_evm.go -destination test_evm_mock.go -package vm_test

const InitialTestGas vm.Gas = 1 << 44

type StateDB interface {
	AccountExists(vm.Address) bool
	GetStorage(vm.Address, vm.Key) vm.Word
	SetStorage(vm.Address, vm.Key, vm.Word) vm.StorageStatus
	GetBalance(vm.Address) vm.Value
	GetCodeSize(vm.Address) int
	GetCodeHash(vm.Address) vm.Hash
	GetCode(vm.Address) []byte
	GetBlockHash(int64) vm.Hash
	EmitLog(vm.Address, []vm.Hash, []byte)
	SelfDestruct(vm.Address, vm.Address) bool
	AccessAccount(vm.Address) vm.AccessStatus
	AccessStorage(vm.Address, vm.Key) vm.AccessStatus
	GetCommittedStorage(vm.Address, vm.Key) vm.Word
	IsAddressInAccessList(vm.Address) bool
	IsSlotInAccessList(vm.Address, vm.Key) (addressPresent, slotPresent bool)
	HasSelfDestructed(vm.Address) bool
}

// TestEVM is a minimal EVM implementation wrapping a VirtualMachine into an EVM
// instance capable of processing recursive calls. It is only intended to be be
// utilized for integration tests in this package, and thus misses almost all
// features of a fully functional EVM.
type TestEVM struct {
	vm       vm.VirtualMachine
	revision vm.Revision
	state    StateDB
}

func GetCleanEVM(revision Revision, interpreter string, stateDB StateDB) TestEVM {
	rev := vm.R07_Istanbul
	switch revision {
	case Istanbul:
		rev = vm.R07_Istanbul
	case Berlin:
		rev = vm.R09_Berlin
	case London:
		rev = vm.R10_London
	}

	return TestEVM{
		vm:       vm.GetVirtualMachine(interpreter),
		revision: rev,
		state:    stateDB,
	}
}

type RunResult struct {
	Output  []byte
	GasUsed vm.Gas
	Success bool
}

func (e *TestEVM) Run(code []byte, input []byte) (RunResult, error) {
	return e.RunWithGas(code, input, InitialTestGas)
}

func (e *TestEVM) RunWithGas(code []byte, input []byte, initialGas vm.Gas) (RunResult, error) {

	params := vm.Parameters{
		Context:  &runContextAdapter{e.state},
		Revision: e.revision,
		Code:     code,
		Input:    input,
		Gas:      initialGas,
		Depth:    0,
		Static:   false,
	}

	result, err := e.vm.Run(params)
	if err != nil {
		return RunResult{}, err
	}

	return RunResult{
		Output:  result.Output,
		GasUsed: initialGas - result.GasLeft,
		Success: result.Success,
	}, nil
}

// --- adapter ---

type runContextAdapter struct {
	StateDB
}

func (a *runContextAdapter) GetTransactionContext() vm.TransactionContext {
	return vm.TransactionContext{}
}

func (a *runContextAdapter) Call(kind vm.CallKind, parameter vm.CallParameter) (vm.CallResult, error) {
	panic("not implemented")
}

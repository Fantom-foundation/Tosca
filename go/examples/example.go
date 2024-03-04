package examples

import (
	"fmt"
	"math"

	"github.com/Fantom-foundation/Tosca/go/vm"
	"golang.org/x/crypto/sha3"
)

// Example is an executable description of a contract and an entry point with a (int)->int signature.
type Example struct {
	exampleSpec
	codeHash vm.Hash // the hash of the code
}

// exampleSpec specifies a contract and an entry point with a (int)->int signature.
type exampleSpec struct {
	Name      string
	code      []byte        // some contract code
	function  uint32        // identifier of the function in the contract to be called
	reference func(int) int // a reference function computing the same function
}

func (s exampleSpec) build() Example {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(s.code)
	var hash vm.Hash
	hasher.Sum(hash[0:0])
	return Example{
		exampleSpec: s,
		codeHash:    hash,
	}
}

type Result struct {
	Result  int
	UsedGas int64
}

// RunOn runs this example on the given interpreter, using the given argument.
func (e *Example) RunOn(interpreter vm.VirtualMachine, argument int) (Result, error) {

	const initialGas = math.MaxInt64
	params := vm.Parameters{
		Context:  &exampleRunContext{},
		Code:     e.code,
		CodeHash: (*vm.Hash)(&e.codeHash),
		Input:    encodeArgument(e.function, argument),
		Gas:      initialGas,
	}

	res, err := interpreter.Run(params)
	if err != nil {
		return Result{}, err
	}

	result, err := decodeOutput(res.Output)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Result:  result,
		UsedGas: initialGas - int64(res.GasLeft),
	}, nil
}

// RunRef runs the reference function of this example to produce the expected result.
func (e *Example) RunReference(argument int) int {
	return e.reference(argument)
}

func encodeArgument(function uint32, arg int) []byte {
	// see details of argument encoding: t.ly/kBl6
	data := make([]byte, 4+32) // parameter is padded up to 32 bytes

	// encode function selector in big-endian format
	data[0] = byte(function >> 24)
	data[1] = byte(function >> 16)
	data[2] = byte(function >> 8)
	data[3] = byte(function)

	// encode argument as a big-endian value
	data[4+28] = byte(arg >> 24)
	data[5+28] = byte(arg >> 16)
	data[6+28] = byte(arg >> 8)
	data[7+28] = byte(arg)

	return data
}

func decodeOutput(output []byte) (int, error) {
	if len(output) != 32 {
		return 0, fmt.Errorf("unexpected length of output; wanted 32, got %d", len(output))
	}
	return (int(output[28]) << 24) | (int(output[29]) << 16) | (int(output[30]) << 8) | (int(output[31]) << 0), nil
}

type exampleRunContext struct{}

func (c *exampleRunContext) AccountExists(vm.Address) bool {
	return false
}

func (c *exampleRunContext) GetStorage(vm.Address, vm.Key) vm.Word {
	return vm.Word{}
}

func (c *exampleRunContext) SetStorage(vm.Address, vm.Key, vm.Word) vm.StorageStatus {
	return vm.StorageAdded
}

func (c *exampleRunContext) GetBalance(vm.Address) vm.Value {
	return vm.Value{}
}

func (c *exampleRunContext) GetCodeSize(vm.Address) int {
	return 0
}

func (c *exampleRunContext) GetCodeHash(vm.Address) vm.Hash {
	return vm.Hash{}
}

func (c *exampleRunContext) GetCode(vm.Address) []byte {
	return nil
}

func (c *exampleRunContext) GetTransactionContext() vm.TransactionContext {
	return vm.TransactionContext{}
}

func (c *exampleRunContext) GetBlockHash(int64) vm.Hash {
	return vm.Hash{}
}

func (c *exampleRunContext) EmitLog(vm.Address, []vm.Hash, []byte) {
}

func (c *exampleRunContext) Call(vm.CallKind, vm.CallParameter) (vm.CallResult, error) {
	return vm.CallResult{}, nil
}

func (c *exampleRunContext) SelfDestruct(vm.Address, vm.Address) bool {
	return false
}

func (c *exampleRunContext) AccessAccount(vm.Address) vm.AccessStatus {
	return vm.ColdAccess
}

func (c *exampleRunContext) AccessStorage(vm.Address, vm.Key) vm.AccessStatus {
	return vm.ColdAccess
}

// -- legacy API needed by LFVM and Geth, to be removed in the future ---

func (c *exampleRunContext) GetCommittedStorage(vm.Address, vm.Key) vm.Word {
	return vm.Word{}
}

func (c *exampleRunContext) IsAddressInAccessList(vm.Address) bool {
	return false
}

func (c *exampleRunContext) IsSlotInAccessList(vm.Address, vm.Key) (bool, bool) {
	return false, false
}

func (c *exampleRunContext) HasSelfDestructed(vm.Address) bool {
	return false
}

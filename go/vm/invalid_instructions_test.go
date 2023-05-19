package vm

import (
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

func TestInterpreterDetectsInvalidInstruction(t *testing.T) {
	for _, rev := range revisions {
		evm := newTestEVM(rev)
		for _, variant := range variants {
			// LFVM currently does not support detection of invalid codes!
			// TODO: fix this
			if strings.Contains(variant, "lfvm") {
				continue
			}
			interpreter := vm.NewInterpreter(variant, evm, vm.Config{})
			instructions := getInstructions(rev)
			for i := 0; i < 256; i++ {
				op := vm.OpCode(i)
				_, exits := instructions[op]
				if exits {
					continue
				}
				t.Run(fmt.Sprintf("%s-%s-%s", variant, rev, op), func(t *testing.T) {
					code := []byte{byte(op), byte(vm.STOP)}
					input := []byte{}
					if err := runCode(interpreter, code, input); !isInvalidOpCodeError(err) {
						t.Errorf("failed to identify invalid OpCode %v as invalid instruction, got %v", op, err)
					}
				})
			}
		}
	}
}

func isInvalidOpCodeError(err error) bool {
	_, ok := err.(*vm.ErrInvalidOpCode)
	return ok
}

func runCode(interpreter vm.EVMInterpreter, code []byte, input []byte) error {
	const initialGas = math.MaxInt64

	// Create a dummy contract using the code hash as a address. This is required
	// to avoid all tests using the same cached LFVM byte code.
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), initialGas)
	contract.Code = code
	contract.CodeHash = getSha256Hash(code)

	// TODO: remove this once code caching is code-hash based.
	var codeAddr common.Address
	copy(codeAddr[:], contract.CodeHash[:])
	contract.CodeAddr = &codeAddr

	_, err := interpreter.Run(contract, input, false)
	return err
}

func getSha256Hash(code []byte) common.Hash {
	hasher := sha256.New()
	hasher.Write(code)
	var hash common.Hash
	hasher.Sum(hash[0:0])
	return hash
}

package vm

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/examples"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	_ "github.com/Fantom-foundation/Tosca/go/vm/evmone"
	_ "github.com/Fantom-foundation/Tosca/go/vm/lfvm"
)

var (
	variants = []string{"geth", "lfvm", "lfvm-si", "evmone"}
)

func TestFib_ComputesCorrectResult(t *testing.T) {
	evm := newTestEVM(London)
	fib := examples.GetFibExample()
	for _, variant := range variants {
		interpreter := vm.NewInterpreter(variant, evm, vm.Config{})
		for i := 0; i < 10; i++ {
			t.Run(fmt.Sprintf("%s-%d", variant, i), func(t *testing.T) {
				want := fib.RunReference(i)
				got, err := fib.RunOn(interpreter, i)
				if err != nil {
					t.Fatalf("error processing contract: %v", err)
				}
				if want != got.Result {
					t.Fatalf("incorrect result, wanted %d, got %d", want, got.Result)
				}
			})
		}
	}
}

func TestFib_ComputesCorrectGasPrice(t *testing.T) {
	fib := examples.GetFibExample()
	for _, revision := range revisions {
		evm := newTestEVM(revision)
		reference := vm.NewInterpreter("geth", evm, vm.Config{})
		for _, variant := range variants {
			interpreter := vm.NewInterpreter(variant, evm, vm.Config{})
			for i := 0; i < 10; i++ {
				t.Run(fmt.Sprintf("%s-%s-%d", revision, variant, i), func(t *testing.T) {
					want, err := fib.RunOn(reference, i)
					if err != nil {
						t.Fatalf("failed to run reference VM: %v", err)
					}

					got, err := fib.RunOn(interpreter, i)
					if err != nil {
						t.Fatalf("error processing contract: %v", err)
					}

					if want.UsedGas != got.UsedGas {
						t.Errorf("incorrect gas usage, wanted %d, got %d", want.UsedGas, got.UsedGas)
					}
				})
			}
		}
	}
}

type Revision int

const (
	Istanbul Revision = 1
	Berlin   Revision = 2
	London   Revision = 3
)

var revisions = []Revision{Istanbul, Berlin, London}

func (r Revision) String() string {
	switch r {
	case Istanbul:
		return "Istanbul"
	case Berlin:
		return "Berlin"
	case London:
		return "London"
	}
	return "Unknown"
}

func newTestEVM(r Revision) *vm.EVM {
	// Configure the block numbers for revision changes.
	chainConfig := params.AllEthashProtocolChanges
	chainConfig.BerlinBlock = big.NewInt(10)
	chainConfig.LondonBlock = big.NewInt(20)

	// Choose the block height to run.
	block := 5
	if r == Berlin {
		block = 15
	} else if r == London {
		block = 25
	}

	blockCtxt := vm.BlockContext{
		BlockNumber: big.NewInt(int64(block)),
	}
	txCtxt := vm.TxContext{}
	config := vm.Config{}
	return vm.NewEVM(blockCtxt, txCtxt, nil, chainConfig, config)
}

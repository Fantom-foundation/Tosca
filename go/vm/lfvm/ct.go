package lfvm

import (
	"fmt"
	"math/big"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
)

////////////////////////////////////////////////////////////
// lfvm -> ct : helper functions

func convertLfvmStatusToCtStatus(status Status) (st.StatusCode, error) {
	switch status {
	case RUNNING:
		return st.Running, nil
	case STOPPED:
		return st.Stopped, nil
	case REVERTED:
		return st.Reverted, nil
	case RETURNED:
		return st.Returned, nil
	case SUICIDED:
		// Suicide is not yet modeled by the CT, and for now it just maps to the STOPPED status.
		return st.Stopped, nil
	case INVALID_INSTRUCTION:
		return st.Failed, nil
	case OUT_OF_GAS:
		return st.Failed, nil
	case SEGMENTATION_FAULT:
		return st.Failed, nil
	case ERROR:
		return st.Failed, nil
	default:
		return st.Failed, fmt.Errorf("unable to convert lfvm status %v to ct status", status)
	}
}

func convertLfvmRevisionToCtRevision(ctx *context) (st.Revision, error) {
	if ctx.isBerlin && !ctx.isLondon {
		return st.Berlin, nil
	} else if !ctx.isBerlin && ctx.isLondon {
		return st.London, nil
	} else if !ctx.isBerlin && !ctx.isLondon {
		return st.Istanbul, nil
	} else {
		return st.NumRevisions, fmt.Errorf("invalid revision, both berlin and london set")
	}
}

func convertLfvmPcToCtPc(ctx *context, originalCode *st.Code) (uint16, error) {
	code := make([]byte, originalCode.Length())
	originalCode.CopyTo(code)
	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		return 0, err
	}
	pc, ok := pcMap.lfvmToEvm[uint16(ctx.pc)]
	if !ok {
		return 0, fmt.Errorf("unable to convert lfvm pc %d to evm pc", ctx.pc)
	}
	return pc, nil
}

func convertLfvmStackToCtStack(ctx *context) *st.Stack {
	stack := st.NewStack()
	for i := ctx.stack.len() - 1; i >= 0; i-- {
		val := ctx.stack.Data()[i]
		stack.Push(ct.NewU256(val[3], val[2], val[1], val[0]))
	}
	return stack
}

////////////////////////////////////////////////////////////
// lfvm -> ct

func ConvertLfvmContextToCtState(ctx *context, originalCode *st.Code) (*st.State, error) {
	status, err := convertLfvmStatusToCtStatus(ctx.status)
	if err != nil {
		return nil, err
	}

	pc, err := convertLfvmPcToCtPc(ctx, originalCode)
	if err != nil {
		return nil, err
	}

	revision, err := convertLfvmRevisionToCtRevision(ctx)
	if err != nil {
		return nil, err
	}

	state := st.State{
		Status:   status,
		Revision: revision,
		Pc:       pc,
		Gas:      ctx.contract.Gas,
		Code:     originalCode,
		Stack:    convertLfvmStackToCtStack(ctx),
	}

	return &state, nil
}

////////////////////////////////////////////////////////////
// ct -> lfvm : helper functions

func convertCtPcToLfvmPc(state *st.State) (int32, error) {
	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)
	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		return 0, err
	}
	pc, ok := pcMap.evmToLfvm[state.Pc]
	if !ok {
		return 0, fmt.Errorf("unable to convert evm pc %d to lfvm pc", state.Pc)
	}
	return int32(pc), nil
}

func convertCtCodeToLfvmCode(state *st.State) (Code, error) {
	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)
	return convert(code, false)
}

func convertCtStatusToLfvmStatus(state *st.State) (Status, error) {
	switch state.Status {
	case st.Running:
		return RUNNING, nil
	case st.Stopped:
		return STOPPED, nil
	case st.Returned:
		return RETURNED, nil
	case st.Reverted:
		return REVERTED, nil
	case st.Failed:
		return ERROR, nil
	default:
		return ERROR, fmt.Errorf("unable to convert ct status %v to lfvm status", state.Status)
	}
}

func convertCtStackToLfvmStack(state *st.State) *Stack {
	stack := NewStack()
	for i := 0; i < state.Stack.Size(); i++ {
		val := state.Stack.Get(i).Uint256()
		stack.push(&val)
	}
	return stack
}

func convertCtRevisionToLfvmRevision(revision st.Revision, ctx *context) error {
	switch revision {
	case st.Istanbul:
		// True by default in context.
	case st.Berlin:
		ctx.isBerlin = true
	case st.London:
		ctx.isLondon = true
	default:
		return fmt.Errorf("failed to convert revision: %v", revision)
	}
	return nil
}

////////////////////////////////////////////////////////////
// ct -> lfvm

func ConvertCtStateToLfvmContext(state *st.State) (*context, error) {
	// Create a dummy contract.
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), state.Gas)

	pc, err := convertCtPcToLfvmPc(state)
	if err != nil {
		return nil, err
	}

	status, err := convertCtStatusToLfvmStatus(state)
	if err != nil {
		return nil, err
	}

	code, err := convertCtCodeToLfvmCode(state)
	if err != nil {
		return nil, err
	}

	data := []byte{}

	// Create execution context.
	ctx := context{
		evm:      &vm.EVM{StateDB: nil},
		pc:       pc,
		stack:    convertCtStackToLfvmStack(state),
		memory:   NewMemory(),
		stateDB:  nil,
		status:   status,
		contract: contract,
		code:     code,
		data:     data,
		callsize: *uint256.NewInt(uint64(len(data))),
		readOnly: false,
		isBerlin: state.Revision == st.Berlin,
		isLondon: state.Revision == st.London,
	}

	err = convertCtRevisionToLfvmRevision(state.Revision, &ctx)
	if err != nil {
		return nil, err
	}

	return &ctx, nil
}

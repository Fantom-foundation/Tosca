package lfvm

import (
	"fmt"
	"math/big"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

func convertLfvmRevisionToCtRevision(ctx *context) (ct.Revision, error) {
	if ctx.isBerlin && !ctx.isLondon {
		return ct.R09_Berlin, nil
	} else if !ctx.isBerlin && ctx.isLondon {
		return ct.R10_London, nil
	} else if !ctx.isBerlin && !ctx.isLondon {
		return ct.R07_Istanbul, nil
	} else {
		return -1, fmt.Errorf("invalid revision, both berlin and london set")
	}
}

func convertLfvmStackToCtStack(ctx *context) *st.Stack {
	stack := st.NewStack()

	for i := 0; i < ctx.stack.len(); i++ {
		val := ctx.stack.Data()[i]
		stack.Push(ct.NewU256(val[3], val[2], val[1], val[0]))
	}
	return stack
}

func convertLfvmMemoryToCtMemory(ctx *context) *st.Memory {
	memory := st.NewMemory()
	memory.Set(ctx.memory.GetSlice(0, ctx.memory.Len()))
	return memory
}

////////////////////////////////////////////////////////////
// lfvm -> ct

func ConvertLfvmContextToCtState(ctx *context, originalCode *st.Code, pcMap *PcMap) (*st.State, error) {
	status, err := convertLfvmStatusToCtStatus(ctx.status)
	if err != nil {
		return nil, err
	}

	pc, ok := pcMap.lfvmToEvm[uint16(ctx.pc)]

	// Since two failed states are considered equal, the PC conversion may fail when the status is failed.
	if !ok && status != st.Failed {
		return nil, fmt.Errorf("unable to convert lfvm pc %d to evm pc", ctx.pc)
	}

	revision, err := convertLfvmRevisionToCtRevision(ctx)
	if err != nil {
		return nil, err
	}

	state := st.NewState(originalCode)
	state.Status = status
	state.Revision = revision
	state.Pc = pc
	state.Gas = ctx.contract.Gas
	state.Code = originalCode
	state.Stack = convertLfvmStackToCtStack(ctx)
	state.Memory = convertLfvmMemoryToCtMemory(ctx)
	if ctx.stateDB != nil {
		state.Storage = ctx.stateDB.(*conformanceTestStateDb).storage
	}
	return state, nil
}

////////////////////////////////////////////////////////////
// ct -> lfvm : helper functions

func convertCtCodeToLfvmCode(state *st.State) (Code, error) {
	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)
	addr := common.Address{}
	return Convert(addr, code, false, 0, false, false, state.Code.Hash())
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
	for i := state.Stack.Size() - 1; i >= 0; i-- {
		val := state.Stack.Get(i).Uint256()
		stack.push(&val)
	}
	return stack
}

func convertCtMemoryToLfvmMemory(state *st.State) (*Memory, error) {
	data := state.Memory.Read(0, uint64(state.Memory.Size()))

	memory := NewMemory()
	memory.EnsureCapacityWithoutGas(uint64(len(data)), nil)
	err := memory.Set(0, uint64(len(data)), data)
	return memory, err
}

func convertCtRevisionToLfvmRevision(revision ct.Revision, ctx *context) error {
	switch revision {
	case ct.R07_Istanbul:
		// True by default in context.
	case ct.R09_Berlin:
		ctx.isBerlin = true
	case ct.R10_London:
		ctx.isLondon = true
	default:
		return fmt.Errorf("failed to convert revision: %v", revision)
	}
	return nil
}

////////////////////////////////////////////////////////////
// ct -> lfvm

func ConvertCtStateToLfvmContext(state *st.State, pcMap *PcMap) (*context, error) {
	// Create a dummy contract.
	addr := vm.AccountRef{}
	contract := vm.NewContract(addr, addr, big.NewInt(0), state.Gas)

	pc, ok := pcMap.evmToLfvm[state.Pc]
	if !ok {
		return nil, fmt.Errorf("unable to convert evm pc %d to lfvm pc", state.Pc)
	}

	status, err := convertCtStatusToLfvmStatus(state)
	if err != nil {
		return nil, err
	}

	code, err := convertCtCodeToLfvmCode(state)
	if err != nil {
		return nil, err
	}

	memory, err := convertCtMemoryToLfvmMemory(state)
	if err != nil {
		return nil, err
	}

	data := []byte{}

	stateDb := &conformanceTestStateDb{
		storage: state.Storage,
	}

	// Create execution context.
	ctx := context{
		evm:      &vm.EVM{StateDB: stateDb},
		pc:       int32(pc),
		stack:    convertCtStackToLfvmStack(state),
		memory:   memory,
		stateDB:  stateDb,
		status:   status,
		contract: contract,
		code:     code,
		data:     data,
		callsize: *uint256.NewInt(uint64(len(data))),
		readOnly: false,
		isBerlin: state.Revision == ct.R09_Berlin,
		isLondon: state.Revision == ct.R10_London,
	}

	err = convertCtRevisionToLfvmRevision(state.Revision, &ctx)
	if err != nil {
		return nil, err
	}

	return &ctx, nil
}

// conformanceTestStateDb is an adapter between the CT framework's storage
// representation and the StateDB interface expected by the LFVM.
type conformanceTestStateDb struct {
	storage *st.Storage
}

func (db *conformanceTestStateDb) GetCommittedState(_ common.Address, key common.Hash) common.Hash {
	k := ct.NewU256FromBytes(key[:]...)
	db.storage.MarkWarm(k)
	return db.storage.Original[k].Bytes32be()
}

func (db *conformanceTestStateDb) GetState(_ common.Address, key common.Hash) common.Hash {
	k := ct.NewU256FromBytes(key[:]...)
	db.storage.MarkWarm(k)
	return db.storage.Current[k].Bytes32be()
}

func (db *conformanceTestStateDb) SetState(_ common.Address, key common.Hash, value common.Hash) {
	k := ct.NewU256FromBytes(key[:]...)
	v := ct.NewU256FromBytes(value[:]...)
	db.storage.MarkWarm(k)
	db.storage.Current[k] = v
}

func (db *conformanceTestStateDb) SlotInAccessList(_ common.Address, key common.Hash) (addressOk bool, slotOk bool) {
	k := ct.NewU256FromBytes(key[:]...)
	return true, db.storage.IsWarm(k)
}

func (db *conformanceTestStateDb) AddSlotToAccessList(_ common.Address, key common.Hash) {
	k := ct.NewU256FromBytes(key[:]...)
	db.storage.MarkWarm(k)
}

func (db *conformanceTestStateDb) GetCode(common.Address) []byte {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) GetCodeHash(common.Address) common.Hash {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) GetCodeSize(common.Address) int {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) AddRefund(uint64) {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) SubRefund(uint64) {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) GetRefund() uint64 {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) CreateAccount(common.Address) {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) Suicide(common.Address) bool {
	panic("not implemented yet")
}

func (db *conformanceTestStateDb) AddLog(*types.Log) {
	panic("not implemented yet")
}

// -- StateDB interface methods that should not be needed ---

// The remaining methods of the conformanceTestStateDb are needed to satisfy
// the interface definition of a StateDB but are not relevant for testing the
// interpreter. These functions are used by the enclosing EVM implementation.

func (db *conformanceTestStateDb) SubBalance(common.Address, *big.Int) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) AddBalance(common.Address, *big.Int) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) GetBalance(common.Address) *big.Int {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) GetNonce(common.Address) uint64 {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) SetNonce(common.Address, uint64) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) SetCode(common.Address, []byte) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) AddressInAccessList(addr common.Address) bool {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) AddAddressToAccessList(addr common.Address) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) HasSuicided(common.Address) bool {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) Exist(common.Address) bool {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) Empty(common.Address) bool {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) PrepareAccessList(sender common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) RevertToSnapshot(int) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) Snapshot() int {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) AddPreimage(common.Hash, []byte) {
	panic("should not be needed")
}

func (db *conformanceTestStateDb) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error {
	panic("should not be needed")
}

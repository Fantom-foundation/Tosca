package geth

import (
	"math/big"
	"testing"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

////////////////////////////////////////////////////////////
// ct -> geth

func getEmptyState() *st.State {
	return st.NewState(st.NewCode([]byte{}))
}

func TestConvertToGeth_StatusCode(t *testing.T) {
	tests := map[string][]struct {
		ctStatus            st.StatusCode
		convertSuccess      bool
		gethStatusPredicate func(state *vm.GethState) bool
	}{
		"running":  {{st.Running, true, func(state *vm.GethState) bool { return !state.Halted && state.Err == nil }}},
		"stopped":  {{st.Stopped, true, func(state *vm.GethState) bool { return state.Halted && state.Err == nil }}},
		"returned": {{st.Returned, true, func(state *vm.GethState) bool { return state.Halted && state.Err == nil && state.Result != nil }}},
		"reverted": {{st.Reverted, true, func(state *vm.GethState) bool { return state.Halted && state.Err == vm.ErrExecutionReverted }}},
		"failed": {{st.Failed, true, func(state *vm.GethState) bool {
			return state.Halted && state.Err != vm.ErrExecutionReverted && state.Err != nil
		}}},
		"invalid": {{st.NumStatusCodes, false, nil}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := getEmptyState()
				state.Status = cur.ctStatus

				_, gethState, err := ConvertCtStateToGeth(state)

				if want, got := cur.convertSuccess, err == nil; want != got {
					t.Errorf("unexpected conversion error: wanted %v, got %v", want, got)
				}

				if err == nil {
					if !cur.gethStatusPredicate(gethState) {
						t.Errorf("status conversion check for %v failed", cur.ctStatus)
					}
				}
			}
		})
	}
}

func TestConvertToGeth_Revision(t *testing.T) {
	tests := map[string][]struct {
		ctRevision            ct.Revision
		convertSuccess        bool
		gethRevisionPredicate func(interpreter *gethInterpreter) bool
	}{
		"istanbul": {{ct.R07_Istanbul, true, func(interpreter *gethInterpreter) bool { return interpreter.isIstanbul() }}},
		"berlin":   {{ct.R09_Berlin, true, func(interpreter *gethInterpreter) bool { return interpreter.isBerlin() }}},
		"london":   {{ct.R10_London, true, func(interpreter *gethInterpreter) bool { return interpreter.isLondon() }}},
		// TODO "next":     {{ct.R99_UnknownNextRevision, true, func(interpreter *gethInterpreter) bool {}}},
		"invalid": {{-1, false, nil}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := getEmptyState()
				state.Revision = cur.ctRevision

				interpreter, _, err := ConvertCtStateToGeth(state)

				if want, got := cur.convertSuccess, err == nil; want != got {
					t.Errorf("unexpected conversion error: wanted %v, got %v", want, got)
				}

				if err == nil {
					if !cur.gethRevisionPredicate(interpreter) {
						t.Errorf("revision check for %v failed", cur.ctRevision)
					}
				}
			}
		})
	}
}

func TestConvertToGeth_Pc(t *testing.T) {
	tests := map[string][]struct {
		evmPc  uint16
		gethPc uint64
	}{
		"empty": {{}},
		"pos-0": {{0, 0}},
		"pos-1": {{1, 1}},
		"end":   {{0x6000, 0x6000}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := st.NewState(st.NewCode([]byte{}))
				state.Pc = cur.evmPc

				_, gethState, err := ConvertCtStateToGeth(state)

				if err != nil {
					t.Fatalf("failed to convert ct state to geth: %v", err)
				}

				if want, got := cur.gethPc, gethState.Pc; want != got {
					t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToGeth_Gas(t *testing.T) {
	state := getEmptyState()
	state.Gas = 777

	_, gethState, err := ConvertCtStateToGeth(state)

	if err != nil {
		t.Fatalf("failed to convert ct state to geth: %v", err)
	}

	if want, got := uint64(777), gethState.Contract.Gas; want != got {
		t.Errorf("unexpected gas value, wanted %v, got %v", want, got)
	}
}

func TestConvertToGeth_Code(t *testing.T) {
	tests := map[string][]struct {
		code []byte
	}{
		"empty": {{}},
		"stop":  {{[]byte{byte(ct.STOP)}}},
		"add": {{[]byte{
			byte(ct.PUSH1), 0x01,
			byte(ct.PUSH1), 0x02,
			byte(ct.ADD)},
		}},
		"jump": {{[]byte{
			byte(ct.PUSH1), 0x04,
			byte(ct.JUMP),
			byte(ct.INVALID),
			byte(ct.JUMPDEST)},
		}},
		"jumpdest": {{[]byte{
			byte(ct.PUSH3), 0x00, 0x00, 0x06,
			byte(ct.JUMP),
			byte(ct.INVALID),
			byte(ct.JUMPDEST)},
		}},
		"push2": {{[]byte{byte(ct.PUSH2), 0xBA, 0xAD}}},
		"push3": {{[]byte{byte(ct.PUSH3), 0xBA, 0xAD, 0xC0}}},
		"push31": {{[]byte{
			byte(ct.PUSH31),
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F},
		}},
		"push32": {{[]byte{
			byte(ct.PUSH32),
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0xFF},
		}},
		"invalid": {{[]byte{byte(ct.INVALID)}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				code := st.NewCode(cur.code)
				state := st.NewState(code)

				_, gethState, err := ConvertCtStateToGeth(state)

				if err != nil {
					t.Fatalf("failed to convert ct state to geth: %v", err)
				}

				want := cur.code
				got := gethState.Contract.Code

				if wantSize, gotSize := len(want), len(got); wantSize != gotSize {
					t.Fatalf("unexpected code size, wanted %d, got %d", wantSize, gotSize)
				}

				for i := 0; i < len(got); i++ {
					if wantInst, gotInst := want[i], got[i]; wantInst != gotInst {
						t.Errorf("unexpected instruction, wanted %v, got %v", wantInst, gotInst)
					}
				}
			}
		})
	}
}

func TestConvertToGeth_Stack(t *testing.T) {
	newGethStack := func(values ...ct.U256) *vm.Stack {
		stack := vm.NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.Push(&value)
		}
		return stack
	}

	tests := map[string][]struct {
		ctStack   *st.Stack
		gethStack *vm.Stack
	}{
		"empty": {{
			st.NewStack(),
			newGethStack()}},
		"one-element": {{
			st.NewStack(ct.NewU256(7)),
			newGethStack(ct.NewU256(7))}},
		"two-elements": {{
			st.NewStack(ct.NewU256(1), ct.NewU256(2)),
			newGethStack(ct.NewU256(1), ct.NewU256(2))}},
		"three-elements": {{
			st.NewStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3)),
			newGethStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3))}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := getEmptyState()
				state.Stack = cur.ctStack

				_, gethState, err := ConvertCtStateToGeth(state)

				if err != nil {
					t.Fatalf("failed to convert ct state to geth: %v", err)
				}

				if want, got := cur.gethStack.Len(), gethState.Stack.Len(); want != got {
					t.Fatalf("unexpected stack size, wanted %v, got %v", want, got)
				}

				for i := 0; i < gethState.Stack.Len(); i++ {
					want := cur.gethStack.Data()[i]
					got := gethState.Stack.Data()[i]
					if want != got {
						t.Errorf("unexpected stack value, wanted %v, got %v", want, got)
					}
				}
			}
		})
	}
}

func TestConvertToGeth_CallContext(t *testing.T) {
	state := getEmptyState()
	state.CallContext.AccountAddress = ct.Address{0xff}
	state.CallContext.OriginAddress = ct.Address{0xfe}
	state.CallContext.CallerAddress = ct.Address{0xfd}
	state.CallContext.Value = ct.NewU256(252)

	gethInterpreter, gethState, err := ConvertCtStateToGeth(state)
	if err != nil {
		t.Fatalf("failed to convert ct state to geth: %v", err)
	}

	if want, got := (common.Address{0xff}), gethState.Contract.Address(); want != got {
		t.Errorf("unexpected account address. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0xfe}), gethInterpreter.evm.Origin; want != got {
		t.Errorf("unexpected origin address. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0xfd}), gethState.Contract.CallerAddress; want != got {
		t.Errorf("unexpected caller address. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(252), gethState.Contract.Value(); want.Cmp(got) != 0 {
		t.Errorf("unexpected call value. wanted %v, got %v", want, got)
	}
}

func TestConvertToLfvm_BlockContext(t *testing.T) {
	state := getEmptyState()
	state.BlockContext.BlockNumber = 5
	state.BlockContext.CoinBase[0] = 0x06
	state.BlockContext.GasLimit = 7
	state.BlockContext.GasPrice = ct.NewU256(8)
	state.BlockContext.PrevRandao[31] = 0x0a
	state.BlockContext.TimeStamp = 11

	gethInterpreter, _, err := ConvertCtStateToGeth(state)
	if err != nil {
		t.Fatalf("failed to convert ct state to geth: %v", err)
	}

	if want, got := big.NewInt(5), gethInterpreter.evm.Context.BlockNumber; want.Cmp(got) != 0 {
		t.Errorf("unexpected block number. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0x06}), gethInterpreter.evm.Context.Coinbase; want != got {
		t.Errorf("unexpected coinbase. wanted %v, got %v", want, got)
	}
	if want, got := uint64(7), gethInterpreter.evm.Context.GasLimit; want != got {
		t.Errorf("unexpected gas limit. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(8), gethInterpreter.evm.GasPrice; want.Cmp(got) != 0 {
		t.Errorf("unexpected gas price. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(10), gethInterpreter.evm.Context.Difficulty; want.Cmp(got) != 0 {
		t.Errorf("unexpected prev randao. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(11), gethInterpreter.evm.Context.Time; want.Cmp(got) != 0 {
		t.Errorf("unexpected timestamp. wanted %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////
// geth -> ct

func getEmptyGeth(revision ct.Revision) (*gethInterpreter, *vm.GethState) {
	geth, err := getGethEvm(revision, nil)
	if err != nil {
		panic(err)
	}

	address := vm.AccountRef{}
	contract := vm.NewContract(address, address, big.NewInt(0), 0)
	contract.Code = make([]byte, 0)

	interpreterState := vm.NewGethState(contract, vm.NewMemory(), vm.NewStack(), 0)

	return geth, interpreterState
}

func TestConvertToCt_StatusCode(t *testing.T) {
	tests := map[string][]struct {
		gethStatusSetter func(state *vm.GethState)
		convertSuccess   bool
		ctStatus         st.StatusCode
	}{
		"running":  {{func(state *vm.GethState) { state.Halted = false; state.Err = nil }, true, st.Running}},
		"stopped":  {{func(state *vm.GethState) { state.Halted = true; state.Err = nil }, true, st.Stopped}},
		"returned": {{func(state *vm.GethState) { state.Halted = true; state.Err = nil; state.Result = make([]byte, 0) }, true, st.Returned}},
		"reverted": {{func(state *vm.GethState) { state.Halted = true; state.Err = vm.ErrExecutionReverted }, true, st.Reverted}},
		"failed":   {{func(state *vm.GethState) { state.Halted = true; state.Err = vm.ErrInvalidCode }, true, st.Failed}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				gethInterpreter, gethState := getEmptyGeth(ct.R07_Istanbul)
				cur.gethStatusSetter(gethState)

				state, err := ConvertGethToCtState(gethInterpreter, gethState)

				if want, got := cur.convertSuccess, err == nil; want != got {
					t.Errorf("unexpected conversion error: wanted %v, got %v", want, got)
				}

				if err == nil {
					if want, got := cur.ctStatus, state.Status; want != got {
						t.Errorf("unexpected status: wanted %v, got %v", want, got)
					}
				}
			}
		})
	}
}

func TestConvertToCt_Revision(t *testing.T) {
	tests := map[string][]struct {
		revision       ct.Revision
		convertSuccess bool
	}{
		"istanbul": {{ct.R07_Istanbul, true}},
		"berlin":   {{ct.R09_Berlin, true}},
		"london":   {{ct.R10_London, true}},
		// TODO "next":     {{ct.R99_UnknownNextRevision, false}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				interpreter, gethState := getEmptyGeth(cur.revision)

				state, err := ConvertGethToCtState(interpreter, gethState)

				if want, got := cur.convertSuccess, (err == nil); want != got {
					t.Errorf("unexpected conversion error: wanted %v, got %v", want, got)
				}

				if err == nil {
					if want, got := cur.revision, state.Revision; want != got {
						t.Errorf("failed to convert revision: wanted %v, got %v", want, got)
					}
				}
			}
		})
	}
}

func TestConvertToCt_Pc(t *testing.T) {
	tests := map[string][]struct {
		gethPc uint64
		evmPc  uint16
	}{
		"empty": {{}},
		"pos-0": {{0, 0}},
		"pos-1": {{1, 1}},
		"end":   {{0x6000, 0x6000}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				interpreter, gethState := getEmptyGeth(ct.R07_Istanbul)

				gethState.Pc = cur.gethPc

				state, err := ConvertGethToCtState(interpreter, gethState)

				if err != nil {
					t.Fatalf("failed to convert geth to ct state: %v", err)
				}

				if want, got := cur.evmPc, state.Pc; want != got {
					t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Gas(t *testing.T) {
	interpreter, gethState := getEmptyGeth(ct.R07_Istanbul)
	gethState.Contract.Gas = 777

	state, err := ConvertGethToCtState(interpreter, gethState)

	if err != nil {
		t.Fatalf("failed to convert geth to ct state: %v", err)
	}

	if want, got := uint64(777), state.Gas; want != got {
		t.Errorf("unexpected gas value, wanted %v, got %v", want, got)
	}
}

func TestConvertToCt_Stack(t *testing.T) {
	newGethStack := func(values ...ct.U256) *vm.Stack {
		stack := vm.NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.Push(&value)
		}
		return stack
	}

	tests := map[string][]struct {
		gethStack *vm.Stack
		ctStack   *st.Stack
	}{
		"empty": {{
			newGethStack(),
			st.NewStack()}},
		"one-element": {{
			newGethStack(ct.NewU256(7)),
			st.NewStack(ct.NewU256(7))}},
		"two-elements": {{
			newGethStack(ct.NewU256(1), ct.NewU256(2)),
			st.NewStack(ct.NewU256(1), ct.NewU256(2))}},
		"three-elements": {{
			newGethStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3)),
			st.NewStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3))}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				interpreter, gethState := getEmptyGeth(ct.R07_Istanbul)
				gethState.Stack = cur.gethStack

				state, err := ConvertGethToCtState(interpreter, gethState)

				if err != nil {
					t.Fatalf("failed to convert lfvm context to ct state: %v", err)
				}

				want := cur.ctStack
				got := state.Stack

				diffs := got.Diff(want)

				for _, diff := range diffs {
					t.Errorf("%s", diff)
				}
			}
		})
	}
}

func TestConvertToCt_CallContext(t *testing.T) {
	interpreter, gethState := getEmptyGeth(ct.R07_Istanbul)
	objAddress := vm.AccountRef{0xff}
	callerAddress := vm.AccountRef{0xfe}
	contract := vm.NewContract(callerAddress, objAddress, big.NewInt(252), 0)
	gethState.Contract = contract
	interpreter.evm.Origin = common.Address{0xfd}

	state, err := ConvertGethToCtState(interpreter, gethState)
	if err != nil {
		t.Fatalf("failed to convert geth to ct state: %v", err)
	}

	if want, got := (ct.Address{0xff}), state.CallContext.AccountAddress; want != got {
		t.Errorf("unexpected account address value, wanted %v, got %v", want, got)
	}
	if want, got := (ct.Address{0xfe}), state.CallContext.CallerAddress; want != got {
		t.Errorf("unexpected caller address value, wanted %v, got %v", want, got)
	}
	if want, got := (ct.Address{0xfd}), state.CallContext.OriginAddress; want != got {
		t.Errorf("unexpected origin address value, wanted %v, got %v", want, got)
	}
	if want, got := ct.NewU256(252), state.CallContext.Value; !want.Eq(got) {
		t.Errorf("unexpected call value. wanted %v, got %v", want, got)
	}
}

func TestConvertToCt_BlockContext(t *testing.T) {
	interpreter, gethState := getEmptyGeth(ct.R07_Istanbul)
	interpreter.evm.Context.BlockNumber = big.NewInt(255)
	interpreter.evm.Context.Coinbase = vm.AccountRef{0xfe}.Address()
	interpreter.evm.Context.GasLimit = uint64(253)
	interpreter.evm.TxContext.GasPrice = big.NewInt(252)
	interpreter.evm.Context.Difficulty = big.NewInt(0).SetBytes([]byte{251})
	interpreter.evm.Context.Time = big.NewInt(250)

	state, err := ConvertGethToCtState(interpreter, gethState)
	if err != nil {
		t.Fatalf("failed to convert geth to ct state: %v", err)
	}

	if want, got := uint64(255), state.BlockContext.BlockNumber; want != got {
		t.Errorf("unexpected block number, wanted %v, got %v", want, got)
	}
	if want, got := (ct.Address{0xfe}), state.BlockContext.CoinBase; want != got {
		t.Errorf("unexpected coinbase, wanted %v, got %v", want, got)
	}
	if want, got := uint64(253), state.BlockContext.GasLimit; want != got {
		t.Errorf("unexpected gas limit, wanted %v, got %v", want, got)
	}
	if want, got := ct.NewU256(252), state.BlockContext.GasPrice; !want.Eq(got) {
		t.Errorf("unexpected gas price, wanted %v, got %v", want, got)
	}
	if want, got := ([32]byte{0xfb}), state.BlockContext.PrevRandao; want != got {
		t.Errorf("unexpected prev randao, wanted %v, got %v", want, got)
	}
	if want, got := uint64(250), state.BlockContext.TimeStamp; want != got {
		t.Errorf("unexpected timestamp, wanted %v, got %v", want, got)
	}

}

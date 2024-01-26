package lfvm

import (
	"math/big"
	"testing"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

////////////////////////////////////////////////////////////
// ct -> lfvm

func getEmptyState() *st.State {
	return st.NewState(st.NewCode([]byte{}))
}

func getByteCodeFromState(state *st.State) []byte {
	code := make([]byte, state.Code.Length())
	state.Code.CopyTo(code)
	return code
}

func TestConvertToLfvm_StatusCode(t *testing.T) {
	state := getEmptyState()
	state.Status = st.Stopped

	pcMap, err := GenPcMapWithoutSuperInstructions(getByteCodeFromState(state))
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	ctx, err := ConvertCtStateToLfvmContext(state, pcMap)

	if err != nil {
		t.Fatalf("failed to convert ct state to lfvm context: %v", err)
	}

	if want, got := STOPPED, ctx.status; want != got {
		t.Errorf("unexpected status, wanted %v, got %v", want, got)
	}
}

func TestConvertToLfvm_InvalidStatusCode(t *testing.T) {
	state := getEmptyState()
	state.Status = st.NumStatusCodes

	pcMap, err := GenPcMapWithoutSuperInstructions(getByteCodeFromState(state))
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	ctx, err := ConvertCtStateToLfvmContext(state, pcMap)

	if err == nil {
		t.Errorf("expected invalid status, but got: %v", ctx.status)
	}
}

func TestConvertToLfvm_Revision(t *testing.T) {
	tests := map[string][]struct {
		ctRevision            ct.Revision
		convertSuccess        bool
		lfvmRevisionPredicate func(ctx *context) bool
	}{
		"istanbul": {{ct.R07_Istanbul, true, func(ctx *context) bool { return !ctx.isBerlin && !ctx.isLondon }}},
		"berlin":   {{ct.R09_Berlin, true, func(ctx *context) bool { return ctx.isBerlin && !ctx.isLondon }}},
		"london":   {{ct.R10_London, true, func(ctx *context) bool { return ctx.isBerlin && ctx.isLondon }}},
		// TODO "next":     {{ct.R99_UnknownNextRevision, true, func(ctx *context) bool { }}},
		"invalid": {{-1, false, nil}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := getEmptyState()
				state.Revision = cur.ctRevision

				pcMap, err := GenPcMapWithoutSuperInstructions(getByteCodeFromState(state))
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}

				ctx, err := ConvertCtStateToLfvmContext(state, pcMap)

				if want, got := cur.convertSuccess, (err == nil); want != got {
					t.Errorf("unexpected conversion error: wanted %v, got %v", want, got)
				}

				if err == nil {
					if !cur.lfvmRevisionPredicate(ctx) {
						t.Errorf("revision check for %v failed", cur.ctRevision)
					}
				}
			}
		})
	}
}

func TestConvertToLfvm_Pc(t *testing.T) {
	tests := map[string][]struct {
		evmCode []byte
		evmPc   uint16
		lfvmPc  uint16
	}{
		"empty":        {{}},
		"pos-0":        {{[]byte{byte(ct.STOP)}, 0, 0}},
		"pos-1":        {{[]byte{byte(ct.STOP), byte(ct.STOP), byte(ct.STOP)}, 1, 1}},
		"one-past-end": {{[]byte{byte(ct.STOP)}, 1, 1}},
		"shifted": {{[]byte{
			byte(ct.PUSH1), 0x01,
			byte(ct.PUSH1), 0x02,
			byte(ct.ADD)}, 2, 1}},
		"jumpdest": {{[]byte{
			byte(ct.PUSH3), 0x00, 0x00, 0x06,
			byte(ct.JUMP),
			byte(ct.INVALID),
			byte(ct.JUMPDEST)},
			6, 6}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				code := st.NewCode(cur.evmCode)
				state := st.NewState(code)
				state.Pc = cur.evmPc

				pcMap, err := GenPcMapWithoutSuperInstructions(getByteCodeFromState(state))
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}

				ctx, err := ConvertCtStateToLfvmContext(state, pcMap)

				if err != nil {
					t.Fatalf("failed to convert ct state to lfvm context: %v", err)
				}

				if want, got := cur.lfvmPc, uint16(ctx.pc); want != got {
					t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToLfvm_Gas(t *testing.T) {
	state := getEmptyState()
	state.Gas = 777

	pcMap, err := GenPcMapWithoutSuperInstructions(getByteCodeFromState(state))
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	ctx, err := ConvertCtStateToLfvmContext(state, pcMap)

	if err != nil {
		t.Fatalf("failed to convert ct state to lfvm context: %v", err)
	}

	if want, got := uint64(777), ctx.contract.Gas; want != got {
		t.Errorf("unexpected gas value, wanted %v, got %v", want, got)
	}
}

func TestConvertToLfvm_Code(t *testing.T) {
	tests := map[string][]struct {
		evmCode  []byte
		lfvmCode Code
	}{
		"empty": {{}},
		"stop":  {{[]byte{byte(ct.STOP)}, Code{Instruction{STOP, 0x0000}}}},
		"add": {{[]byte{
			byte(ct.PUSH1), 0x01,
			byte(ct.PUSH1), 0x02,
			byte(ct.ADD)},
			Code{Instruction{PUSH1, 0x0100},
				Instruction{PUSH1, 0x0200},
				Instruction{ADD, 0x0000}}}},
		"jump": {{[]byte{
			byte(ct.PUSH1), 0x04,
			byte(ct.JUMP),
			byte(ct.INVALID),
			byte(ct.JUMPDEST)},
			Code{Instruction{PUSH1, 0x0400},
				Instruction{JUMP, 0x0000},
				Instruction{INVALID, 0x0000},
				Instruction{JUMP_TO, 0x0004},
				Instruction{JUMPDEST, 0x0000}}}},
		"jumpdest": {{[]byte{
			byte(ct.PUSH3), 0x00, 0x00, 0x06,
			byte(ct.JUMP),
			byte(ct.INVALID),
			byte(ct.JUMPDEST)},
			Code{Instruction{PUSH3, 0x0000},
				Instruction{DATA, 0x0600},
				Instruction{JUMP, 0x0000},
				Instruction{INVALID, 0x0000},
				Instruction{JUMP_TO, 0x0006},
				Instruction{NOOP, 0x0000},
				Instruction{JUMPDEST, 0x0000}}}},
		"push2": {{[]byte{byte(ct.PUSH2), 0xBA, 0xAD}, Code{Instruction{PUSH2, 0xBAAD}}}},
		"push3": {{[]byte{byte(ct.PUSH3), 0xBA, 0xAD, 0xC0}, Code{Instruction{PUSH3, 0xBAAD}, Instruction{DATA, 0xC000}}}},
		"push31": {{[]byte{
			byte(ct.PUSH31),
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F},
			Code{Instruction{PUSH31, 0x0102},
				Instruction{DATA, 0x0304},
				Instruction{DATA, 0x0506},
				Instruction{DATA, 0x0708},
				Instruction{DATA, 0x090A},
				Instruction{DATA, 0x0B0C},
				Instruction{DATA, 0x0D0E},
				Instruction{DATA, 0x0F10},
				Instruction{DATA, 0x1112},
				Instruction{DATA, 0x1314},
				Instruction{DATA, 0x1516},
				Instruction{DATA, 0x1718},
				Instruction{DATA, 0x191A},
				Instruction{DATA, 0x1B1C},
				Instruction{DATA, 0x1D1E},
				Instruction{DATA, 0x1F00}}}},
		"push32": {{[]byte{
			byte(ct.PUSH32),
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0xFF},
			Code{Instruction{PUSH32, 0x0102},
				Instruction{DATA, 0x0304},
				Instruction{DATA, 0x0506},
				Instruction{DATA, 0x0708},
				Instruction{DATA, 0x090A},
				Instruction{DATA, 0x0B0C},
				Instruction{DATA, 0x0D0E},
				Instruction{DATA, 0x0F10},
				Instruction{DATA, 0x1112},
				Instruction{DATA, 0x1314},
				Instruction{DATA, 0x1516},
				Instruction{DATA, 0x1718},
				Instruction{DATA, 0x191A},
				Instruction{DATA, 0x1B1C},
				Instruction{DATA, 0x1D1E},
				Instruction{DATA, 0x1FFF}}}},
		"invalid": {{[]byte{byte(ct.INVALID)}, Code{Instruction{INVALID, 0x0000}}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				code := st.NewCode(cur.evmCode)
				state := st.NewState(code)

				pcMap, err := GenPcMapWithoutSuperInstructions(getByteCodeFromState(state))
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}

				ctx, err := ConvertCtStateToLfvmContext(state, pcMap)

				if err != nil {
					t.Fatalf("failed to convert ct state to lfvm context: %v", err)
				}

				want := cur.lfvmCode
				got := ctx.code

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

func TestConvertToLfvm_Stack(t *testing.T) {
	newLfvmStack := func(values ...ct.U256) *Stack {
		stack := NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.push(&value)
		}
		return stack
	}

	tests := map[string][]struct {
		ctStack   *st.Stack
		lfvmStack *Stack
	}{
		"empty": {{
			st.NewStack(),
			newLfvmStack()}},
		"one-element": {{
			st.NewStack(ct.NewU256(7)),
			newLfvmStack(ct.NewU256(7))}},
		"two-elements": {{
			st.NewStack(ct.NewU256(1), ct.NewU256(2)),
			newLfvmStack(ct.NewU256(1), ct.NewU256(2))}},
		"three-elements": {{
			st.NewStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3)),
			newLfvmStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3))}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := getEmptyState()
				state.Stack = cur.ctStack

				pcMap, err := GenPcMapWithoutSuperInstructions(getByteCodeFromState(state))
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}

				ctx, err := ConvertCtStateToLfvmContext(state, pcMap)

				if err != nil {
					t.Fatalf("failed to convert ct state to lfvm context: %v", err)
				}

				if want, got := cur.lfvmStack.len(), ctx.stack.len(); want != got {
					t.Fatalf("unexpected stack size, wanted %v, got %v", want, got)
				}

				for i := 0; i < ctx.stack.len(); i++ {
					want := cur.lfvmStack.Data()[i]
					got := ctx.stack.Data()[i]
					if want != got {
						t.Errorf("unexpected stack value, wanted %v, got %v", want, got)
					}
				}
			}
		})
	}
}

func TestConvertToLfvm_callContext(t *testing.T) {
	state := getEmptyState()
	state.CallContext.AccountAddress = ct.Address{0xff}
	state.CallContext.OriginAddress = ct.Address{0xfe}
	state.CallContext.CallerAddress = ct.Address{0xfd}
	state.CallContext.Value = ct.NewU256(252)

	code := []byte{}
	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	context, err := ConvertCtStateToLfvmContext(state, pcMap)

	if err != nil {
		t.Fatalf("failed to convert ct state to lfvm context: %v", err)
	}

	if want, got := (common.Address{0xff}), context.contract.Address(); want != got {
		t.Errorf("unexpected account address. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0xfe}), context.evm.Origin; want != got {
		t.Errorf("unexpected origin address. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0xfd}), context.contract.CallerAddress; want != got {
		t.Errorf("unexpected caller address. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(252), context.contract.Value(); want.Cmp(got) != 0 {
		t.Errorf("unexpected call value. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(252), context.contract.Value(); want.Cmp(got) != 0 {
		t.Errorf("unexpected call value. wanted %v, got %v", want, got)
	}

}

func TestConvertToLfvm_BlockContext(t *testing.T) {
	state := getEmptyState()
	state.BlockContext.BlockNumber = 5
	state.BlockContext.CoinBase[0] = 0x06
	state.BlockContext.GasLimit = 7
	state.BlockContext.GasPrice = ct.NewU256(8)
	state.BlockContext.PrevRandao = ct.NewU256(9)
	state.BlockContext.TimeStamp = 10

	code := []byte{}
	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	context, err := ConvertCtStateToLfvmContext(state, pcMap)
	if err != nil {
		t.Fatalf("failed to convert ct state to lfvm context: %v", err)
	}

	if want, got := big.NewInt(5), context.evm.Context.BlockNumber; want.Cmp(got) != 0 {
		t.Errorf("unexpected block number. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0x06}), context.evm.Context.Coinbase; want != got {
		t.Errorf("unexpected coinbase. wanted %v, got %v", want, got)
	}
	if want, got := uint64(7), context.evm.Context.GasLimit; want != got {
		t.Errorf("unexpected gas limit. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(8), context.evm.GasPrice; want.Cmp(got) != 0 {
		t.Errorf("unexpected gas price. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(9), context.evm.Context.Difficulty; want.Cmp(got) != 0 {
		t.Errorf("unexpected prev randao. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(10), context.evm.Context.Time; want.Cmp(got) != 0 {
		t.Errorf("unexpected timestamp. wanted %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////
// lfvm -> ct

func getContextWithEvmCode(code *st.Code) (*context, error) {
	byteCode := make([]byte, code.Length())
	code.CopyTo(byteCode)
	lfvmCode, err := convert(byteCode, false)
	if err != nil {
		return nil, err
	}
	data := make([]byte, 0)
	ctx := getContext(lfvmCode, data, 0, nil, 0, false, false)
	return &ctx, nil
}

func TestConvertToCt_StatusCode(t *testing.T) {
	ctx := getEmptyContext()
	ctx.status = STOPPED
	code := []byte{}

	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	state, err := ConvertLfvmContextToCtState(&ctx, st.NewCode(code), pcMap)

	if err != nil {
		t.Fatalf("failed to convert lfvm context to ct state: %v", err)
	}

	if want, got := st.Stopped, state.Status; want != got {
		t.Errorf("unexpected status, wanted %v, got %v", want, got)
	}
}

func TestConvertToCt_InvalidStatusCode(t *testing.T) {
	ctx := getEmptyContext()
	ctx.status = 0xFF
	code := []byte{}

	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	state, err := ConvertLfvmContextToCtState(&ctx, st.NewCode(code), pcMap)

	if err == nil {
		t.Errorf("expected invalid status, but got: %v", state.Status)
	}
}

func TestConvertToCt_Revision(t *testing.T) {
	tests := map[string][]struct {
		lfvmRevisionSetter func(ctx *context)
		convertSuccess     bool
		ctRevision         ct.Revision
	}{
		"istanbul": {{func(ctx *context) { ctx.isBerlin = false; ctx.isLondon = false }, true, ct.R07_Istanbul}},
		"berlin":   {{func(ctx *context) { ctx.isBerlin = true; ctx.isLondon = false }, true, ct.R09_Berlin}},
		"london":   {{func(ctx *context) { ctx.isBerlin = false; ctx.isLondon = true }, true, ct.R10_London}},
		// TODO "next":     {{func(ctx *context) {  }, true, ct.R99_UnknownNextRevision}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				ctx := getEmptyContext()
				cur.lfvmRevisionSetter(&ctx)
				code := []byte{}

				pcMap, err := GenPcMapWithoutSuperInstructions(code)
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}

				state, err := ConvertLfvmContextToCtState(&ctx, st.NewCode(code), pcMap)

				if want, got := cur.convertSuccess, (err == nil); want != got {
					t.Errorf("unexpected conversion error: wanted %v, got %v", want, got)
				}

				if err == nil {
					if want, got := cur.ctRevision, state.Revision; want != got {
						t.Errorf("failed to convert revision: wanted %v, got %v", want, got)
					}
				}
			}
		})
	}
}

func TestConvertToCt_Pc(t *testing.T) {
	tests := map[string][]struct {
		evmCode []byte
		lfvmPc  uint16
		evmPc   uint16
	}{
		"empty":        {{}},
		"pos-0":        {{[]byte{byte(ct.STOP)}, 0, 0}},
		"pos-1":        {{[]byte{byte(ct.STOP), byte(ct.STOP), byte(ct.STOP)}, 1, 1}},
		"one-past-end": {{[]byte{byte(ct.STOP)}, 1, 1}},
		"shifted": {{[]byte{
			byte(ct.PUSH1), 0x01,
			byte(ct.PUSH1), 0x02,
			byte(ct.ADD)}, 1, 2}},
		"jumpdest": {{[]byte{
			byte(ct.PUSH3), 0x00, 0x00, 0x06,
			byte(ct.JUMP),
			byte(ct.INVALID),
			byte(ct.JUMPDEST)},
			6, 6}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				origCode := st.NewCode(cur.evmCode)
				ctx, err := getContextWithEvmCode(origCode)
				if err != nil {
					t.Fatalf("failed to create lfvm context with code: %v", err)
				}

				ctx.pc = int32(cur.lfvmPc)

				pcMap, err := GenPcMapWithoutSuperInstructions(cur.evmCode)
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}

				state, err := ConvertLfvmContextToCtState(ctx, origCode, pcMap)

				if err != nil {
					t.Fatalf("failed to convert lfvm context to ct state: %v", err)
				}

				if want, got := cur.evmPc, state.Pc; want != got {
					t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Gas(t *testing.T) {
	ctx := getEmptyContext()
	ctx.contract.Gas = 777
	code := []byte{}

	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	state, err := ConvertLfvmContextToCtState(&ctx, st.NewCode(code), pcMap)

	if err != nil {
		t.Fatalf("failed to convert lfvm context to ct state: %v", err)
	}

	if want, got := uint64(777), state.Gas; want != got {
		t.Errorf("unexpected gas value, wanted %v, got %v", want, got)
	}
}

func TestConvertToCt_Stack(t *testing.T) {
	newLfvmStack := func(values ...ct.U256) *Stack {
		stack := NewStack()
		for i := 0; i < len(values); i++ {
			value := values[i].Uint256()
			stack.push(&value)
		}
		return stack
	}

	tests := map[string][]struct {
		lfvmStack *Stack
		ctStack   *st.Stack
	}{
		"empty": {{
			newLfvmStack(),
			st.NewStack()}},
		"one-element": {{
			newLfvmStack(ct.NewU256(7)),
			st.NewStack(ct.NewU256(7))}},
		"two-elements": {{
			newLfvmStack(ct.NewU256(1), ct.NewU256(2)),
			st.NewStack(ct.NewU256(1), ct.NewU256(2))}},
		"three-elements": {{
			newLfvmStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3)),
			st.NewStack(ct.NewU256(1), ct.NewU256(2), ct.NewU256(3))}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				ctx := getEmptyContext()
				ctx.stack = cur.lfvmStack
				code := []byte{}

				pcMap, err := GenPcMapWithoutSuperInstructions(code)
				if err != nil {
					t.Fatalf("failed to generate pc map: %v", err)
				}

				state, err := ConvertLfvmContextToCtState(&ctx, st.NewCode(code), pcMap)

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
	ctx := getEmptyContext()
	objAddress := vm.AccountRef{0xff}
	callerAddress := vm.AccountRef{0xfe}
	contract := vm.NewContract(callerAddress, objAddress, big.NewInt(252), 0)
	ctx.contract = contract
	ctx.evm.Origin = common.Address{0xfd}
	code := []byte{}

	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	state, err := ConvertLfvmContextToCtState(&ctx, st.NewCode(code), pcMap)

	if err != nil {
		t.Fatalf("failed to convert lfvm context to ct state: %v", err)
	}

	if want, got := (ct.Address{0xff}), state.CallContext.AccountAddress; want != got {
		t.Errorf("unexpected account address, wanted %v, got %v", want, got)
	}
	if want, got := (ct.Address{0xfe}), state.CallContext.CallerAddress; want != got {
		t.Errorf("unexpected caller address, wanted %v, got %v", want, got)
	}
	if want, got := (ct.Address{0xfd}), state.CallContext.OriginAddress; want != got {
		t.Errorf("unexpected origin address, wanted %v, got %v", want, got)
	}
	if want, got := ct.NewU256(252), state.CallContext.Value; !want.Eq(got) {
		t.Errorf("unexpected call value. wanted %v, got %v", want, got)
	}
}

func TestConvertToCt_BlockContext(t *testing.T) {
	ctx := getEmptyContext()

	newBlockNumber := big.NewInt(255)
	newCoinBase := vm.AccountRef{0xfe}
	newGasLimit := uint64(253)
	newGasPrice := big.NewInt(252)
	newDifficulty := big.NewInt(251)
	newTimestamp := big.NewInt(250)

	ctx.evm = &vm.EVM{
		Context: vm.BlockContext{
			BlockNumber: newBlockNumber,
			Coinbase:    newCoinBase.Address(),
			GasLimit:    newGasLimit,
			Difficulty:  newDifficulty,
			Time:        newTimestamp,
		},
		TxContext: vm.TxContext{
			GasPrice: newGasPrice,
		}}

	code := []byte{}

	pcMap, err := GenPcMapWithoutSuperInstructions(code)
	if err != nil {
		t.Fatalf("failed to generate pc map: %v", err)
	}

	state, err := ConvertLfvmContextToCtState(&ctx, st.NewCode(code), pcMap)
	if err != nil {
		t.Fatalf("failed to convert lfvm context to ct state: %v", err)
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

	if want, got := ct.NewU256(251), state.BlockContext.PrevRandao; !want.Eq(got) {
		t.Errorf("unexpected prev randao, wanted %v, got %v", want, got)
	}
	if want, got := uint64(250), state.BlockContext.TimeStamp; want != got {
		t.Errorf("unexpected timestamp, wanted %v, got %v", want, got)
	}

}

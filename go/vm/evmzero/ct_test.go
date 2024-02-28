package evmzero

import (
	"errors"
	"math"
	"math/big"
	"slices"
	"testing"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/ethereum/evmc/v10/bindings/go/evmc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

////////////////////////////////////////////////////////////
// ct -> evmzero

func getEmptyState() *st.State {
	return st.NewState(st.NewCode([]byte{}))
}

func TestConvertToEvmzero_StatusCode(t *testing.T) {
	tests := map[string][]struct {
		ctStatus       st.StatusCode
		evmcStatus     evmc.StepStatus
		convertSuccess bool
	}{
		"running":  {{st.Running, evmc.Running, true}},
		"stopped":  {{st.Stopped, evmc.Stopped, true}},
		"returned": {{st.Returned, evmc.Returned, true}},
		"reverted": {{st.Reverted, evmc.Reverted, true}},
		"failed":   {{st.Failed, evmc.Failed, true}},
		"error":    {{st.NumStatusCodes, evmc.Failed, false}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				evmcStatus, err := convertCtStatusToEvmcStatus(cur.ctStatus)

				if cur.convertSuccess && err != nil {
					t.Fatalf("unexpected conversion error: %v", err)
				}

				if !cur.convertSuccess && err == nil {
					t.Fatalf("expected conversion error, but got none")
				}

				if want, got := cur.evmcStatus, evmcStatus; cur.convertSuccess && want != got {
					t.Errorf("unexpected status: wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToEvmzero_Revision(t *testing.T) {
	tests := map[string][]struct {
		ctRevision     ct.Revision
		evmcRevision   evmc.Revision
		convertSuccess bool
	}{
		"istanbul": {{ct.R07_Istanbul, evmc.Istanbul, true}},
		"berlin":   {{ct.R09_Berlin, evmc.Berlin, true}},
		"london":   {{ct.R10_London, evmc.London, true}},
		"next":     {{ct.R99_UnknownNextRevision, evmc.MaxRevision, true}},
		"error":    {{ct.R99_UnknownNextRevision + 1, evmc.MaxRevision, false}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				evmcRevision, err := convertCtRevisionToEvmcRevision(cur.ctRevision)

				if cur.convertSuccess && err != nil {
					t.Fatalf("unexpected conversion error: %v", err)
				}

				if !cur.convertSuccess && err == nil {
					t.Fatalf("expected conversion error, but got none")
				}

				if want, got := cur.evmcRevision, evmcRevision; cur.convertSuccess && want != got {
					t.Errorf("unexpected revision: wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToEvmzero_Pc(t *testing.T) {
	tests := map[string][]struct {
		evmPc     uint16
		evmzeroPc uint64
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

				evmzeroEvaluation := CreateEvaluation(state)

				if len(evmzeroEvaluation.issues) > 0 {
					t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
				}

				if want, got := cur.evmzeroPc, evmzeroEvaluation.pc; want != got {
					t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToEvmzero_Gas(t *testing.T) {
	tests := map[string][]struct {
		ctGas               uint64
		evmzeroGas          uint64
		evmzeroGasReduction uint64
	}{
		"zero":           {{0, 0, 0}},
		"non-zero":       {{777, 777, 0}},
		"int64-max":      {{math.MaxInt64, math.MaxInt64, 0}},
		"over-int64-max": {{math.MaxInt64 + 777, math.MaxInt64, 777}},
		"uint64-max":     {{math.MaxUint64, math.MaxInt64, math.MaxInt64 + 1}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := st.NewState(st.NewCode([]byte{}))
				state.Gas = cur.ctGas

				evmzeroEvaluation := CreateEvaluation(state)

				if len(evmzeroEvaluation.issues) > 0 {
					t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
				}

				gas := evmzeroEvaluation.contract.Gas
				reduction := evmzeroEvaluation.gasReduction
				total := gas + reduction

				if want, got := cur.evmzeroGas, gas; want != got {
					t.Errorf("unexpected gas value, wanted %v, got %v", want, got)
				}

				if want, got := cur.evmzeroGasReduction, reduction; want != got {
					t.Errorf("unexpected gas reduction value, wanted %v, got %v", want, got)
				}

				if want, got := cur.ctGas, total; want != got {
					t.Errorf("unexpected total gas value, wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToEvmzero_GasRefund(t *testing.T) {
	tests := map[string][]struct {
		ctGasRefund               uint64
		evmzeroGasRefund          uint64
		evmzeroGasRefundReduction uint64
	}{
		"zero":           {{0, 0, 0}},
		"non-zero":       {{777, 777, 0}},
		"int64-max":      {{math.MaxInt64, math.MaxInt64, 0}},
		"over-int64-max": {{math.MaxInt64 + 777, math.MaxInt64, 777}},
		"uint64-max":     {{math.MaxUint64, math.MaxInt64, math.MaxInt64 + 1}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := st.NewState(st.NewCode([]byte{}))
				state.GasRefund = cur.ctGasRefund

				evmzeroEvaluation := CreateEvaluation(state)

				if len(evmzeroEvaluation.issues) > 0 {
					t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
				}

				gasRefund := evmzeroEvaluation.gasRefund
				reduction := evmzeroEvaluation.gasRefundReduction
				total := gasRefund + reduction

				if want, got := cur.evmzeroGasRefund, gasRefund; want != got {
					t.Errorf("unexpected gas refund value, wanted %v, got %v", want, got)
				}

				if want, got := cur.evmzeroGasRefundReduction, reduction; want != got {
					t.Errorf("unexpected gas refund reduction value, wanted %v, got %v", want, got)
				}

				if want, got := cur.ctGasRefund, total; want != got {
					t.Errorf("unexpected total gas refund value, wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToEvmzero_Code(t *testing.T) {
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

				convertedCode := convertCtCodeToEvmcCode(state.Code)

				want := cur.code
				got := convertedCode

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

func TestConvertToEvmzero_Stack(t *testing.T) {
	tests := map[string][]struct {
		ctStack   *st.Stack
		evmcStack []byte
	}{
		"empty": {{
			st.NewStack(),
			[]byte{}}},
		"one-element": {{
			st.NewStack(ct.NewU256(0xAB000000000000CD, 0xBA000000000000AD)),
			[]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xAB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xCD,
				0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
			}}},
		"two-elements": {{
			st.NewStack(
				ct.NewU256(0xDE000000000000AD, 0xBE000000000000EF, 0, 0),
				ct.NewU256(0xAB000000000000CD, 0xBA000000000000AD)),
			[]byte{
				0xDE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
				0xBE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xEF,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xAB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xCD,
				0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
			}}},
		"three-elements": {{
			st.NewStack(
				ct.NewU256(0x0001020304050607, 0x0809101112131415, 0x1617181920212223, 0x2425262728293031),
				ct.NewU256(0xDE000000000000AD, 0xBE000000000000EF, 0x0000000000000000, 0x0000000000000000),
				ct.NewU256(0xAB000000000000CD, 0xBA000000000000AD)),
			[]byte{
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
				0x16, 0x17, 0x18, 0x19, 0x20, 0x21, 0x22, 0x23,
				0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30, 0x31,

				0xDE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
				0xBE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xEF,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xAB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xCD,
				0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
			}}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				state := getEmptyState()
				state.Stack = cur.ctStack

				evmcStack := convertCtStackToEvmcStack(state.Stack)

				if want, got := len(cur.evmcStack), len(evmcStack); want != got {
					t.Fatalf("unexpected stack size, wanted %v, got %v", want, got)
				}

				for i := 0; i < len(evmcStack); i++ {
					want := cur.evmcStack[i]
					got := evmcStack[i]
					if want != got {
						t.Errorf("unexpected stack value, wanted 0x%02x, got 0x%02x", want, got)
					}
				}
			}
		})
	}
}

func TestConvertToEvmzero_CallContext(t *testing.T) {
	state := getEmptyState()
	state.CallContext.AccountAddress = ct.Address{0xff}
	state.CallContext.OriginAddress = ct.Address{0xfe}
	state.CallContext.CallerAddress = ct.Address{0xfd}
	state.CallContext.Value = ct.NewU256(252)

	evmzeroEvaluation := CreateEvaluation(state)
	if len(evmzeroEvaluation.issues) > 0 {
		t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
	}

	if want, got := (common.Address{0xff}), evmzeroEvaluation.contract.Address(); want != got {
		t.Errorf("unexpected account address. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0xfe}), evmzeroEvaluation.evmzero.evm.Origin; want != got {
		t.Errorf("unexpected origin address. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0xfd}), evmzeroEvaluation.contract.CallerAddress; want != got {
		t.Errorf("unexpected caller address. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(252), evmzeroEvaluation.contract.Value(); want.Cmp(got) != 0 {
		t.Errorf("unexpected call value. wanted %v, got %v", want, got)
	}
}

func TestConvertToEvmzero_BlockContext(t *testing.T) {
	state := getEmptyState()
	state.BlockContext.BaseFee = ct.NewU256(4)
	state.BlockContext.BlockNumber = 5
	state.BlockContext.ChainID = ct.NewU256(11)
	state.BlockContext.CoinBase = ct.Address{0x06}
	state.BlockContext.GasLimit = 7
	state.BlockContext.GasPrice = ct.NewU256(8)
	state.BlockContext.Difficulty = ct.NewU256(9)
	state.BlockContext.TimeStamp = 10

	evmzeroEvaluation := CreateEvaluation(state)
	if len(evmzeroEvaluation.issues) > 0 {
		t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
	}

	if want, got := big.NewInt(4), evmzeroEvaluation.evmzero.evm.Context.BaseFee; want.Cmp(got) != 0 {
		t.Errorf("unexpected base fee. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(5), evmzeroEvaluation.evmzero.evm.Context.BlockNumber; want.Cmp(got) != 0 {
		t.Errorf("unexpected block number. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(11), evmzeroEvaluation.evmzero.chainConfig.ChainID; want.Cmp(got) != 0 {
		t.Errorf("unexpected chain ID. wanted %v, got %v", want, got)
	}
	if want, got := (common.Address{0x06}), evmzeroEvaluation.evmzero.evm.Context.Coinbase; want != got {
		t.Errorf("unexpected coinbase. wanted %v, got %v", want, got)
	}
	if want, got := uint64(7), evmzeroEvaluation.evmzero.evm.Context.GasLimit; want != got {
		t.Errorf("unexpected gas limit. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(8), evmzeroEvaluation.evmzero.evm.GasPrice; want.Cmp(got) != 0 {
		t.Errorf("unexpected gas price. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(9), evmzeroEvaluation.evmzero.evm.Context.Difficulty; want.Cmp(got) != 0 {
		t.Errorf("unexpected difficulty. wanted %v, got %v", want, got)
	}
	if want, got := big.NewInt(10), evmzeroEvaluation.evmzero.evm.Context.Time; want.Cmp(got) != 0 {
		t.Errorf("unexpected timestamp. wanted %v, got %v", want, got)
	}
}

func TestConvertToEvmzero_CallData(t *testing.T) {
	state := getEmptyState()
	state.CallData = []byte{1}

	evmzeroEvaluation := CreateEvaluation(state)
	if len(evmzeroEvaluation.issues) > 0 {
		t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
	}

	if want, got := state.CallData, evmzeroEvaluation.input; !slices.Equal(want, got) {
		t.Errorf("unexpected calldata value. wanted %v, got %v", want, got)
	}
}

////////////////////////////////////////////////////////////
// evmzero -> ct

func TestConvertToCt_StatusCode(t *testing.T) {
	tests := map[string][]struct {
		evmcStatus     evmc.StepStatus
		ctStatus       st.StatusCode
		convertSuccess bool
	}{
		"running":  {{evmc.Running, st.Running, true}},
		"stopped":  {{evmc.Stopped, st.Stopped, true}},
		"returned": {{evmc.Returned, st.Returned, true}},
		"reverted": {{evmc.Reverted, st.Reverted, true}},
		"failed":   {{evmc.Failed, st.Failed, true}},
		"error":    {{-1, st.NumStatusCodes, false}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				ctStatus, err := convertEvmcStatusToCtStatus(cur.evmcStatus)

				if cur.convertSuccess && err != nil {
					t.Fatalf("unexpected conversion error: %v", err)
				}

				if !cur.convertSuccess && err == nil {
					t.Fatalf("expected conversion error, but got none")
				}

				if want, got := cur.ctStatus, ctStatus; cur.convertSuccess && want != got {
					t.Errorf("unexpected status: wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Revision(t *testing.T) {
	tests := map[string][]struct {
		evmcRevision   evmc.Revision
		ctRevision     ct.Revision
		convertSuccess bool
	}{
		"istanbul": {{evmc.Istanbul, ct.R07_Istanbul, true}},
		"berlin":   {{evmc.Berlin, ct.R09_Berlin, true}},
		"london":   {{evmc.London, ct.R10_London, true}},
		"next":     {{evmc.MaxRevision, ct.R99_UnknownNextRevision, true}},
		"error":    {{evmc.MaxRevision + 1, ct.R99_UnknownNextRevision, false}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				ctRevision, err := convertEvmcRevisionToCtRevision(cur.evmcRevision)

				if cur.convertSuccess && err != nil {
					t.Fatalf("unexpected conversion error: %v", err)
				}

				if !cur.convertSuccess && err == nil {
					t.Fatalf("expected conversion error, but got none")
				}

				if want, got := cur.ctRevision, ctRevision; cur.convertSuccess && want != got {
					t.Errorf("unexpected revision: wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Pc(t *testing.T) {
	tests := map[string][]struct {
		evmzeroPc uint64
		evmPc     uint16
	}{
		"empty": {{}},
		"pos-0": {{0, 0}},
		"pos-1": {{1, 1}},
		"end":   {{0x6000, 0x6000}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				evmzeroEvaluation := CreateEvaluation(st.NewState(st.NewCode([]byte{})))
				if len(evmzeroEvaluation.issues) > 0 {
					t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
				}

				stepResult := evmc.StepResult{
					StepStatusCode: evmc.Running,
					Revision:       evmc.Istanbul,
					Pc:             cur.evmzeroPc,
				}

				state, err := evmzeroEvaluation.convertEvmzeroStateToCtState(stepResult)

				if err != nil {
					t.Fatalf("failed to convert evmzero to ct state: %v", err)
				}

				if want, got := cur.evmPc, state.Pc; want != got {
					t.Errorf("unexpected program counter, wanted %d, got %d", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Gas(t *testing.T) {
	tests := map[string][]struct {
		evmzeroGas          int64
		evmzeroGasReduction uint64
		ctGas               uint64
	}{
		"zero":           {{0, 0, 0}},
		"non-zero":       {{777, 0, 777}},
		"int64-max":      {{math.MaxInt64, 0, math.MaxInt64}},
		"over-int64-max": {{math.MaxInt64, 777, math.MaxInt64 + 777}},
		"uint64-max":     {{math.MaxInt64, math.MaxInt64 + 1, math.MaxUint64}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				evmzeroEvaluation := CreateEvaluation(st.NewState(st.NewCode([]byte{})))
				evmzeroEvaluation.gasReduction = cur.evmzeroGasReduction
				if len(evmzeroEvaluation.issues) > 0 {
					t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
				}

				stepResult := evmc.StepResult{
					StepStatusCode: evmc.Running,
					Revision:       evmc.Istanbul,
					GasLeft:        cur.evmzeroGas,
				}

				state, err := evmzeroEvaluation.convertEvmzeroStateToCtState(stepResult)

				if err != nil {
					t.Fatalf("failed to convert evmzero to ct state: %v", err)
				}

				if want, got := cur.ctGas, state.Gas; want != got {
					t.Errorf("unexpected gas value, wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_GasRefund(t *testing.T) {
	tests := map[string][]struct {
		evmzeroGasRefund          int64
		evmzeroGasRefundReduction uint64
		ctGasRefund               uint64
	}{
		"zero":           {{0, 0, 0}},
		"non-zero":       {{777, 0, 777}},
		"int64-max":      {{math.MaxInt64, 0, math.MaxInt64}},
		"over-int64-max": {{math.MaxInt64, 777, math.MaxInt64 + 777}},
		"uint64-max":     {{math.MaxInt64, math.MaxInt64 + 1, math.MaxUint64}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				evmzeroEvaluation := CreateEvaluation(st.NewState(st.NewCode([]byte{})))
				evmzeroEvaluation.gasRefundReduction = cur.evmzeroGasRefundReduction
				if len(evmzeroEvaluation.issues) > 0 {
					t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
				}

				stepResult := evmc.StepResult{
					StepStatusCode: evmc.Running,
					Revision:       evmc.Istanbul,
					GasRefund:      cur.evmzeroGasRefund,
				}

				state, err := evmzeroEvaluation.convertEvmzeroStateToCtState(stepResult)

				if err != nil {
					t.Fatalf("failed to convert evmzero to ct state: %v", err)
				}

				if want, got := cur.ctGasRefund, state.GasRefund; want != got {
					t.Errorf("unexpected gas value, wanted %v, got %v", want, got)
				}
			}
		})
	}
}

func TestConvertToCt_Stack(t *testing.T) {
	tests := map[string][]struct {
		evmcStack      []byte
		ctStack        *st.Stack
		convertSuccess bool
	}{
		"empty": {{
			[]byte{},
			st.NewStack(),
			true}},
		"one-element": {{
			[]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xAB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xCD,
				0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
			},
			st.NewStack(ct.NewU256(0xAB000000000000CD, 0xBA000000000000AD)),
			true}},
		"two-elements": {{
			[]byte{
				0xDE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
				0xBE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xEF,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xAB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xCD,
				0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
			},
			st.NewStack(
				ct.NewU256(0xAB000000000000CD, 0xBA000000000000AD),
				ct.NewU256(0xDE000000000000AD, 0xBE000000000000EF, 0, 0)),
			true}},
		"three-elements": {{
			[]byte{
				0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
				0x08, 0x09, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15,
				0x16, 0x17, 0x18, 0x19, 0x20, 0x21, 0x22, 0x23,
				0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x30, 0x31,

				0xDE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
				0xBE, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xEF,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xAB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xCD,
				0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xAD,
			},
			st.NewStack(
				ct.NewU256(0xAB000000000000CD, 0xBA000000000000AD),
				ct.NewU256(0xDE000000000000AD, 0xBE000000000000EF, 0x0000000000000000, 0x0000000000000000),
				ct.NewU256(0x0001020304050607, 0x0809101112131415, 0x1617181920212223, 0x2425262728293031)),
			true}},
		"error": {{
			[]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0xAB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xCD,
				0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			st.NewStack(),
			false}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cur := range test {
				ctStack, err := convertEvmcStackToCtStack(cur.evmcStack)

				if cur.convertSuccess && err != nil {
					t.Fatalf("unexpected conversion error: %v", err)
				}

				if !cur.convertSuccess && err == nil {
					t.Fatalf("expected conversion error, but got none")
				}

				if cur.convertSuccess {
					if want, got := cur.ctStack.Size(), ctStack.Size(); want != got {
						t.Fatalf("unexpected stack size, wanted %v, got %v", want, got)
					}

					for i := 0; i < ctStack.Size(); i++ {
						want := cur.ctStack.Get(i)
						got := ctStack.Get(i)
						if !want.Eq(got) {
							t.Errorf("unexpected stack value, wanted %v, got %v", want, got)
						}
					}
				}
			}
		})
	}
}

func TestConvertToCt_CallContext(t *testing.T) {
	evmzeroEvaluation := CreateEvaluation(st.NewState(st.NewCode([]byte{})))
	if len(evmzeroEvaluation.issues) > 0 {
		t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
	}

	objAddress := vm.AccountRef{0xff}
	callerAddress := vm.AccountRef{0xfe}
	contract := vm.NewContract(callerAddress, objAddress, big.NewInt(252), 0)
	evmzeroEvaluation.contract = contract
	evmzeroEvaluation.evmzero.evm.Origin = common.Address{0xfd}

	stepResult := evmc.StepResult{
		StepStatusCode: evmc.Running,
		Revision:       evmc.Istanbul,
	}

	state, err := evmzeroEvaluation.convertEvmzeroStateToCtState(stepResult)
	if err != nil {
		t.Fatalf("failed to convert evmzero to ct state: %v", err)
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
	evmzeroEvaluation := CreateEvaluation(st.NewState(st.NewCode([]byte{})))
	if len(evmzeroEvaluation.issues) > 0 {
		t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
	}

	evmzeroEvaluation.evmzero.evm.Context.BaseFee = big.NewInt(254)
	evmzeroEvaluation.evmzero.evm.Context.BlockNumber = big.NewInt(255)
	evmzeroEvaluation.evmzero.chainConfig.ChainID = big.NewInt(256)
	evmzeroEvaluation.evmzero.evm.Context.Coinbase = common.Address{0xfe}
	evmzeroEvaluation.evmzero.evm.Context.GasLimit = uint64(253)
	evmzeroEvaluation.evmzero.evm.TxContext.GasPrice = big.NewInt(252)
	evmzeroEvaluation.evmzero.evm.Context.Difficulty = big.NewInt(251)
	evmzeroEvaluation.evmzero.evm.Context.Time = big.NewInt(250)

	stepResult := evmc.StepResult{
		StepStatusCode: evmc.Running,
		Revision:       evmc.Istanbul,
	}

	state, err := evmzeroEvaluation.convertEvmzeroStateToCtState(stepResult)
	if err != nil {
		t.Fatalf("failed to convert evmzero to ct state: %v", err)
	}

	if want, got := ct.NewU256(254), state.BlockContext.BaseFee; !want.Eq(got) {
		t.Errorf("unexpected base fee, wanted %v, got %v", want, got)
	}
	if want, got := uint64(255), state.BlockContext.BlockNumber; want != got {
		t.Errorf("unexpected block number, wanted %v, got %v", want, got)
	}
	if want, got := ct.NewU256(256), state.BlockContext.ChainID; !want.Eq(got) {
		t.Errorf("unexpected chain ID, wanted %v, got %v", want, got)
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
	if want, got := ct.NewU256(251), state.BlockContext.Difficulty; !want.Eq(got) {
		t.Errorf("unexpected difficulty, wanted %v, got %v", want, got)
	}
	if want, got := uint64(250), state.BlockContext.TimeStamp; want != got {
		t.Errorf("unexpected timestamp, wanted %v, got %v", want, got)
	}
}

func TestConvertToCt_CallData(t *testing.T) {
	evmzeroEvaluation := CreateEvaluation(st.NewState(st.NewCode([]byte{})))
	if len(evmzeroEvaluation.issues) > 0 {
		t.Fatalf("failed to convert ct state to evmzero: %v", errors.Join(evmzeroEvaluation.issues...))
	}
	evmzeroEvaluation.input = []byte{1}

	stepResult := evmc.StepResult{
		StepStatusCode: evmc.Running,
		Revision:       evmc.Istanbul,
	}

	state, err := evmzeroEvaluation.convertEvmzeroStateToCtState(stepResult)
	if err != nil {
		t.Fatalf("failed to convert evmzero to ct state: %v", err)
	}

	if want, got := evmzeroEvaluation.input, state.CallData; !slices.Equal(want, got) {
		t.Errorf("unexpected calldata value. wanted %v, got %v", want, got)
	}
}

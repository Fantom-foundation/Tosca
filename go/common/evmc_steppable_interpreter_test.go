package common

import (
	"testing"

	ct "github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/ethereum/evmc/v10/bindings/go/evmc"
)

////////////////////////////////////////////////////////////
// ct -> evmzero

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
				evmcStack := convertCtStackToEvmcStack(cur.ctStack)

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

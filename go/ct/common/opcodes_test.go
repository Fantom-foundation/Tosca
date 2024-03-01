package common

import (
	"regexp"
	"slices"
	"testing"
)

func TestOpCode_ValidOpCodes(t *testing.T) {
	noPrettyPrint := regexp.MustCompile(`^op\(\d+\)$`)
	for i := 0; i < 256; i++ {
		op := OpCode(i)

		want := !noPrettyPrint.MatchString(op.String())
		if op == INVALID {
			want = false
		}
		got := IsValid(op)
		if want != got {
			t.Errorf("invalid classification of instruction %v, wanted %t, got %t", op, want, got)
		}
	}
}

func TestOpCode_ValidOpCodesNoPush(t *testing.T) {
	validOps := ValidOpCodesNoPush()

	noPrettyPrint := regexp.MustCompile(`^op\(\d+\)$`)
	for i := 0; i < 256; i++ {
		op := OpCode(i)

		shouldBePresent := !noPrettyPrint.MatchString(op.String())
		if op == INVALID {
			shouldBePresent = false
		} else if PUSH1 <= op && op <= PUSH32 {
			shouldBePresent = false
		}

		if present := slices.Contains(validOps, op); present && !shouldBePresent {
			t.Errorf("%v should not be in ValidOpCodesNoPush", op)
		} else if !present && shouldBePresent {
			t.Errorf("%v should be in ValidOpCodesNoPush", op)
		}
	}
}

func TestOpCode_CanBePrinted(t *testing.T) {
	validName := regexp.MustCompile(`^op\(\d+\)|([A-Z0-9]+)$`)
	for i := 0; i < 256; i++ {
		op := OpCode(i)
		if !validName.MatchString(op.String()) {
			t.Errorf("Invalid print for op %v (%d)", op, i)
		}
	}
}

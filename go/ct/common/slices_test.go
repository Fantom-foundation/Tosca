package common

import (
	"slices"
	"testing"
)

func TestSlices_RightPadSlice(t *testing.T) {
	base := []int{1, 2, 3}
	tests := map[string]struct {
		length int
		want   []int
	}{
		"longer":  {length: 5, want: []int{1, 2, 3, 0, 0}},
		"shorter": {length: 1, want: base[:1]},
		"equal":   {length: len(base), want: base},
	}

	for name, test := range tests {
		if got := RightPadSlice(base, test.length); !slices.Equal(test.want, got) {
			t.Errorf("Right padding for %v failed, wanted %v, but got %v", name, test.want, got)
		}
	}
}

func TestSlices_LeftPadSlice(t *testing.T) {
	base := []int{1, 2, 3}
	tests := map[string]struct {
		length int
		want   []int
	}{
		"longer":  {length: 5, want: []int{0, 0, 1, 2, 3}},
		"shorter": {length: 1, want: base[:1]},
		"equal":   {length: len(base), want: base},
	}

	for name, test := range tests {
		if got := LeftPadSlice(base, test.length); !slices.Equal(test.want, got) {
			t.Errorf("Right padding for %v failed, wanted %v, but got %v", name, test.want, got)
		}
	}
}

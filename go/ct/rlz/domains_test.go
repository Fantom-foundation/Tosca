// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package rlz

import (
	"math"
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"github.com/Fantom-foundation/Tosca/go/tosca"
	"github.com/Fantom-foundation/Tosca/go/tosca/vm"
)

func TestRemoveDuplicatesGeneric(t *testing.T) {

	tests := map[string]struct {
		generic_type reflect.Type
		input        any
		expected     any
	}{
		"empty": {
			generic_type: reflect.TypeOf(1),
			input:        []int{},
			expected:     []int{},
		},
		"int-with-duplicates": {
			generic_type: reflect.TypeOf(1),
			input:        []int{1, 2, 3, 2, 4, 3, 5, 1},
			expected:     []int{1, 2, 3, 4, 5},
		},
		"int-no-duplicates": {
			generic_type: reflect.TypeOf(1),

			input:    []int{1, 2, 3, 4, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		"string-with-duplicates": {
			generic_type: reflect.TypeOf(""),
			input:        []string{"apple", "banana", "orange", "banana", "kiwi", "orange"},
			expected:     []string{"apple", "banana", "orange", "kiwi"},
		},
		"string-no-duplicates": {
			generic_type: reflect.TypeOf(""),
			input:        []string{"apple", "banana", "orange", "kiwi"},
			expected:     []string{"apple", "banana", "orange", "kiwi"},
		},
		"float-with-duplicates": {
			generic_type: reflect.TypeOf(1.1),
			input:        []float64{1.1, 2.2, 3.3, 2.2, 4.4, 3.3, 5.5, 1.1},
			expected:     []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		},
		"float-no-duplicates": {
			generic_type: reflect.TypeOf(1.1),
			input:        []float64{1.1, 2.2, 3.3, 4.4, 5.5},
			expected:     []float64{1.1, 2.2, 3.3, 4.4, 5.5},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var result any
			switch tc.generic_type {
			case reflect.TypeOf(1):
				result = removeDuplicatesGeneric[int](tc.input.([]int))
			case reflect.TypeOf(""):
				result = removeDuplicatesGeneric[string](tc.input.([]string))
			case reflect.TypeOf(1.1):
				result = removeDuplicatesGeneric[float64](tc.input.([]float64))
			default:
				t.Errorf("Add type to test cases: %v", tc.generic_type)
			}
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %v, but got %v", tc.expected, result)
			}
		})
	}
}

func TestDomain_Equal(t *testing.T) {

	tests := map[string]struct {
		got  bool
		want bool
	}{
		"bool-true":       {got: boolDomain{}.Equal(true, true), want: true},
		"bool-false":      {got: boolDomain{}.Equal(false, false), want: true},
		"bool-true-false": {got: boolDomain{}.Equal(true, false), want: false},
		"U256-equals": {got: u256Domain{}.Equal(common.NewU256(1, 2, 3, 4), common.NewU256(1, 2, 3, 4)),
			want: true},
		"U256-not-equals": {got: u256Domain{}.Equal(common.NewU256(1, 2, 3, 4), common.NewU256(4, 3, 2, 1)),
			want: false},
		"revision-equals": {got: revisionDomain{}.Equal(tosca.R07_Istanbul, tosca.R07_Istanbul),
			want: true},
		"revision-not-equals": {got: revisionDomain{}.Equal(tosca.R07_Istanbul, tosca.R09_Berlin),
			want: false},
		"blocknumberoffset-equals": {got: BlockNumberOffsetDomain{}.Equal(int64(1), int64(1)),
			want: true},
		"blocknumberoffset-not-equals": {got: BlockNumberOffsetDomain{}.Equal(int64(1), int64(2)),
			want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %v, but got %v", tc.want, tc.got)
			}
		})
	}
}

func TestDomain_Less(t *testing.T) {
	tests := map[string]struct {
		got  bool
		want bool
	}{
		"bool-true":       {got: boolDomain{}.Less(true, true), want: false},
		"bool-false":      {got: boolDomain{}.Less(false, false), want: false},
		"bool-true-false": {got: boolDomain{}.Less(true, false), want: false},
		"bool-false-true": {got: boolDomain{}.Less(false, true), want: true},
		"U256-less-1": {got: u256Domain{}.Less(common.NewU256(0, 2, 3, 4), common.NewU256(1, 2, 3, 4)),
			want: true},
		"U256-less-2": {got: u256Domain{}.Less(common.NewU256(1, 1, 3, 4), common.NewU256(1, 2, 3, 4)),
			want: true},
		"U256-less-3": {got: u256Domain{}.Less(common.NewU256(1, 2, 2, 4), common.NewU256(1, 2, 3, 4)),
			want: true},
		"U256-less-4": {got: u256Domain{}.Less(common.NewU256(1, 2, 3, 3), common.NewU256(1, 2, 3, 4)),
			want: true},
		"U256-less-greater": {got: u256Domain{}.Less(common.NewU256(2, 2, 3, 4), common.NewU256(1, 2, 3, 4)),
			want: false},
		"revision-less": {got: revisionDomain{}.Less(tosca.R07_Istanbul, tosca.R09_Berlin),
			want: true},
		"revision-greater": {got: revisionDomain{}.Less(tosca.R09_Berlin, tosca.R07_Istanbul),
			want: false},
		"blocknumberoffset-less": {got: BlockNumberOffsetDomain{}.Less(int64(1), int64(2)),
			want: true},
		"blocknumberoffset-greater": {got: BlockNumberOffsetDomain{}.Less(int64(2), int64(1)),
			want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %v, but got %v", tc.want, tc.got)
			}
		})
	}
}

func TestDomain_PredecessorPanic(t *testing.T) {

	tests := map[string]any{
		"bool":       true,
		"statusCode": st.Running,
		"opCode":     vm.ADD,
	}

	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()
			switch value := value.(type) {
			case bool:
				boolDomain{}.Predecessor(value)
			case st.StatusCode:
				statusCodeDomain{}.Predecessor(value)
			case vm.OpCode:
				opCodeDomain{}.Predecessor(value)
			default:
				t.Errorf("Add type to test cases: %T", value)
			}
		})
	}

}

func TestDomain_Predecessor(t *testing.T) {

	tests := map[string]struct {
		got  any
		want any
	}{
		"uint16": {got: uint16Domain{}.Predecessor(uint16(1)), want: uint16(0)},
		"revision": {got: revisionDomain{}.Predecessor(tosca.R09_Berlin),
			want: tosca.R07_Istanbul},
		"revision-istanbul": {got: revisionDomain{}.Predecessor(tosca.R07_Istanbul),
			want: common.R99_UnknownNextRevision},
		"revision-unknown": {got: revisionDomain{}.Predecessor(common.R99_UnknownNextRevision),
			want: common.NewestSupportedRevision},
		"pc": {got: pcDomain{}.Predecessor(common.NewU256(1, 2, 3, 4)),
			want: common.NewU256(1, 2, 3, 3)},
		"pc-0": {got: pcDomain{}.Predecessor(common.NewU256(1, 2, 3, 0)),
			want: common.NewU256(1, 2, 2, 0xffffffffffffffff)},
		"stackSize":         {got: stackSizeDomain{}.Predecessor(1), want: 0},
		"blocknumberoffset": {got: BlockNumberOffsetDomain{}.Predecessor(int64(1)), want: int64(0)},
		"u256": {got: u256Domain{}.Predecessor(common.NewU256(1, 2, 3, 4)),
			want: common.NewU256(1, 2, 3, 3)},
		"u256-0": {got: u256Domain{}.Predecessor(common.NewU256(1, 2, 0, 4)),
			want: common.NewU256(1, 2, 0, 3)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %v, but got %v", tc.want, tc.got)
			}
		})
	}
}

func TestDomain_SuccessorPanic(t *testing.T) {
	tests := map[string]any{
		"bool":       true,
		"statusCode": st.Running,
		"opCode":     vm.ADD,
	}

	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("The code did not panic")
				}
			}()
			switch value := value.(type) {
			case bool:
				boolDomain{}.Successor(value)
			case st.StatusCode:
				statusCodeDomain{}.Successor(value)
			case vm.OpCode:
				opCodeDomain{}.Successor(value)
			default:
				t.Errorf("Add type to test cases: %T", value)
			}
		})
	}

}

func TestDomain_Successor(t *testing.T) {

	tests := map[string]struct {
		got  any
		want any
	}{
		"uint16": {got: uint16Domain{}.Successor(uint16(1)), want: uint16(2)},
		"revision": {got: revisionDomain{}.Successor(tosca.R09_Berlin),
			want: tosca.R10_London},
		"revision-newest": {got: revisionDomain{}.Successor(common.NewestSupportedRevision),
			want: common.R99_UnknownNextRevision},
		"revision-unknown": {got: revisionDomain{}.Successor(common.R99_UnknownNextRevision),
			want: tosca.R07_Istanbul},
		"pc": {got: pcDomain{}.Successor(common.NewU256(1, 2, 3, 4)),
			want: common.NewU256(1, 2, 3, 5)},
		"pc-0": {got: pcDomain{}.Successor(common.NewU256(1, 2, 2, 0xffffffffffffffff)),
			want: common.NewU256(1, 2, 3, 0)},
		"stackSize":         {got: stackSizeDomain{}.Successor(1), want: 2},
		"blocknumberoffset": {got: BlockNumberOffsetDomain{}.Successor(int64(1)), want: int64(2)},
		"u256": {got: u256Domain{}.Successor(common.NewU256(1, 2, 3, 4)),
			want: common.NewU256(1, 2, 3, 5)},
		"u256-0": {got: u256Domain{}.Successor(common.NewU256(1, 2, 0, 4)),
			want: common.NewU256(1, 2, 0, 5)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %v, but got %v", tc.want, tc.got)
			}
		})
	}
}

func TestDomain_SomethingNotEqual(t *testing.T) {

	tests := map[string]struct {
		got  any
		want any
	}{
		"bool-true":  {got: boolDomain{}.SomethingNotEqual(true), want: false},
		"bool-false": {got: boolDomain{}.SomethingNotEqual(false), want: true},
		"uint16":     {got: uint16Domain{}.SomethingNotEqual(uint16(1)), want: uint16(2)},
		"revision": {got: revisionDomain{}.SomethingNotEqual(tosca.R09_Berlin),
			want: tosca.R10_London},
		"statusCode-running": {got: statusCodeDomain{}.SomethingNotEqual(st.Running),
			want: st.Stopped},
		"statusCode-not-running": {got: statusCodeDomain{}.SomethingNotEqual(st.Stopped),
			want: st.Running},
		"pc": {got: pcDomain{}.SomethingNotEqual(common.NewU256(1, 2, 3, 4)),
			want: common.NewU256(1, 2, 3, 5)},
		"opCode":            {got: opCodeDomain{}.SomethingNotEqual(vm.ADD), want: vm.MUL},
		"stackSize":         {got: stackSizeDomain{}.SomethingNotEqual(1), want: 2},
		"blocknumberoffset": {got: BlockNumberOffsetDomain{}.SomethingNotEqual(int64(1)), want: int64(3)},
		"u256": {got: u256Domain{}.SomethingNotEqual(common.NewU256(1, 2, 3, 4)),
			want: common.NewU256(1, 2, 3, 5)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %v, but got %v", tc.want, tc.got)
			}
		})
	}
}

func TestDomain_SamplesBool(t *testing.T) {

	opcodesAllSamples := make([]vm.OpCode, 0, 256)
	for i := 0; i < 256; i++ {
		opcodesAllSamples = append(opcodesAllSamples, vm.OpCode(i))
	}

	tests := map[string]struct {
		got  any
		want any
	}{
		"bool-samples": {got: boolDomain{}.Samples(true), want: []bool{false, true}},
		"bool-samples-for-all": {got: boolDomain{}.SamplesForAll([]bool{}),
			want: []bool{false, true}},
		"readonly-samples": {got: readOnlyDomain{}.Samples(true), want: []bool{true}},
		"readonly-samples-for-all": {got: readOnlyDomain{}.SamplesForAll([]bool{}),
			want: []bool{}},
		// samples calls samples for all
		"uint16-samples": {got: uint16Domain{}.Samples(uint16(1)),
			want: []uint16{0, 65535, 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048,
				4096, 8192, 16384, 32768}},
		// samples calls samples for all
		"u256-samples": {got: u256Domain{}.Samples(common.NewU256(1, 2, 3, 4)),
			want: []common.U256{common.NewU256(1, 2, 3, 3), common.NewU256(1, 2, 3, 4),
				common.NewU256(1, 2, 3, 5), common.NewU256(), common.NewU256(1), common.NewU256(0x100),
				common.NewU256(0x10000), common.NewU256(0x100000000), common.NewU256(0x1000000000000),
				common.NewU256(1, 0), common.NewU256(1, 0, 0), common.NewU256(1, 0, 0, 0),
				common.NewU256(1).Shl(common.NewU256(255)), common.NewU256(0).Not(), common.NewU256(1, 1)}},
		// samples calls samples for all
		"value-samples": {got: valueDomain{}.Samples(common.NewU256(1, 2, 3, 4)),
			want: []common.U256{common.NewU256(1, 2, 3, 3), common.NewU256(1, 2, 3, 4),
				common.NewU256(1, 2, 3, 5)},
		},
		// samples calls samples for all
		"revision-samples": {got: revisionDomain{}.Samples(tosca.R09_Berlin),
			want: []tosca.Revision{common.R99_UnknownNextRevision, tosca.R07_Istanbul, tosca.R09_Berlin, tosca.R10_London,
				tosca.R11_Paris, tosca.R12_Shanghai, tosca.R13_Cancun}},
		"statusCode-samplesforall": {got: statusCodeDomain{}.SamplesForAll([]st.StatusCode{}),
			want: []st.StatusCode{st.Running, st.Stopped, st.Reverted, st.Failed}},
		"opcpode-samplesforall": {got: opCodeDomain{}.SamplesForAll([]vm.OpCode{}),
			want: opcodesAllSamples},
		// samples calls samples for all
		"blocknumberoffset-samples": {got: BlockNumberOffsetDomain{}.Samples(int64(23)),
			want: []int64{math.MinInt64, -1, 0, 1, 255, 256, 257, math.MaxInt64, 22, 23, 24},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Errorf("want %v, but got %v", tc.want, tc.got)
			}
		})
	}
}

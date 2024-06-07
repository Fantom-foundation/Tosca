package rlz

import (
	"reflect"
	"testing"
)

func TestRemoveDuplicatesGeneric(t *testing.T) {

	tests := map[string]struct {
		generic_type reflect.Type
		input        interface{}
		expected     interface{}
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
			var result interface{}
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

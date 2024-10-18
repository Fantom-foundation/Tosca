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
	"reflect"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestNumericParameter_Samples(t *testing.T) {

	tests := map[string]struct {
		got  []common.U256
		want []common.U256
	}{
		"NumericParameter": {
			got:  NumericParameter{}.Samples(),
			want: numericParameterSamples,
		},
		"jumpTargetParameter": {
			got:  JumpTargetParameter{}.Samples(),
			want: jumpTargetParameterSamples,
		},
		"MemoryOffsetParameter": {
			got:  MemoryOffsetParameter{}.Samples(),
			want: memoryOffsetParameterSamples,
		},
		"SizeParameter": {
			got:  SizeParameter{}.Samples(),
			want: sizeParameterSamples,
		},
		"TopicParameter": {
			got:  TopicParameter{}.Samples(),
			want: topicParameterSamples,
		},
		"AddressParameter": {
			got:  AddressParameter{}.Samples(),
			want: addressParameterSamples,
		},
		"GasParameter": {
			got:  GasParameter{}.Samples(),
			want: gasParameterSamples,
		},
		"ValueParameter": {
			got:  ValueParameter{}.Samples(),
			want: valueParameterSamples,
		},
		"DataOffsetParameter": {
			got:  DataOffsetParameter{}.Samples(),
			want: dataOffsetParameterSamples,
		},
	}

	for name, tc := range tests {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Errorf("%s = %v, want %v", name, tc.got, tc.want)
		}
	}
}

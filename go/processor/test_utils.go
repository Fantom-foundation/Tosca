// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package test_utils

import "github.com/Fantom-foundation/Tosca/go/tosca"

// valid input for point evaluation taken from geth
var ValidPointEvaluationInput = []byte{1, 231, 152, 21, 71, 8, 254, 119, 137, 66, 150, 52, 5, 60, 191, 159,
	153, 182, 25, 249, 240, 132, 4, 137, 39, 51, 63, 206, 99, 127, 84, 155, 86, 76,
	10, 17, 160, 247, 4, 244, 252, 62, 138, 207, 224, 248, 36, 95, 10, 209, 52, 123,
	55, 143, 191, 150, 226, 6, 218, 17, 165, 211, 99, 6, 36, 210, 80, 50, 230, 122,
	126, 106, 73, 16, 223, 88, 52, 184, 254, 112, 230, 188, 254, 234, 192, 53, 36,
	52, 25, 107, 223, 75, 36, 133, 213, 161, 143, 89, 168, 210, 161, 166, 37, 161,
	127, 63, 234, 15, 229, 235, 140, 137, 109, 179, 118, 79, 49, 133, 72, 27, 194,
	47, 145, 180, 170, 255, 204, 162, 95, 38, 147, 104, 87, 188, 58, 124, 37, 57,
	234, 142, 195, 169, 82, 183, 135, 48, 51, 224, 56, 50, 110, 135, 237, 62, 18,
	118, 253, 20, 2, 83, 250, 8, 233, 252, 37, 251, 45, 154, 152, 82, 127, 194, 42,
	44, 150, 18, 251, 234, 253, 173, 68, 108, 188, 123, 205, 189, 205, 120, 10, 242,
	193, 106}

func NewAddress(in byte) tosca.Address {
	val := tosca.NewValue(uint64(in))
	return tosca.Address(val[12:32])
}

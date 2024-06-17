// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package common

func RightPadSlice[T any](source []T, size int) []T {
	res := make([]T, size)
	copy(res, source)
	return res
}

func LeftPadSlice[T any](source []T, size int) []T {
	res := make([]T, size)
	if size < len(source) {
		copy(res, source)
	} else {
		copy(res[size-len(source):], source)
	}
	return res
}

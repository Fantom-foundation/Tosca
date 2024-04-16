//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package lfvm

const (
	errGasUintOverflow       = ConstError("gas uint64 overflow")
	errInvalidCode           = ConstError("invalid code")
	errInvalidJump           = ConstError("invalid jump destination")
	errOutOfGas              = ConstError("out of gas")
	errReturnDataOutOfBounds = ConstError("return data out of bounds")
	errStackOverflow         = ConstError("stack overflow")
	errStackUnderflow        = ConstError("stack underflow")
	errWriteProtection       = ConstError("write protection")
)

// ConstError is an error type that can be used to define immutable
// error constants.
type ConstError string

func (e ConstError) Error() string {
	return string(e)
}

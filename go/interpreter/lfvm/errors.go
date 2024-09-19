// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import "github.com/Fantom-foundation/Tosca/go/tosca"

const (
	errGasUintOverflow       = tosca.ConstError("gas uint64 overflow")
	errInvalidCode           = tosca.ConstError("invalid code")
	errInvalidJump           = tosca.ConstError("invalid jump destination")
	errOutOfGas              = tosca.ConstError("out of gas")
	errReturnDataOutOfBounds = tosca.ConstError("return data out of bounds")
	errStackOverflow         = tosca.ConstError("stack overflow")
	errStackUnderflow        = tosca.ConstError("stack underflow")
	errWriteProtection       = tosca.ConstError("write protection")
	errInitCodeTooLarge      = tosca.ConstError("init code larger than allowed")
)

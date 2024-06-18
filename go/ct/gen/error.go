// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package gen

import "github.com/Fantom-foundation/Tosca/go/ct/common"

// ErrUnsatisfiable is an error returned by generators if constraints
// are not satisfiable.
const ErrUnsatisfiable = common.ConstErr("unsatisfiable constraints")

// ErrUnboundVariable is an error returned by generators if a Variable is used
// in a constraint, but not bound to a value by the given Assignment.
const ErrUnboundVariable = common.ConstErr("unbound variable")

package gen

import "github.com/Fantom-foundation/Tosca/go/ct/common"

// ErrUnsatisfiable is an error returned by generators if constraints
// are not satisfiable.
const ErrUnsatisfiable = common.ConstErr("unsatisfiable constraints")

// ErrUnboundVariable is an error returned by generators if a Variable is used
// in a constraint, but not bound to a value by the given Assignment.
const ErrUnboundVariable = common.ConstErr("unbound variable")

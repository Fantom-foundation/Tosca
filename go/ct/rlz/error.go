package rlz

import "github.com/Fantom-foundation/Tosca/go/ct/common"

var ErrNoEnumeration = common.ConstErr("None of the generated states fulfilled all the conditions")
var ErrSkipped = common.ConstErr("Skipped test")
var ErrInapplicable = common.ConstErr("State does not apply")

var IgnoredErrors []error = []error{ErrSkipped, ErrInapplicable}

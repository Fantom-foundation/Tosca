package rlz

import "errors"

var ErrNoExecution = errors.New("NO STATE FULFILLED ALL CONDITIONS")
var ErrSkipped = errors.New("Skipped test")
var ErrUnapplicable = errors.New("State does not apply")

var IgnoredErrors []error = []error{ErrSkipped, ErrUnapplicable}

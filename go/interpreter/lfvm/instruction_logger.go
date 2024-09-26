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

import (
	"fmt"
	"io"
	"os"
)

// loggingRunner is a runner that logs the execution of the contract code to
// stdout. It is used for debugging purposes.
type loggingRunner struct {
	log io.Writer
}

// NewLoggingRunner creates a new logging runner.
func NewLoggingRunner(writer io.Writer) loggingRunner {
	return loggingRunner{log: writer}
}

func (l loggingRunner) run(c *context) (status, error) {
	if l.log == nil {
		l.log = os.Stderr
	}
	status := statusRunning
	var err error
	for status == statusRunning {
		// log format: <op>, <gas>, <top-of-stack>\n
		if int(c.pc) < len(c.code) {
			top := "-empty-"
			if c.stack.len() > 0 {
				top = c.stack.peek().ToBig().String()
			}
			l.log.Write([]byte(fmt.Sprintf("%v, %d, %v\n", c.code[c.pc].opcode, c.gas, top)))
		}
		status, err = step(c)
		if err != nil {
			return status, err
		}
	}
	return status, nil
}

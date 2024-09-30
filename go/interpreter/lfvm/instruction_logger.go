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
)

// loggingRunner is a runner that logs the execution of the contract code to
// an io.Writer. If no writter is provided with NewLoggingRunner, the log will
// be written to os.Stderr.
type loggingRunner struct {
	log io.Writer
}

// newLogger creates a new logging runner that writes to the provided
// io.Writer.
func newLogger(writer io.Writer) loggingRunner {
	return loggingRunner{log: writer}
}

func (l loggingRunner) run(c *context) (status, error) {
	status := statusRunning
	var err error
	for status == statusRunning {
		// log format: <op>, <gas>, <top-of-stack>\n
		if int(c.pc) < len(c.code) {
			top := "-empty-"
			if c.stack.len() > 0 {
				top = c.stack.peek().ToBig().String()
			}
			if l.log != nil {
				_, err = l.log.Write([]byte(fmt.Sprintf("%v, %d, %v\n", c.code[c.pc].opcode, c.gas, top)))
				if err != nil {
					// TODO: rework this error to be handled differently than step errors
					return status, err
				}
			}
		}
		status, err = step(c)
		if err != nil {
			return status, err
		}
	}
	return status, nil
}

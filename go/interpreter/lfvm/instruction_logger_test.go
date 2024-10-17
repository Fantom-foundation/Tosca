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
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestInterpreter_Logger_ExecutesCodeAndLogs(t *testing.T) {

	tests := map[string]struct {
		code []Instruction
		want string
	}{
		"empty": {},
		"stop": {
			code: []Instruction{{STOP, 0}},
			want: "STOP, 3, -empty-\n",
		},
		"multiple codes": {
			code: []Instruction{{PUSH4, 0}, {DATA, 1}, {STOP, 0}},
			want: "PUSH4, 3, -empty-\nSTOP, 0, 1\n",
		},
		"out of gas": {
			code: []Instruction{
				{PUSH1, 0},
				{PUSH1, 64},
				{MSTORE8, 0},
				{STOP, 0},
			},
			want: "PUSH1, 3, -empty-\nPUSH1, 0, 0\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			// Get tosca.Parameters
			params := tosca.Parameters{Gas: 3}
			code := test.code
			buffer := bytes.NewBuffer([]byte{})
			logger := newLogger(buffer)
			config := config{
				runner: logger,
			}
			_, err := run(config, params, code)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if strings.Compare(buffer.String(), test.want) != 0 {
				t.Errorf("unexpected log: want %v, got %v", test.want, buffer.String())
			}
		})
	}
}

func TestInterpreter_Logger_RunsWithoutOutput(t *testing.T) {

	// Get tosca.Parameters
	params := tosca.Parameters{}
	code := []Instruction{{STOP, 0}}

	// redirect stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut
	defer func() { os.Stdout = oldOut }()

	oldErr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr
	defer func() { os.Stderr = oldErr }()

	logger := newLogger(nil)
	config := config{
		runner: logger,
	}

	_, err := run(config, params, code)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = wOut.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	outOut, err := io.ReadAll(rOut)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = wErr.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	outErr, err := io.ReadAll(rErr)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(outOut) != 0 {
		t.Errorf("unexpected stdout: want \"\", got \"%v\"", outOut)
	}
	if len(outErr) != 0 {
		t.Errorf("unexpected stderr: want \"\", got \"%v\"", outErr)
	}
}

type loggerErrorMock struct{}

func (l loggerErrorMock) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("error")
}

func TestInterpreter_logger_PropagatesWriterError(t *testing.T) {

	logger := newLogger(loggerErrorMock{})
	config := config{
		runner: logger,
	}
	// Get tosca.Parameters
	params := tosca.Parameters{}
	code := []Instruction{{STOP, 0}}

	_, err := run(config, params, code)
	if strings.Compare(err.Error(), "error") != 0 {
		t.Errorf("unexpected error: want error, got %v", err)
	}
}

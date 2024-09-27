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
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Fantom-foundation/Tosca/go/tosca"
)

func TestLogger_ExecutesCodeAndLogs(t *testing.T) {

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
			params := tosca.Parameters{
				Input:  []byte{},
				Static: true,
				Gas:    3,
				Code:   []byte{byte(STOP), 0},
			}
			code := test.code
			buffer := bytes.NewBuffer([]byte{})
			logger := newLogger(buffer)
			config := interpreterConfig{
				runner: logger,
			}
			_, err := run(config, params, code)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if strings.Compare(string(buffer.String()), test.want) != 0 {
				t.Errorf("unexpected log: want %v, got %v", test.want, buffer.String())
			}
		})
	}
}

func TestLogger_IfNoWritterIsProvidedStdErrAndStdOutAreNotUsed(t *testing.T) {

	// Get tosca.Parameters
	params := tosca.Parameters{
		Input:  []byte{},
		Static: true,
		Gas:    10,
		Code:   []byte{byte(STOP), 0},
	}
	code := []Instruction{{STOP, 0}}

	// redirect stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	oldErr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	logger := newLogger(nil)
	config := interpreterConfig{
		runner: logger,
	}
	_, err := run(config, params, code)

	_ = wOut.Close() // ignore error in test
	outOut, _ := io.ReadAll(rOut)
	os.Stdout = oldOut
	_ = wErr.Close() // ignore error in test
	outErr, _ := io.ReadAll(rErr)
	os.Stderr = oldErr

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if strings.Compare(string(outOut), "") != 0 {
		t.Errorf("unexpected stdout: want \"\", got \"%v\"", outOut)
	}
	if strings.Compare(string(outErr), "") != 0 {
		t.Errorf("unexpected stderr: want \"\", got \"%v\"", outErr)
	}
}

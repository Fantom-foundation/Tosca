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
			want: "STOP, 10, -empty-\n",
		},
		"multiple codes": {
			code: []Instruction{{PUSH4, 0}, {DATA, 1}, {STOP, 0}},
			want: "PUSH4, 10, -empty-\nSTOP, 7, 1\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			// Get tosca.Parameters
			params := tosca.Parameters{
				Input:  []byte{},
				Static: true,
				Gas:    10,
				Code:   []byte{byte(STOP), 0},
			}
			code := test.code
			buffer := bytes.NewBuffer([]byte{})
			logger := NewLoggingRunner(buffer)
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

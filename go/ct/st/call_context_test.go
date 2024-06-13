// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package st

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

func TestCallContext_Diff(t *testing.T) {
	tests := map[string]struct {
		change func(*CallContext)
	}{
		"Account Address": {func(c *CallContext) { c.AccountAddress[0]++ }},
		"Caller Address":  {func(c *CallContext) { c.CallerAddress[0]++ }},
		"Value":           {func(c *CallContext) { c.Value = NewU256(1) }},
	}

	callContext := CallContext{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c2 := CallContext{}
			test.change(&c2)
			if diffs := callContext.Diff(&c2); len(diffs) == 0 {
				t.Errorf("No difference found in modified %v", name)
			}
		})
	}
}

func TestCallContext_String(t *testing.T) {
	tests := map[string]struct {
		change func(*CallContext) any
	}{
		"Account Address": {func(c *CallContext) any { c.AccountAddress[19] = 0xff; return c.AccountAddress }},
		"Caller Address":  {func(c *CallContext) any { c.CallerAddress[19] = 0xfd; return c.CallerAddress }},
		"Value":           {func(c *CallContext) any { c.Value = NewU256(1); return c.Value }},
	}

	c := CallContext{}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			v := test.change(&c)
			str := c.String()
			if !strings.Contains(str, fmt.Sprintf("%v: %v", name, v)) {
				t.Errorf("Did not find %v string", name)
			}
		})
	}
}

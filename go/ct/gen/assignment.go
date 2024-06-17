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

import (
	"fmt"
	"sort"
	"strings"

	. "github.com/Fantom-foundation/Tosca/go/ct/common"
)

// Variable is a placeholder for the generation process that will be mapped to a
// specific value during generation. Variables allow us to combine expressions
// without restricting degrees of freedom unnecessarily.
//
// For instance, `Op(Pc()).Restrict(ADD, generator)` restricts the  generator in
// a way that an ADD OpCode will be placed at the position of the current
// program counter without setting the program counter to a specific value.
type Variable string

func (v Variable) String() string {
	return "$" + string(v)
}

// Assignment holds the mapping from Variables to specific values. It is
// populated during the generation process.
type Assignment map[Variable]U256

func (a Assignment) String() string {
	if a == nil {
		return "{}"
	}

	keys := make([]Variable, 0, len(a))
	for key := range a {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	entries := make([]string, 0, len(a))
	for _, key := range keys {
		entries = append(entries, fmt.Sprintf("%s->%v", string(key), a[key]))
	}

	return "{" + strings.Join(entries, ",") + "}"
}

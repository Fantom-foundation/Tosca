//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package main

import (
	"fmt"
	"sort"

	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/urfave/cli/v2"
)

var ListCmd = cli.Command{
	Action: doList,
	Name:   "list",
	Usage:  "List all rules by name",
}

func doList(context *cli.Context) error {
	rules := spc.Spec.GetRules()
	sort.Slice(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	for _, rule := range rules {
		fmt.Println(rule.Name)
	}
	return nil
}

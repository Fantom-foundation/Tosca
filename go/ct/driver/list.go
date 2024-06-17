// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package main

import (
	"fmt"
	"sort"

	cliUtils "github.com/Fantom-foundation/Tosca/go/ct/driver/cli"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/urfave/cli/v2"
)

var ListCmd = cli.Command{
	Action: doList,
	Name:   "list",
	Usage:  "List all rules by name",
	Flags: []cli.Flag{
		cliUtils.FilterFlag,
	},
}

func doList(context *cli.Context) error {

	filter, err := cliUtils.FilterFlag.Fetch(context)
	if err != nil {
		return err
	}

	rules := spc.FilterRules(spc.Spec.GetRules(), filter)
	sort.Slice(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	for _, rule := range rules {
		fmt.Println(rule.Name)
	}
	return nil
}

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
	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
)

var GeneratorInfoCmd = cli.Command{
	Action: doListGeneratorInfo,
	Name:   "generator-info",
	Usage:  "Lists details on the number of test cases produced per rule",
	Flags: []cli.Flag{
		cliUtils.FilterFlag,
	},
}

func doListGeneratorInfo(context *cli.Context) error {

	filter, err := cliUtils.FilterFlag.Fetch(context)
	if err != nil {
		return err
	}
	rules := spc.FilterRules(spc.Spec.GetRules(), filter)

	infos := map[string]rlz.TestCaseEnumerationInfo{}
	for _, rule := range rules {
		infos[rule.Name] = rule.GetTestCaseEnumerationInfo()
	}

	names := maps.Keys(infos)
	sort.Slice(names, func(i, j int) bool {
		infoA := infos[names[i]]
		infoB := infos[names[j]]
		return infoA.TotalNumberOfCases() < infoB.TotalNumberOfCases()
	})

	total := 0
	for _, info := range infos {
		total += info.TotalNumberOfCases()
	}

	for _, name := range names {
		info := infos[name]
		fmt.Printf("----- Rule: %s -----\n%v", name, &info)
		numCases := info.TotalNumberOfCases()
		fmt.Printf("Share of total rules: %.1f%%\n\n", (float32(numCases)/float32(total))*100)
	}
	fmt.Printf("Total number of tests: %d\n", total)
	return nil
}

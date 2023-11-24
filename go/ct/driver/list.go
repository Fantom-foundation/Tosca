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

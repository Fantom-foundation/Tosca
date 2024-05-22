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

package cliUtils

import (
	"regexp"
	"runtime"

	"github.com/urfave/cli/v2"
)

type filterFlagType struct {
	flag cli.StringFlag
}

var FilterFlag = filterFlagType{flag: cli.StringFlag{
	Name:    "filter",
	Aliases: []string{"f"},
	Usage:   "execute only for rules which name matches the given regex",
	Value:   "",
},
}

func (f *filterFlagType) GetFlag() cli.Flag {
	return &f.flag
}

func (f *filterFlagType) Fetch(context *cli.Context) (*regexp.Regexp, error) {
	return regexp.Compile(context.String(f.flag.Name))
}

type jobsFlagType struct {
	flag cli.IntFlag
}

var JobsFlag = jobsFlagType{
	cli.IntFlag{
		Name:    "jobs",
		Aliases: []string{"j"},
		Usage:   "number of jobs run simultaneously",
		Value:   runtime.NumCPU(),
	},
}

func (f *jobsFlagType) GetFlag() cli.Flag {
	return &f.flag
}

func (f *jobsFlagType) Fetch(context *cli.Context) int {
	return context.Int(f.flag.Name)
}

type seedFlagType struct {
	flag cli.Uint64Flag
}

var SeedFlag = seedFlagType{
	cli.Uint64Flag{
		Name:    "seed",
		Aliases: []string{"s"},
		Usage:   "seed for the random number generator",
	},
}

func (f *seedFlagType) GetFlag() cli.Flag {
	return &f.flag
}

func (f *seedFlagType) Fetch(context *cli.Context) uint64 {
	return context.Uint64(f.flag.Name)
}

type cpuProfileType struct {
	flag cli.StringFlag
}

var CpuProfileFlag = cpuProfileType{
	cli.StringFlag{
		Name:      "cpuprofile",
		Usage:     "store CPU profile in the provided filename",
		TakesFile: true,
	},
}

func (f *cpuProfileType) GetFlag() cli.Flag {
	return &f.flag
}

func (f *cpuProfileType) Fetch(context *cli.Context) string {
	return context.String(f.flag.Name)
}

type fullModeFlagType struct {
	flag cli.BoolFlag
}

var FullModeFlag = fullModeFlagType{
	cli.BoolFlag{
		Name:  "full-mode",
		Usage: "if enabled, test cases targeting rules other than the one generating the case will be executed",
	},
}

func (f *fullModeFlagType) GetFlag() cli.Flag {
	return &f.flag
}

func (f *fullModeFlagType) Fetch(context *cli.Context) bool {
	return context.Bool(f.flag.Name)
}

// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package cliUtils

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"

	"github.com/urfave/cli/v2"
)

type filterFlagType struct {
	cli.StringFlag
}

var FilterFlag = &filterFlagType{
	cli.StringFlag{
		Name:    "filter",
		Aliases: []string{"f"},
		Usage:   "execute only for rules which name matches the given regex",
		Value:   "",
	},
}

func (f *filterFlagType) Fetch(context *cli.Context) (*regexp.Regexp, error) {
	return regexp.Compile(context.String(f.Name))
}

type jobsFlagType struct {
	cli.IntFlag
}

var JobsFlag = &jobsFlagType{
	cli.IntFlag{
		Name:    "jobs",
		Aliases: []string{"j"},
		Usage:   "number of jobs run simultaneously",
		Value:   runtime.NumCPU(),
	},
}

func (f *jobsFlagType) Fetch(context *cli.Context) int {
	return context.Int(f.Name)
}

type seedFlagType struct {
	cli.Uint64Flag
}

var SeedFlag = &seedFlagType{
	cli.Uint64Flag{
		Name:    "seed",
		Aliases: []string{"s"},
		Usage:   "seed for the random number generator",
	},
}

func (f *seedFlagType) Fetch(context *cli.Context) uint64 {
	return context.Uint64(f.Name)
}

type cpuProfileType struct {
	cli.StringFlag
}

var CpuProfileFlag = &cpuProfileType{
	cli.StringFlag{
		Name:      "cpuprofile",
		Usage:     "store CPU profile in the provided filename",
		TakesFile: true,
	},
}

func (f *cpuProfileType) Fetch(context *cli.Context) string {
	return context.String(f.Name)
}

type fullModeFlagType struct {
	cli.BoolFlag
}

var FullModeFlag = &fullModeFlagType{
	cli.BoolFlag{
		Name:  "full-mode",
		Usage: "if enabled, test cases targeting rules other than the one generating the case will be executed",
	},
}

func (f *fullModeFlagType) Fetch(context *cli.Context) bool {
	return context.Bool(f.Name)
}

var commonFlags = []cli.Flag{
	cpuProfileFlag,
}

var cpuProfileFlag = &cli.StringFlag{
	Name:  "cpuprofile",
	Usage: "store CPU profile in the provided filename",
}

func AddCommonFlags(command cli.Command) cli.Command {
	command.Flags = append(command.Flags, commonFlags...)

	action := command.Action
	command.Action = func(ctx *cli.Context) (err error) {

		if cpuprofileFilename := ctx.String(cpuProfileFlag.Name); cpuprofileFilename != "" {
			f, err := os.Create(cpuprofileFilename)
			if err != nil {
				return fmt.Errorf("could not create CPU profile: %w", err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				return fmt.Errorf("could not start CPU profile: %w", err)
			}
			defer pprof.StopCPUProfile()
		}

		return action(ctx)
	}
	return command
}

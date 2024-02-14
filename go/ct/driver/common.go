package main

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/urfave/cli/v2"
)

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

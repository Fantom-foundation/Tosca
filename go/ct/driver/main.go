package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "driver",
		Usage:     "Tosca Conformance Tests Driver",
		Copyright: "(c) 2023 Fantom Foundation",
		Flags:     []cli.Flag{},
		Commands: []*cli.Command{
			&RunCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"github.com/PlatONnetwork/PlatON-Integration-Tests/testcases"
	"gopkg.in/urfave/cli.v1"
	"os"
	"sort"
)

var (
	app = cli.NewApp()
)

func init() {

	// Initialize the CLI app
	app.Commands = []cli.Command{
		testcases.StabPrepareCmd,
		testcases.StartCmd,
		testcases.ExecCmd,
		testcases.ListCmd,
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	app.After = func(ctx *cli.Context) error {
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

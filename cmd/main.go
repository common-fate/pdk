package main

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/pdk/cmd/command"
	"github.com/common-fate/pdk/internal/build"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:      "pdk",
		Writer:    os.Stderr,
		Usage:     "https://commonfate.io",
		UsageText: "pdk [options] [command]",
		Version:   build.Version,
		Commands:  []*cli.Command{&command.Init},
	}
	err := app.Run(os.Args)
	if err != nil {
		// if the error is an instance of clierr.PrintCLIErrorer then print the error accordingly
		if cliError, ok := err.(clierr.PrintCLIErrorer); ok {
			cliError.PrintCLIError()
		} else {
			clio.Error(err.Error())
		}
		os.Exit(1)
	}
}

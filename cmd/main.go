package main

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/pdk/cmd/command"
	"github.com/common-fate/pdk/cmd/command/devhandler"
	"github.com/common-fate/pdk/cmd/command/resources"
	"github.com/common-fate/pdk/cmd/command/run"
	"github.com/common-fate/pdk/cmd/command/test"
	"github.com/common-fate/pdk/internal/build"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

func main() {
	_ = godotenv.Load()

	app := &cli.App{
		Name:  "pdk",
		Usage: "Common Fate Provider Development Kit",
		Commands: []*cli.Command{
			&command.UploadCommand,
			&command.Package,
			&command.Invoke,
			&command.Init,
			&devhandler.Command,
			&test.Test,
			&command.SchemaCommand,
			&resources.Command,
			&run.Command,
			&command.Configure,
			&command.Login,
			&command.Logout,
			&command.PublishCommand,
			&command.PublisherCommand,
		},
		Version: build.Version,
	}

	if err := app.Run(os.Args); err != nil {
		if cliError, ok := err.(clierr.PrintCLIErrorer); ok {
			cliError.PrintCLIError()
		} else {
			clio.Error(err)
		}
		os.Exit(1)
	}
}

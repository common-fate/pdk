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
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

func main() {
	_ = godotenv.Load()

	app := &cli.App{
		Name:  "pdk",
		Usage: "deployment tool for access providers",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "verbose", Usage: "Enable verbose logging, effectively sets environment variable CF_LOG=DEBUG"},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Bool("verbose") {
				os.Setenv("CF_LOG", "DEBUG")
			}
			clio.SetLevelFromEnv("CF_LOG")
			return nil
		},
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

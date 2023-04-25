package devhandler

import "github.com/urfave/cli/v2"

var Command = cli.Command{
	Name:  "devhandler",
	Usage: "Manage development deployments of Provider handlers",
	Subcommands: []*cli.Command{
		&deploy,
		&cleanup,
	},
}

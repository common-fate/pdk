package command

import (
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
)

var SchemaCommand = cli.Command{
	Name:  "schema",
	Usage: "Print the schema of the provider in the current folder",
	Action: func(c *cli.Context) error {
		cmd := exec.Command(".venv/bin/provider", "schema")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	},
}

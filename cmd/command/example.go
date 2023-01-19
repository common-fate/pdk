package command

import (
	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

var Example = cli.Command{
	Name:  "example",
	Usage: "prints some example text",
	Action: func(c *cli.Context) error {
		clio.Success("hello world!")
		return nil
	},
}

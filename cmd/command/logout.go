package command

import (
	"github.com/common-fate/clio"
	"github.com/common-fate/pdk/pkg/tokenstore"
	"github.com/urfave/cli/v2"
)

var Logout = cli.Command{
	Name:  "logout",
	Usage: "Log out of Common Fate Provider Registry",
	Action: func(c *cli.Context) error {
		ts := tokenstore.New()
		err := ts.Clear()
		if err != nil {
			return err
		}

		clio.Success("logged out")

		return nil
	},
}

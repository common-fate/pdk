package command

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/pdk/pkg/client"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
	"github.com/urfave/cli/v2"
)

var PublisherCommand = cli.Command{
	Name:        "publisher",
	Usage:       "Manage publishers",
	Subcommands: []*cli.Command{&CreatePublisherCommand},
}

var CreatePublisherCommand = cli.Command{
	Name:  "create",
	Usage: "Create a publisher",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "id"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		registryclient, err := client.NewWithAuthToken(ctx)
		if err != nil {
			return err
		}
		id := c.String("id")
		if id == "" {
			err = survey.AskOne(&survey.Input{Message: "Publisher ID"}, &id)
			if err != nil {
				return err
			}
		}
		_, err = registryclient.UserCreatePublisher(ctx, providerregistrysdk.UserCreatePublisherJSONRequestBody{
			Id: id,
		})
		if err != nil {
			return err
		}
		clio.Successf("Successfully created publisher")

		return nil
	},
}

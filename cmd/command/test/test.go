package test

import (
	"encoding/json"
	"fmt"

	"github.com/common-fate/pdk/cmd/run"
	"github.com/common-fate/provider-registry-sdk-go/pkg/msg"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

var Test = cli.Command{
	Name: "test",
	Subcommands: []*cli.Command{
		&TestRevoke,
		&TestDescribe,
	},
}

var TestRevoke = cli.Command{
	Name: "revoke",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "subject"},
		&cli.StringFlag{Name: "target"},
	},
	Action: func(c *cli.Context) error {
		// expects that the config exists in the dotenv file
		env, _ := godotenv.Read()
		var target msg.Target
		err := json.Unmarshal([]byte(c.String("target")), &target)
		if err != nil {
			return err
		}

		subject := c.String("subject")

		out, err := run.RunEntrypoint(msg.Revoke{Subject: subject, Target: target}, env)
		if err != nil {
			return err
		}
		fmt.Println(string(out.Response))
		return nil
	},
}

var TestDescribe = cli.Command{
	Name: "describe",
	Action: func(c *cli.Context) error {
		// expects that the config exists in the dotenv file
		env, _ := godotenv.Read()

		out, err := run.RunEntrypoint(msg.Describe{}, env)
		if err != nil {
			return err
		}
		fmt.Println(string(out.Response))
		return nil
	},
}

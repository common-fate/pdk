package run

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/common-fate/clio"
	"github.com/common-fate/provider-registry-sdk-go/pkg/handlerclient"
	"github.com/common-fate/provider-registry-sdk-go/pkg/msg"
	"github.com/joho/godotenv"
	"github.com/segmentio/ksuid"
	"github.com/urfave/cli/v2"
)

var Command = cli.Command{
	Name: "run",
	Subcommands: []*cli.Command{
		&grantCommand,
		&revokeCommand,
		&describeCommand,
	},
}

var grantCommand = cli.Command{
	Name: "grant",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "subject", Required: true},
		&cli.StringFlag{Name: "kind", Required: true},
		&cli.StringSliceFlag{Name: "arg", Aliases: []string{"a"}},
		&cli.StringFlag{Name: "request-id", Usage: "supply a particular request ID (if not provided, an auto-generated ID will be used)"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		// expects that the config exists in the dotenv file
		_ = godotenv.Load()

		requestID := c.String("request-id")
		if requestID == "" {
			// use 'pdk_' prefix to denote generated requests which came from the PDK CLI.
			requestID = "pdk_" + ksuid.New().String()
			clio.Infof("generated a unique Access Request ID: %s", requestID)
		}

		target := msg.Target{
			Kind:      c.String("kind"),
			Arguments: map[string]string{},
		}

		var argFlags []string

		for _, arg := range c.StringSlice("arg") {
			argFlags = append(argFlags, "-a "+arg)

			parts := strings.SplitN(arg, "=", 2) // args are in key=value format
			key := parts[0]
			val := parts[1]
			target.Arguments[key] = val
		}

		subject := c.String("subject")

		rt := handlerclient.Client{
			Executor: handlerclient.Local{},
		}

		request := msg.Grant{
			Subject: subject,
			Target:  target,
			Request: msg.AccessRequest{
				ID: requestID,
			},
		}

		clio.Infow("granting access", "request", request)

		res, err := rt.Grant(ctx, request)
		if err != nil {
			return err
		}

		resJSON, err := json.Marshal(res)
		if err != nil {
			return err
		}

		clio.Successf("granted access: %s", string(resJSON))

		var stateFlags []string
		for key, val := range res.State {
			flag := fmt.Sprintf("-s %s=%s", key, val)
			stateFlags = append(stateFlags, flag)
		}

		revoke := fmt.Sprintf("pdk run revoke --request-id %s --subject %s --kind %s %s %s", requestID, subject, target.Kind, strings.Join(argFlags, " "), strings.Join(stateFlags, " "))

		clio.Infof("revoke access by running:\n%s", revoke)

		return nil
	},
}

var revokeCommand = cli.Command{
	Name: "revoke",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "subject", Required: true},
		&cli.StringFlag{Name: "kind", Required: true},
		&cli.StringSliceFlag{Name: "arg", Aliases: []string{"a"}},
		&cli.StringSliceFlag{Name: "state", Aliases: []string{"s"}},
		&cli.StringFlag{Name: "request-id", Usage: "supply a particular request ID (if not provided, an auto-generated ID will be used)"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		// expects that the config exists in the dotenv file
		_ = godotenv.Load()

		requestID := c.String("request-id")
		if requestID == "" {
			// use 'pdk_' prefix to denote generated requests which came from the PDK CLI.
			requestID = "pdk_" + ksuid.New().String()
			clio.Infof("generated a unique Access Request ID: %s", requestID)
		}

		request := msg.Revoke{
			Subject: c.String("subject"),
			Target: msg.Target{
				Kind:      c.String("kind"),
				Arguments: map[string]string{},
			},
			State: map[string]any{},
			Request: msg.AccessRequest{
				ID: requestID,
			},
		}

		for _, arg := range c.StringSlice("arg") {
			parts := strings.SplitN(arg, "=", 2) // args are in key=value format
			key := parts[0]
			val := parts[1]
			request.Target.Arguments[key] = val
		}

		for _, arg := range c.StringSlice("state") {
			parts := strings.SplitN(arg, "=", 2) // args are in key=value format
			key := parts[0]
			val := parts[1]
			request.State[key] = val
		}

		rt := handlerclient.Client{
			Executor: handlerclient.Local{},
		}

		clio.Infow("revoking access", "request", request)

		err := rt.Revoke(ctx, request)
		if err != nil {
			return err
		}

		clio.Successf("revoked access")
		return nil
	},
}

var describeCommand = cli.Command{
	Name: "describe",
	Action: func(c *cli.Context) error {
		// expects that the config exists in the dotenv file
		_ = godotenv.Load()

		rt := handlerclient.Client{
			Executor: handlerclient.Local{},
		}
		out, err := rt.Describe(c.Context)
		if err != nil {
			return err
		}

		outBytes, err := json.Marshal(out)
		if err != nil {
			return err
		}

		fmt.Println(string(outBytes))
		return nil
	},
}

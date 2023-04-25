package command

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

type Payload struct {
	Type string `json:"type"`
	Data Data   `json:"data"`
}

type Data struct {
	Subject string `json:"subject"`
	Args    any    `json:"args"`
}

var Invoke = cli.Command{
	Name: "invoke",
	Subcommands: []*cli.Command{
		&invokeGrant,
		&invokeRevoke,
		&invokeSchema,
	},
}

var invokeGrant = cli.Command{
	Name: "grant",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "subject", Required: true},
		&cli.StringFlag{Name: "args", Required: true},
		&cli.StringFlag{Name: "handler-id", Required: true},
		&cli.StringFlag{Name: "invoke-role-arn"},
	},
	Action: func(c *cli.Context) error {

		argsStr := c.String("args")
		var args any

		err := json.Unmarshal([]byte(argsStr), &args)
		if err != nil {
			return err
		}
		fmt.Println(args)
		payload := Payload{
			Type: "grant",
			Data: Data{
				Subject: c.String("subject"),
				Args:    args,
			},
		}

		handlerID := c.String("handler-id")
		roleARN := c.String("invoke-role-arn")

		return invokeLambda(c.Context, invokeLambdaOpts{
			Payload:      payload,
			FunctionName: handlerID,
			RoleARN:      roleARN,
		})
	},
}

var invokeRevoke = cli.Command{
	Name: "revoke",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "subject", Required: true},
		&cli.StringFlag{Name: "args", Required: true},
		&cli.StringFlag{Name: "handler-id", Required: true},
		&cli.StringFlag{Name: "invoke-role-arn"},
	},
	Action: func(c *cli.Context) error {

		argsStr := c.String("args")
		var args any

		err := json.Unmarshal([]byte(argsStr), &args)
		if err != nil {
			return err
		}

		handlerID := c.String("handler-id")
		roleARN := c.String("invoke-role-arn")

		payload := Payload{
			Type: "revoke",
			Data: Data{
				Subject: c.String("subject"),
				Args:    args,
			},
		}

		return invokeLambda(c.Context, invokeLambdaOpts{
			Payload:      payload,
			FunctionName: handlerID,
			RoleARN:      roleARN,
		})
	},
}
var invokeSchema = cli.Command{
	Name: "schema",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "handler-id", Required: true},
		&cli.StringFlag{Name: "invoke-role-arn"},
	},
	Action: func(c *cli.Context) error {

		payload := Payload{
			Type: "schema",
		}

		handlerID := c.String("handler-id")
		roleARN := c.String("invoke-role-arn")

		return invokeLambda(c.Context, invokeLambdaOpts{
			Payload:      payload,
			FunctionName: handlerID,
			RoleARN:      roleARN,
		})
	},
}

type invokeLambdaOpts struct {
	Payload      Payload
	FunctionName string
	RoleARN      string
}

func invokeLambda(ctx context.Context, opts invokeLambdaOpts) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}

	payloadbytes, err := json.Marshal(opts.Payload)
	if err != nil {
		return err
	}

	if opts.RoleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		provider := stscreds.NewAssumeRoleProvider(stsClient, opts.RoleARN)
		cfg.Credentials = aws.NewCredentialsCache(provider)
	}

	lambdaclient := lambda.NewFromConfig(cfg)
	out, err := lambdaclient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &opts.FunctionName,
		Payload:      payloadbytes,
		LogType:      types.LogTypeTail,
	})
	if err != nil {
		return err
	}

	clio.Infof(string(out.Payload))

	logs, err := base64.StdEncoding.DecodeString(*out.LogResult)
	if err != nil {
		return err
	}

	clio.Infof(string(logs))
	return nil
}

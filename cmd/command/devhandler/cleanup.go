package devhandler

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

var cleanup = cli.Command{
	Name:  "cleanup",
	Usage: "destroy a development Provider handler deployment",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "id", Required: true, Usage: "the handler ID"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		handlerID := "cf-handler-" + c.String("id")

		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return err
		}

		lambdaclient := lambda.NewFromConfig(cfg)
		_, err = lambdaclient.DeleteFunction(ctx, &lambda.DeleteFunctionInput{
			FunctionName: &handlerID,
		})
		var rnf *types.ResourceNotFoundException
		if err != nil && !errors.As(err, &rnf) {
			clio.Errorw("delete lambda error", "error", err.Error())
		} else {
			clio.Infof("deleted lambda %s", handlerID)
		}

		iamclient := iam.NewFromConfig(cfg)
		_, err = iamclient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
			PolicyArn: aws.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
			RoleName:  &handlerID,
		})
		if err != nil {
			clio.Errorw("detach role policy error", "error", err.Error())
		}

		_, err = iamclient.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: &handlerID,
		})
		if err != nil {
			clio.Errorw("delete role error", "error", err.Error())
		} else {
			clio.Infof("deleted role %s", handlerID)
		}

		return nil
	},
}

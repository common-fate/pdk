package devhandler

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/common-fate/clio"
	"github.com/common-fate/cloudform/deployer"
	"github.com/common-fate/pdk/pkg/pythonconfig"
	"github.com/common-fate/provider-registry-sdk-go/pkg/bootstrapper"
	"github.com/common-fate/provider-registry-sdk-go/pkg/configure"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

var deploy = cli.Command{
	Name:  "deploy",
	Usage: "create a development Provider handler deployment",
	Flags: []cli.Flag{
		&cli.PathFlag{Name: "path", Value: ".", Usage: "the path to the folder containing your provider code e.g ./cf-provider-example"},
		&cli.StringFlag{Name: "id", Required: true, Usage: "the handler ID"},
		&cli.BoolFlag{Name: "confirm", Aliases: []string{"y"}, Usage: "Confirm creation of resources"},
	},
	Action: func(c *cli.Context) error {
		providerPath := c.Path("path")

		_ = godotenv.Load(".env", filepath.Join(providerPath, ".env"))

		ctx := c.Context

		confirm := c.Bool("confirm")

		configFile := filepath.Join(providerPath, "provider.toml")
		pconfig, err := pythonconfig.LoadFile(configFile)
		if err != nil {
			return err
		}

		handlerID := c.String("id")

		dist := filepath.Join(providerPath, "dist")

		fpath := filepath.Join(dist, "handler.zip")

		handlerFile, err := os.Open(fpath)
		if err != nil {
			return err
		}
		defer handlerFile.Close()

		templateFilePath := filepath.Join(dist, "cloudformation.json")
		template, err := os.ReadFile(templateFilePath)
		if err != nil {
			return err
		}

		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return err
		}

		bs := bootstrapper.NewFromConfig(cfg)
		bootstrap, err := bs.GetOrDeployBootstrapBucket(ctx, deployer.WithConfirm(confirm))
		if err != nil {
			return err
		}

		// get the schema of the provider
		var out bytes.Buffer
		cmd := exec.Command(".venv/bin/commonfate-provider-py", "schema")
		cmd.Stderr = os.Stderr
		cmd.Dir = providerPath
		cmd.Stdout = &out
		err = cmd.Run()
		if err != nil {
			return err
		}

		d := deployer.NewFromConfig(cfg)

		s3client := s3.NewFromConfig(cfg)

		lambdaAssetPath := path.Join("dev", "providers", pconfig.Publisher, pconfig.Name, pconfig.Version)
		clio.Infof("Uploading %s to %s", fpath, path.Join(bootstrap.AssetsBucket, lambdaAssetPath, "handler.zip"))

		_, err = s3client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &bootstrap.AssetsBucket,
			Key:    aws.String(path.Join(lambdaAssetPath, "handler.zip")),
			Body:   handlerFile,
		})
		if err != nil {
			return err
		}

		var schema providerregistrysdk.Schema
		err = json.Unmarshal(out.Bytes(), &schema)
		if err != nil {
			return err
		}

		configVals := configure.ConfigFromSchema(schema.Config)
		err = configVals.Fill(ctx, configure.Dev())
		if err != nil {
			return err
		}

		stsClient := sts.NewFromConfig(cfg)
		ci, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return err
		}

		parameters := configVals.CfnParams()

		parameters = append(parameters, types.Parameter{
			ParameterKey:   aws.String("CommonFateAWSAccountID"),
			ParameterValue: ci.Account,
		})

		parameters = append(parameters, types.Parameter{
			ParameterKey:   aws.String("AssetPath"),
			ParameterValue: aws.String(path.Join(lambdaAssetPath, "handler.zip")),
		})

		parameters = append(parameters, types.Parameter{
			ParameterKey:   aws.String("BootstrapBucketName"),
			ParameterValue: aws.String(bootstrap.AssetsBucket),
		})

		parameters = append(parameters, types.Parameter{
			ParameterKey:   aws.String("HandlerID"),
			ParameterValue: aws.String(handlerID),
		})

		paramsJSON, err := json.Marshal(parameters)
		if err != nil {
			return err
		}

		clio.Infow("deploying CloudFormation stack", "name", handlerID, "parameters", string(paramsJSON))

		_, err = d.Deploy(ctx, deployer.DeployOpts{
			Template:  string(template),
			StackName: handlerID,
			Confirm:   confirm,
			Params:    parameters,
		})
		if err != nil {
			return err
		}

		return nil
	},
}

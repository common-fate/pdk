package cfngen

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfn "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/lambda"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/common-fate/pdk/pkg/cfngen/ref"
	"github.com/common-fate/pdk/pkg/iamp"
	"github.com/common-fate/pdk/pkg/pythonconfig"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
)

type Key struct {
	Key              string
	Data             Variable
	EnvVarConfigName string
}

type Variable struct {
	Type   string `json:"type"`
	Usage  string `json:"usage"`
	Secret bool   `json:"secret"`
}

type Opts struct {
	Config map[string]Variable `json:"config"`
}

func ConvertToPascalCase(s string) string {
	arg := strings.Split(s, "_")
	var formattedStr []string

	for _, v := range arg {
		formattedStr = append(formattedStr, strings.ToUpper(v[0:1])+v[1:])
	}

	return strings.Join(formattedStr, "")
}

var reservedParameters = map[string]bool{
	"AssetPath":              true,
	"BootstrapBucketName":    true,
	"CommonFateAWSAccountID": true,
	"HandlerID":              true,
}

func Generate(pconfig pythonconfig.Config, schema providerregistrysdk.Schema) ([]byte, error) {
	template := cfn.NewTemplate()

	template.Metadata["CommonFate::HandlerTemplate::Version"] = "v1"

	template.Parameters[ref.AssetPath] = cfn.Parameter{
		Type:        "String",
		MinLength:   cfn.Int(1),
		Description: cfn.String("The path of the asset in the bootstrap bucket"),
	}

	template.Parameters[ref.BootstrapBucketName] = cfn.Parameter{
		Type:        "String",
		MinLength:   cfn.Int(1),
		Description: cfn.String("The name of the bucket used to bootstrap assets from Common Fate Releases into this account"),
	}

	template.Parameters["CommonFateAWSAccountID"] = cfn.Parameter{
		Type:        "String",
		MinLength:   cfn.Int(1),
		Description: cfn.String("The AWS account Id for the account where Common Fate is deployed"),
	}

	template.Parameters["HandlerID"] = cfn.Parameter{
		Type:        "String",
		MinLength:   aws.Int(1),
		Description: cfn.String("The name of invoke handler lambda function"),
	}

	lambdaFunction := &lambda.Function{
		Runtime:      cfn.String("python3.9"),
		FunctionName: cfn.RefPtr("HandlerID"),
		Timeout:      cfn.Int(600),
		Role:         cfn.GetAtt(ref.LambdaRole, "Arn"),
		Handler:      cfn.String("provider.runtime.aws_lambda_entrypoint.lambda_handler"),
		Tags: []tags.Tag{
			{Key: "common-fate-abac-role", Value: "access-provider"},
		},
		Environment: &lambda.Function_Environment{
			Variables: map[string]string{},
		},
		Code: &lambda.Function_Code{
			S3Bucket: cfn.RefPtr(ref.BootstrapBucketName),
			S3Key:    cfn.RefPtr(ref.AssetPath),
		},
		AWSCloudFormationDependsOn: []string{
			ref.LambdaRole,
		},
	}

	lambdaArn := cfn.GetAtt(ref.LambdaFunction, "Arn")

	arpd := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Action": "sts:AssumeRole",
				"Effect": "Allow",
				"Principal": map[string]any{
					"Service": "lambda.amazonaws.com",
				},
			},
		},
	}

	// scope down the SSM policy, so that the provider is only allowed to read from
	// paths which include the publisher and name of the provider.

	ssmPath := fmt.Sprintf("/common-fate/provider/%s/%s/*", pconfig.Publisher, pconfig.Name)

	hasSecrets := false
	if schema.Config != nil {
		for k, v := range *schema.Config {
			// prevent users from submitting providers that overwrite our built-in
			// parameter names as it could cause unexpected behaviour.
			if reserved, ok := reservedParameters[k]; ok && reserved {
				return nil, fmt.Errorf("%s is a reserved parameter name", k)
			}

			cfnKey := ConvertToPascalCase(k)
			envPrefix := "PROVIDER_CONFIG_"

			if v.Secret != nil && *v.Secret {
				cfnKey += "Secret"
				envPrefix = "PROVIDER_SECRET_"
				hasSecrets = true
			}

			template.Parameters[cfnKey] = cfn.Parameter{
				Type:        "String",
				Description: v.Description,
				MinLength:   cfn.Int(1),
			}

			envVar := envPrefix + strings.ToUpper(k)

			lambdaFunction.Environment.Variables[envVar] = cfn.Ref(cfnKey)
		}
	}

	policyStatements := []map[string]any{
		{
			"Action":    "sts:AssumeRole",
			"Effect":    "Allow",
			"Resource":  "*",
			"Condition": map[string]any{"StringEquals": map[string]string{"iam:ResourceTag/common-fate-abac-role": "access-provider-permissions-role"}},
		}}

	// only give SSM permissions if the Provider actually needs to read secrets.
	if hasSecrets {
		policyStatements = append(policyStatements, map[string]any{
			"Action":   "ssm:GetParameter",
			"Effect":   "Allow",
			"Resource": cfn.Join("", []string{"arn:", ref.AWSPartitionRef, ":ssm:", ref.AWSRegionRef, ":", ref.AWSAccountIDRef, ":parameter", ssmPath}),
		})
	}

	inlinePolicy := map[string]any{
		"Version":   "2012-10-17",
		"Statement": policyStatements,
	}

	template.Resources[ref.LambdaRole] = &iam.Role{
		AssumeRolePolicyDocument: arpd,
		ManagedPolicyArns:        []string{cfn.Join("", []string{"arn:", ref.AWSPartitionRef, ":iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"})},
		Policies: []iam.Role_Policy{
			{
				PolicyName:     "handler-policy",
				PolicyDocument: inlinePolicy,
			},
		},
		RoleName: cfn.RefPtr("HandlerID"),
	}

	template.Resources[ref.LambdaFunction] = lambdaFunction

	invokeRoleARPD := iamp.NewPolicy(
		iamp.Statement{
			Effect: iamp.Allow,
			Action: iamp.Value{"sts:AssumeRole"},
			Principal: map[string]iamp.Value{
				"AWS": {cfn.Join("", []string{"arn:", ref.AWSPartitionRef, ":iam::", cfn.Ref(ref.CommonFateAWSAccountID), ":root"})},
			},
			// TODO: add external ID condition
		},
	)

	invokePolicy := iamp.NewPolicy(
		iamp.Statement{
			Effect: iamp.Allow,
			Sid:    "AllowInvokingFunction",
			Action: iamp.Value{
				"lambda:InvokeFunction",
			},
			Resource: iamp.Value{lambdaArn},
		},
		iamp.Statement{
			Effect: iamp.Allow,
			Sid:    "AllowIntrospectingFunction",
			Action: iamp.Value{
				"lambda:GetFunction",
				"lambda:GetFunctionConfiguration",
			},
			Resource: iamp.Value{lambdaArn},
		},
		iamp.Statement{
			Effect: iamp.Allow,
			Sid:    "AllowReadingFunctionLogs",
			Action: iamp.Value{
				"logs:DescribeLogStreams",
				"logs:GetLogEvents",
			},
			Resource: iamp.Value{
				// log group ARN looks like
				// arn:aws:logs:region:account-id:log-group:log_group_name
				cfn.Join("", []string{"arn:", ref.AWSPartitionRef, ":logs:", ref.AWSRegionRef, ":", ref.AWSAccountIDRef, ":log-group:/aws/lambda/", cfn.Ref(ref.HandlerID), "*"}),
			},
		},
	)

	invokeRoleDescription := cfn.Join("", []string{"Allows Common Fate to invoke the Lambda Function for the ", cfn.Ref(ref.HandlerID), " Handler"})

	invokeRoleName := cfn.Join("", []string{cfn.Ref(ref.HandlerID), "-invoke"})
	// add the invocation role - this is the role that Common Fate assumes in order to invoke the Lambda function
	template.Resources[ref.LambdaInvocationRole] = &iam.Role{
		AssumeRolePolicyDocument: invokeRoleARPD,
		RoleName:                 &invokeRoleName,
		Description:              &invokeRoleDescription,
		Policies: []iam.Role_Policy{
			{
				PolicyName:     "invoke-policy",
				PolicyDocument: invokePolicy,
			},
		},
		Tags: []tags.Tag{{Key: "common-fate-abac-role", Value: "handler-invoke"}},
	}

	return template.JSON()
}

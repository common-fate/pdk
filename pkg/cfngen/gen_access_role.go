package cfngen

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfn "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/common-fate/pdk/pkg/cfngen/ref"
	"github.com/common-fate/pdk/pkg/iamp"
	"github.com/common-fate/pdk/pkg/pythonconfig"
)

func GenerateAccessRole(pconfig pythonconfig.Config, roleName string, policy iamp.Policy) ([]byte, error) {
	template := cfn.NewTemplate()

	template.Metadata["CommonFate::AccessRoleTemplate::Version"] = "v1"
	template.Metadata["CommonFate::Provider::Publisher"] = pconfig.Publisher
	template.Metadata["CommonFate::Provider::Name"] = pconfig.Name

	handlerAccountIDDesc := fmt.Sprintf("The ID of the AWS account that the %s/%s Provider will be deployed to", pconfig.Publisher, pconfig.Name)
	template.Parameters[ref.HandlerAccountID] = cfn.Parameter{
		Type:        "String",
		MinLength:   aws.Int(1),
		Description: &handlerAccountIDDesc,
	}

	template.Parameters[ref.HandlerID] = cfn.Parameter{
		Type:        "String",
		MinLength:   aws.Int(1),
		Description: aws.String("The name of the Lambda function deployed for the provider"),
		Default:     fmt.Sprintf("cf-handler-%s-%s", pconfig.Publisher, pconfig.Name),
	}

	arpd := iamp.NewPolicy(
		iamp.Statement{
			Effect: iamp.Allow,
			Action: iamp.Value{"sts:AssumeRole"},
			Principal: map[string]iamp.Value{
				// only allow the handler function to assume the role
				"AWS": {cfn.Join("", []string{"arn:", ref.AWSPartitionRef, ":iam::", cfn.Ref(ref.HandlerAccountID), ":role/", cfn.Ref(ref.HandlerID)})},
			},
		},
	)

	roleDesc := fmt.Sprintf("Common Fate %s/%s Access Role - %s", pconfig.Publisher, pconfig.Name, roleName)

	cfnRoleName := cfn.Join("", []string{cfn.Ref(ref.HandlerID), "-access-" + roleName})

	template.Resources["Role"] = &iam.Role{
		AssumeRolePolicyDocument: arpd,
		RoleName:                 &cfnRoleName,
		Description:              &roleDesc,
		Policies: []iam.Role_Policy{
			{
				PolicyName:     "access-policy",
				PolicyDocument: policy,
			},
		},
		Tags: []tags.Tag{{Key: "common-fate-abac-role", Value: "access-provider-permissions-role"}},
	}

	template.Outputs["Role"] = cfn.Output{
		Value: cfn.GetAtt("Role", "Arn"),
	}

	return template.JSON()
}

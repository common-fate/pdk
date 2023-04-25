// Package ref exports constants
// used in CloudFormation template generation.
package ref

import "github.com/awslabs/goformation/v7/cloudformation"

// CloudFormation parameters
const (
	BootstrapBucketName    = "BootstrapBucketName"
	AssetPath              = "AssetPath"
	HandlerID              = "HandlerID"
	CommonFateAWSAccountID = "CommonFateAWSAccountID"
	HandlerAccountID       = "HandlerAccountID"
)

// CloudFormation Logical IDs
const (
	LambdaFunction       = "LambdaFunction"
	LambdaRole           = "LambdaRole"
	LambdaInvocationRole = "LambdaInvocationRole"
)

var (
	// { "Ref": "AWS::Partition" }
	AWSPartitionRef = cloudformation.Ref("AWS::Partition")

	// { "Ref": "AWS::Region" }
	AWSRegionRef = cloudformation.Ref("AWS::Region")

	// { "Ref": "AWS::AccountId" }
	AWSAccountIDRef = cloudformation.Ref("AWS::AccountId")
)

package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type AwsStackProps struct {
	awscdk.StackProps
}

func NewAwsStack(scope constructs.Construct, id string, props *AwsStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Upload config as asset in S3
	config := awss3assets.NewAsset(stack, jsii.String("SplitYnabConfig"),
		&awss3assets.AssetProps{
			Path: jsii.String("config.yml"),
		})

	// DynamoDB table as key/value store
	table := awsdynamodb.NewTableV2(stack, jsii.String("SplitYnab"),
		&awsdynamodb.TablePropsV2{
			PartitionKey: &awsdynamodb.Attribute{
				Name: jsii.String("key"),
				Type: awsdynamodb.AttributeType_STRING,
			},
		})

	// Lambda for compute
	lambdaEnv := map[string]*string{
		"CONFIG_BUCKET": config.S3BucketName(),
		"CONFIG_KEY":    config.S3ObjectKey(),
		"TABLE_NAME":    table.TableName(),
	}
	lambda := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("SplitYnabLambda"),
		&awscdklambdagoalpha.GoFunctionProps{
			Entry:       jsii.String("cmd/split-ynab-lambda/"),
			ModuleDir:   jsii.String("."),
			Environment: &lambdaEnv,
		})
	// Grant lambda IAM permissions needed at runtime
	config.GrantRead(lambda)
	table.GrantReadWriteData(lambda)

	// Trigger lambda on a timer
	eventRule := awsevents.NewRule(stack, jsii.String("SplitYnabRule"),
		&awsevents.RuleProps{
			Schedule: awsevents.Schedule_Rate(awscdk.Duration_Hours(jsii.Number(1))),
		})
	eventRule.AddTarget(awseventstargets.NewLambdaFunction(lambda, nil))

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewAwsStack(app, "SplitYnabStack", &AwsStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
	//  Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	// }
}

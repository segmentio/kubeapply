package util

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

// GetSSMValue gets a secret value from AWS SSM.
func GetSSMValue(
	ctx context.Context,
	sess *session.Session,
	ssmKeyName string,
) (string, error) {
	ssmClient := ssm.New(sess)
	result, err := ssmClient.GetParameterWithContext(
		ctx,
		&ssm.GetParameterInput{
			Name:           aws.String(ssmKeyName),
			WithDecryption: aws.Bool(true),
		},
	)
	if err != nil {
		return "", err
	}
	return aws.StringValue(result.Parameter.Value), nil
}

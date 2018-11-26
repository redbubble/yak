package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/redbubble/yak/saml"
)

func AssumeRole(login saml.LoginData, role saml.LoginRole, duration int64) (*sts.AssumeRoleWithSAMLOutput, error) {
	session := session.Must(session.NewSession())
	stsClient := sts.New(session)

	input := sts.AssumeRoleWithSAMLInput{
		DurationSeconds: &duration,
		PrincipalArn:    &role.PrincipalArn,
		RoleArn:         &role.RoleArn,
		SAMLAssertion:   &login.Assertion,
	}

	return stsClient.AssumeRoleWithSAML(&input)
}

func EnvironmentVariables(stsOutput *sts.AssumeRoleWithSAMLOutput) map[string]string {
	subject := make(map[string]string)

	subject["AWS_ACCESS_KEY_ID"] = *stsOutput.Credentials.AccessKeyId
	subject["AWS_SECRET_ACCESS_KEY"] = *stsOutput.Credentials.SecretAccessKey
	subject["AWS_SESSION_TOKEN"] = *stsOutput.Credentials.SessionToken
	subject["AWS_METADATA_USER_ARN"] = *stsOutput.AssumedRoleUser.Arn

	return subject
}

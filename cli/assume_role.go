package cli

import (
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/saml"
)

func AssumeRole(login saml.LoginData, desiredRole string) (*sts.AssumeRoleWithSAMLOutput, error) {
	role, err := login.GetLoginRole(desiredRole)

	if err != nil {
		return nil, err
	}

	return aws.AssumeRole(login, role, viper.GetInt64("aws.session_duration"))
}

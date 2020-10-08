package cli

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cache"
	"github.com/redbubble/yak/saml"
)

var notARoleErrorMessage = `'%s' is neither an IAM role ARN nor a configured alias.

Run 'yak --list-roles' to see which roles and aliases you can use.`

func AssumeRoleFromCache(role string) *sts.AssumeRoleWithSAMLOutput {
	if viper.GetBool("cache.no_cache") {
		return nil
	}

	data, ok := cache.Check(role).(sts.AssumeRoleWithSAMLOutput)

	if !ok {
		return nil
	}

	return &data
}

func ResolveRole(roleName string) (string, error) {
	if viper.IsSet("alias." + roleName) {
		return viper.GetString("alias." + roleName), nil
	}

	if isIamRoleArn(roleName) {
		return roleName, nil
	}

	return "", fmt.Errorf(notARoleErrorMessage, roleName)
}

func AssumeRole(login saml.LoginData, desiredRole string) (*sts.AssumeRoleWithSAMLOutput, error) {
	if viper.GetBool("cache.cache_only") {
		return nil, errors.New("Could not find credentials in cache and --cache-only specified. Exiting.")
	}

	role, err := login.GetLoginRole(desiredRole)

	if err != nil {
		return nil, err
	}

	return aws.AssumeRole(login, role, viper.GetInt64("aws.session_duration"))
}

func isIamRoleArn(roleName string) bool {
	return arn.IsARN(roleName)
}

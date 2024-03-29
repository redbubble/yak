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
	log "github.com/sirupsen/logrus"
)

var notARoleErrorMessage = `'%s' is neither an IAM role ARN nor a configured alias.

Run 'yak --list-roles' to see which roles and aliases you can use.`

func AssumeRole(role string) (*sts.AssumeRoleWithSAMLOutput, error) {
	creds := getAssumedRoleFromCache(role)

	if creds == nil {
		log.Infof("Role %s not in cache", role)
		if viper.GetBool("cache.cache_only") {
			return nil, errors.New("Could not find credentials in cache and --cache-only specified. Run `yak <role>` to remedy.")
		}

		loginData, err := GetLoginDataWithTimeout()

		if err != nil {
			return nil, err
		}

		CacheLoginRoles(loginData.Roles)
		creds, err = assumeRoleFromAws(loginData, role)

		if err != nil {
			return nil, err
		}

		log.WithField("role", creds).Debug("assume_role.go: Role assumption credentials from AWS")

		cache.WriteDefault(role, creds)
		cache.Export()
	}

	return creds, nil
}

func getAssumedRoleFromCache(role string) *sts.AssumeRoleWithSAMLOutput {
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

func assumeRoleFromAws(login saml.LoginData, desiredRole string) (*sts.AssumeRoleWithSAMLOutput, error) {
	log.Infof("Assuming role %s from AWS", desiredRole)

	role, err := login.GetLoginRole(desiredRole)

	if err != nil {
		return nil, err
	}

	return aws.AssumeRole(login, role, viper.GetInt64("aws.session_duration"))
}

func isIamRoleArn(roleName string) bool {
	return arn.IsARN(roleName)
}

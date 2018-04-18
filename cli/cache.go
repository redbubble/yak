package cli

import (
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/cache"
)

func SaveCacheWithCreds(roleName string, creds *sts.AssumeRoleWithSAMLOutput) error {
	if viper.GetBool("cache.no_cache") {
		return nil
	}

	cache.Write(roleName, creds)

	return cache.Export()
}

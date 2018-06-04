package cli

import (
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/cache"
)

func CacheCredentials(roleName string, creds *sts.AssumeRoleWithSAMLOutput) {
	if viper.GetBool("cache.no_cache") {
		return
	}

	cache.WriteDefault(roleName, creds)
}

func WriteCache() {
	if !viper.GetBool("cache.no_cache") {
		cache.Export()
	}
}

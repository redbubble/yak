package cache

import (
	"bufio"
	"encoding/gob"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/output"
)

var cacheHandle *cache.Cache

func Cache() *cache.Cache {
	if cacheHandle == nil {
		roleExpiryDuration := time.Duration(viper.GetInt("aws.session_duration")) * time.Second
		err := importCache(roleExpiryDuration)

		if err != nil {
			output.ErrorPrintf("Warning: Couldn't read cache from file: %v\n", err)
			cacheHandle = cache.New(roleExpiryDuration, roleExpiryDuration)
		}
	}

	return cacheHandle
}

func importCache(roleExpiryDuration time.Duration) error {
	cacheFile, err := os.Open(viper.GetString("cache.file_location"))
	defer cacheFile.Close()

	if err != nil {
		return err
	}

	gob.Register(sts.AssumeRoleWithSAMLOutput{})
	decoder := gob.NewDecoder(bufio.NewReader(cacheFile))
	var items map[string]cache.Item

	err = decoder.Decode(&items)

	if err != nil {
		return err
	}

	cacheHandle = cache.NewFrom(roleExpiryDuration, roleExpiryDuration, items)

	return nil
}

func Write(key string, value interface{}, duration time.Duration) {
	Cache().Set(key, value, duration)
}

func WriteDefault(key string, value interface{}) {
	Cache().SetDefault(key, value)
}

func Check(roleArn string) interface{} {
	creds, credsExist := Cache().Get(roleArn)

	if !credsExist {
		return nil
	}

	return creds
}

func Export() error {
	cacheFile, err := os.Create(viper.GetString("cache.file_location"))
	defer cacheFile.Close()

	if err != nil {
		return err
	}

	writer := bufio.NewWriter(cacheFile)
	gob.Register(sts.AssumeRoleWithSAMLOutput{})
	enc := gob.NewEncoder(writer)
	err = enc.Encode(Cache().Items())

	if err != nil {
		return err
	}

	return writer.Flush()
}

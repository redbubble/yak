package cache

import (
	"bufio"
	"encoding/gob"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	gocache "github.com/patrickmn/go-cache"
	"github.com/spf13/viper"
)

var cacheHandle *gocache.Cache

func cache() *gocache.Cache {
	if cacheHandle == nil {
		roleExpiryDuration := time.Duration(viper.GetInt("aws.session_duration")) * time.Second
		err := importCache(roleExpiryDuration)

		if err != nil {
			cacheHandle = gocache.New(roleExpiryDuration, roleExpiryDuration)
		}
	}

	return cacheHandle
}

func Enabled() bool {
	return !viper.GetBool("cache.no_cache")
}

func importCache(roleExpiryDuration time.Duration) error {
	cacheFile, err := os.Open(viper.GetString("cache.file_location"))
	defer cacheFile.Close()

	if err != nil {
		return err
	}

	gob.Register(sts.AssumeRoleWithSAMLOutput{})
	decoder := gob.NewDecoder(bufio.NewReader(cacheFile))
	var items map[string]gocache.Item

	err = decoder.Decode(&items)

	if err != nil {
		return err
	}

	cacheHandle = gocache.NewFrom(roleExpiryDuration, roleExpiryDuration, items)

	return nil
}

func Write(key string, value interface{}, duration time.Duration) {
	if !Enabled() {
		return
	}

	cache().Set(key, value, duration)
}

func WriteDefault(key string, value interface{}) {
	if !Enabled() {
		return
	}

	cache().SetDefault(key, value)
}

func Check(roleArn string) interface{} {
	if !Enabled() {
		return nil
	}

	creds, credsExist := cache().Get(roleArn)

	if !credsExist {
		return nil
	}

	return creds
}

func Export() error {
	if !Enabled() {
		return nil
	}

	cacheFile, err := os.Create(viper.GetString("cache.file_location"))
	defer cacheFile.Close()

	if err != nil {
		return err
	}

	writer := bufio.NewWriter(cacheFile)
	gob.Register(sts.AssumeRoleWithSAMLOutput{})
	enc := gob.NewEncoder(writer)
	err = enc.Encode(cache().Items())

	if err != nil {
		return err
	}

	return writer.Flush()
}

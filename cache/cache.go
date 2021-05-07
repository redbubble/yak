package cache

import (
	"bufio"
	"encoding/gob"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/service/sts"
	gocache "github.com/patrickmn/go-cache"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/okta"
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

func gobInit() {
	gob.Register(sts.AssumeRoleWithSAMLOutput{})
	gob.Register(okta.OktaSession{})
}

func importCache(roleExpiryDuration time.Duration) error {
	cacheFile, err := os.Open(viper.GetString("cache.file_location"))
	defer cacheFile.Close()

	if err != nil {
		return err
	}

	gobInit()
	decoder := gob.NewDecoder(bufio.NewReader(cacheFile))
	var items map[string]gocache.Item

	if err = decoder.Decode(&items); err != nil {
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

func Check(key string) interface{} {
	if !Enabled() {
		return nil
	}

	data, dataExists := cache().Get(key)

	if !dataExists {
		return nil
	}

	return data
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
	gobInit()
	enc := gob.NewEncoder(writer)
	if err = enc.Encode(cache().Items()); err != nil {
		return err
	}

	return writer.Flush()
}

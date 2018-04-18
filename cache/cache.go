package cache

import (
	"bufio"
	"encoding/gob"
	"os"
	"time"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"
)

var cacheHandle *cache.Cache

func Cache() *cache.Cache{
	if cacheHandle == nil {
		roleExpiryDuration := time.Duration(viper.GetInt("aws.session_duration")) * time.Second
		err := importCache(roleExpiryDuration)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Couldn't read cache from file: %v\n", err)
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

func Write(roleArn string, creds interface{}) {
	Cache().SetDefault(roleArn, creds)
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


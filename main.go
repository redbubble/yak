package main

import (
	"github.com/spf13/viper"

	"github.com/redbubble/yak/cmd"
)

var version string

func main() {
	viper.Set("yak.version", version)
	cmd.Execute()
}

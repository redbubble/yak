package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/cli"
	"github.com/redbubble/yak/format"
)

func printCredentialsCmd(cmd *cobra.Command, args []string) error {
	roleName, err := cli.ResolveRole(args[0])

	if err != nil {
		return err
	}

	creds, err := cli.AssumeRole(roleName)
	if err != nil {
		return err
	}

	output, err := format.Credentials(viper.GetString("output.format"), creds)

	if err != nil {
		return err
	}

	fmt.Print(output)

	return nil
}

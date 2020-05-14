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

	creds := cli.AssumeRoleFromCache(roleName)

	if creds == nil {
		loginData, err := cli.GetLoginDataWithTimeout()

		if err != nil {
			return err
		}

		cli.CacheLoginRoles(loginData.Roles)
		creds, err = cli.AssumeRole(loginData, roleName)

		if err != nil {
			return err
		}

		cli.CacheCredentials(roleName, creds)
		cli.WriteCache()
	}

	output, err := format.Credentials(viper.GetString("output.format"), creds)

	if err != nil {
		return err
	}

	fmt.Print(output)

	return nil
}

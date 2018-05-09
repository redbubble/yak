package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cli"
)

func printVarsCmd(cmd *cobra.Command, args []string) error {
	roleName := cli.ResolveRole(args[0])

	creds := cli.AssumeRoleFromCache(roleName)

	if creds == nil {

		loginData, err := cli.GetLoginData()

		if err != nil {
			return err
		}

		cli.CacheLoginRoles(loginData.Roles)
		creds, err = cli.AssumeRole(loginData, roleName)

		if err != nil {
			return err
		}

		cli.CacheCredentials(roleName, creds)
	}

	for key, value := range aws.EnvironmentVariables(creds.Credentials) {
		fmt.Printf("export %s=%s\n", key, value)
	}

	return nil
}

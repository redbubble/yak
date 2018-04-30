package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cli"
	"github.com/redbubble/yak/output"
)

func printVarsCmd(cmd *cobra.Command, args []string) {
	roleName := cli.ResolveRole(args[0])

	creds := cli.AssumeRoleFromCache(roleName)

	if creds == nil {

		loginData, err := cli.GetLoginData()

		if err != nil {
			output.ErrorPrintf("%v\n", err)
			os.Exit(1)
		}

		cli.CacheLoginRoles(loginData.Roles)
		creds, err = cli.AssumeRole(loginData, roleName)

		if err != nil {
			output.ErrorPrintf("%v\n", err)
			os.Exit(1)
		}

		cli.CacheCredentials(roleName, creds)
	}

	for key, value := range aws.EnvironmentVariables(creds.Credentials) {
		fmt.Printf("export %s='%s'\n", key, value)
	}
}

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cli"
)

func printVarsCmd(cmd *cobra.Command, args []string) {
	roleName := cli.ResolveRole(args[0])

	creds := cli.AssumeRoleFromCache(roleName)

	if creds == nil {

		loginData, err := cli.GetLoginData()

		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		creds, err = cli.AssumeRole(loginData, roleName)

		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		cli.SaveCacheWithCreds(roleName, creds)
	}

	for key, value := range aws.EnvironmentVariables(creds.Credentials) {
		fmt.Printf("export %s='%s'\n", key, value)
	}
}

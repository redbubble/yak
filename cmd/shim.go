package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cli"
)

func shimCmd(cmd *cobra.Command, args []string) {
	roleName := cli.ResolveRole(args[0])
	command := args[1:]

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

		cli.CacheCredentials(roleName, creds)
	}

	err := cli.Exec(
		command,
		cli.EnrichedEnvironment(
			aws.EnvironmentVariables(creds.Credentials),
		),
	)

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

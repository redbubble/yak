package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cli"
)

func shimCmd(cmd *cobra.Command, args []string) {
	desiredRole := args[0]
	command := args[1:]

	loginData, err := cli.GetLoginData()

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	role, err := cli.AssumeRole(loginData, desiredRole)

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	err = cli.Exec(
		command,
		cli.EnrichedEnvironment(
			aws.EnvironmentVariables(role.Credentials),
		),
	)

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

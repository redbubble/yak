package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cli"
)

func printVarsCmd(cmd *cobra.Command, args []string) {
	desiredRole := args[0]

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

	for key, value := range aws.EnvironmentVariables(role.Credentials) {
		fmt.Printf("export %s='%s'\n", key, value)
	}
}

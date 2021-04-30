package cmd

import (
	"github.com/spf13/cobra"

	"github.com/redbubble/yak/aws"
	"github.com/redbubble/yak/cli"
)

func shimCmd(cmd *cobra.Command, args []string) error {
	roleName, err := cli.ResolveRole(args[0])

	if err != nil {
		return err
	}

	command := args[1:]

	creds, err := cli.AssumeRole(roleName)
	if err != nil {
		return err
	}

	return cli.Exec(
		command,
		cli.EnrichedEnvironment(
			aws.EnvironmentVariables(creds),
		),
	)
}

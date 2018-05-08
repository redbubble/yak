package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

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
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		cli.CacheLoginRoles(loginData.Roles)
		creds, err = cli.AssumeRole(loginData, roleName)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
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
		exitError, ok := err.(*exec.ExitError)
		if ok {
			os.Exit(getExitCode(exitError))
		} else {
			// In this case, something went wrong, but the subprocess didn't return an error code; we should output an
			// error message because it's likely nothing went to stderr.
			fmt.Printf("%v\n", err)
			// 126 represents 'command invoked cannot execute', which seems like a reasonable default
			os.Exit(126)
		}
	}
}

func getExitCode(err *exec.ExitError) int {
	ws := err.Sys().(syscall.WaitStatus)
	return ws.ExitStatus()
}

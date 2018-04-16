package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/cli"
)

func listRolesCmd(cmd *cobra.Command, args []string) {
	loginData, err := cli.GetLoginData()

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nAvailable Roles:")
	for _, role := range loginData.Roles {
		fmt.Printf("    %s\n", role.RoleArn)
	}
	fmt.Println()
}

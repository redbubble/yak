package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/redbubble/yak/cli"
)

func listRolesCmd(cmd *cobra.Command, args []string) {
	roles, gotRoles := cli.GetRolesFromCache()

	if gotRoles {
		loginData, err := cli.GetLoginData()

		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		cli.CacheLoginRoles(loginData.Roles)

		roles = (loginData.Roles)
	}

	fmt.Println("\nAvailable Roles:")
	for _, role := range roles {
		fmt.Printf("    %s\n", role.RoleArn)
	}
	fmt.Println()
}

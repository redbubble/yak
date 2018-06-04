package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/cli"
)

func listRolesCmd(cmd *cobra.Command, args []string) error {
	roles, gotRoles := cli.GetRolesFromCache()

	if !gotRoles {
		loginData, err := cli.GetLoginData()

		if err != nil {
			return err
		}

		cli.CacheLoginRoles(loginData.Roles)
		cli.WriteCache()

		roles = (loginData.Roles)
	}

	aliases, _ := getAliases()

	for _, alias := range aliases {
		fmt.Printf("    %s\n", alias)
	}

	for _, role := range roles {
		fmt.Printf("    %s\n", role.RoleArn)
	}
	fmt.Println()

	return nil
}

func getAliases() ([]string, error) {
	var aliases map[string]string

	if !viper.IsSet("alias") {
		return []string{}, nil
	}

	err := viper.Sub("alias").Unmarshal(&aliases)

	if err != nil {
		return []string{}, err
	}

	keys := []string{}

	for key, _ := range aliases {
		keys = append(keys, key)
	}

	return keys, nil
}

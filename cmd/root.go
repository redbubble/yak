package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/redbubble/yak/cache"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "yak [flags] <role> [<subcommand>]",
	Short: "A command-line shim to do stuff with AWS credentials pulled from Okta",
	Long: `A command-line shim to do stuff with AWS credentials pulled from Okta

  If the --list-roles or -l flag is set, yak will log in to Okta and return
  the list of roles available in the SAML assertion. Otherwise, it will attempt
  to assume the <role> provided. If a <subcommand> is set, yak will attempt to
  run it with the credentials injected into its environment. Without a
  <subcommand>, the credentials will be printed to standard output inside export
  statements.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 && !viper.GetBool("list-roles") {
			cmd.Help()
			return
		}

		// The no-cache and cache-only flags are mutually exclusive, so bail out when both are specified
		if viper.GetBool("cache.no_cache") && viper.GetBool("cache.cache_only") {
			fmt.Fprintln(os.Stderr, "Please don't use --cache-only and --no-cache simultaneously.")
			return
		}

		if viper.GetBool("list-roles") {
			listRolesCmd(cmd, args)
		} else if len(args) == 1 {
			printVarsCmd(cmd, args)
		} else {
			shimCmd(cmd, args)
		}

		if !viper.GetBool("cache.no_cache") {
			cache.Export()
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initCache)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Display this help message and exit")
	rootCmd.PersistentFlags().BoolP("list-roles", "l", false, "List available AWS roles and exit")
	viper.BindPFlag("list-roles", rootCmd.PersistentFlags().Lookup("list-roles"))

	rootCmd.PersistentFlags().StringP("okta-username", "u", "", "Your Okta username")
	rootCmd.PersistentFlags().Int64P("aws-session-duration", "d", 0, "The session duration to request from AWS (in seconds)")
	rootCmd.PersistentFlags().Bool("no-cache", false, "Ignore cache for this request. Mutually exclusive with --cache-only")
	rootCmd.PersistentFlags().Bool("cache-only", false, "Only use cache, do not make external requests. Mutually exclusive with --no-cache")
	viper.BindPFlag("okta.username", rootCmd.PersistentFlags().Lookup("okta-username"))
	viper.BindPFlag("aws.session_duration", rootCmd.PersistentFlags().Lookup("aws-session-duration"))
	viper.BindPFlag("cache.no_cache", rootCmd.PersistentFlags().Lookup("no-cache"))
	viper.BindPFlag("cache.cache_only", rootCmd.PersistentFlags().Lookup("cache-only"))
}

func initCache() {
	dir, err := homedir.Dir()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	viper.SetDefault("cache.file_location", path.Join(dir, ".yak", "cache"))
}

func initConfig() {
	home, err := homedir.Dir()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	viper.AddConfigPath(path.Join(home, ".yak"))
	viper.SetConfigName("config")
	err = viper.ReadInConfig()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

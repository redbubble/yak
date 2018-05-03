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
	Use:   "yak [flags] [--list-roles | <role> [<subcommand...>]]",
	Short: "A shim to do stuff with AWS credentials using Okta",
	Long: `A shim to do stuff with AWS credentials using Okta

  * With --list-roles, print a list of your available AWS roles.
    Otherwise, yak will attempt to generate AWS keys for <role>.

  * If <subcommand> is set, yak will attempt to execute it with the
    AWS keys injected into the environment.  Otherwise, the
    credentials will conveniently be printed stdout.`,
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

		// If we've made it to this point, we need to have an Okta domain and an AWS path
		if viper.GetString("okta.domain") == "" || viper.GetString("okta.aws_saml_endpoint") == "" {
			fmt.Fprintln(os.Stderr, "An Okta domain and an AWS SAML Endpoint must be configured for yak to work.")
			fmt.Fprintln(os.Stderr, "These can be configured either in the [okta] section of ~/.config/yak/config.toml or by passing the --okta-domain and --okta-aws-saml-endpoint arguments.")
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
	cobra.OnInitialize(defaultConfigValues)
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initCache)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Display this help message and exit")
	rootCmd.PersistentFlags().BoolP("list-roles", "l", false, "List available AWS roles and exit")
	viper.BindPFlag("list-roles", rootCmd.PersistentFlags().Lookup("list-roles"))

	rootCmd.PersistentFlags().StringP("okta-username", "u", "", "Your Okta username")
	rootCmd.PersistentFlags().String("okta-domain", "", "The domain to use for requests to Okta")
	rootCmd.PersistentFlags().String("okta-aws-saml-endpoint", "", "The app embed path for the AWS app within Okta")
	rootCmd.PersistentFlags().Int64P("aws-session-duration", "d", 0, "The session duration to request from AWS (in seconds)")
	rootCmd.PersistentFlags().Bool("no-cache", false, "Ignore cache for this request. Mutually exclusive with --cache-only")
	rootCmd.PersistentFlags().Bool("cache-only", false, "Only use cache, do not make external requests. Mutually exclusive with --no-cache")
	viper.BindPFlag("okta.username", rootCmd.PersistentFlags().Lookup("okta-username"))
	viper.BindPFlag("okta.domain", rootCmd.PersistentFlags().Lookup("okta-domain"))
	viper.BindPFlag("okta.aws_saml_endpoint", rootCmd.PersistentFlags().Lookup("okta-aws-saml-endpoint"))
	viper.BindPFlag("aws.session_duration", rootCmd.PersistentFlags().Lookup("aws-session-duration"))
	viper.BindPFlag("cache.no_cache", rootCmd.PersistentFlags().Lookup("no-cache"))
	viper.BindPFlag("cache.cache_only", rootCmd.PersistentFlags().Lookup("cache-only"))
}

func initCache() {
	viper.SetDefault("cache.file_location", path.Join(getDataPath(), "cache"))
}

func initConfig() {
	viper.AddConfigPath(getConfigPath())

	viper.SetConfigName("config")
	viper.ReadInConfig()
}

func getDataPath() string {
	dataPath := os.Getenv("XDG_DATA_HOME")

	if dataPath == "" {
		home, err := homedir.Dir()

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		dataPath = path.Join(home, ".local", "share")
	}

	yakPath := path.Join(dataPath, "yak")

	if _, err := os.Stat(yakPath); os.IsNotExist(err) {
		os.MkdirAll(yakPath, 0700)
	}

	return yakPath
}

func getConfigPath() string {
	configPath := os.Getenv("XDG_CONFIG_HOME")

	if configPath == "" {
		home, err := homedir.Dir()

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		configPath = path.Join(home, ".config")
	}

	yakPath := path.Join(configPath, "yak")

	if _, err := os.Stat(yakPath); os.IsNotExist(err) {
		os.MkdirAll(yakPath, 0700)
	}

	return yakPath
}

func defaultConfigValues() {
	viper.SetDefault("aws.session_duration", 3600)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

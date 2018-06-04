package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/redbubble/yak/format"
)

var rootCmd = &cobra.Command{
	Use:   "yak [flags] [--list-roles | [--] <role> [<subcommand...>]]",
	Short: "A shim to do stuff with AWS credentials using Okta",
	Long: `A shim to do stuff with AWS credentials using Okta

  * With --list-roles, print a list of your available AWS roles.
    Otherwise, yak will attempt to generate AWS keys for <role>.

  * If <subcommand> is set, yak will attempt to execute it with the
    AWS keys injected into the environment.  Otherwise, the
    credentials will conveniently be printed stdout.

    Note that if you want to pass -/-- flags to your <subcommand>,
    you'll need to put a '--' separator before the <role> so yak
    knows not to interpret those arguments for itself`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error

		if viper.GetBool("version") {
			versionCmd()
			return nil
		}

		// The no-cache and cache-only flags are mutually exclusive, so bail out when both are specified
		if viper.GetBool("cache.no_cache") && viper.GetBool("cache.cache_only") {
			return errors.New("Please don't use --cache-only and --no-cache simultaneously.")
		}

		// Likewise, it doesn't make much sense to clear the cache if --no-cache was specified too
		if viper.GetBool("cache.no_cache") && viper.GetBool("clear-cache") {
			return errors.New("Please don't use --no-cache and --clear-cache simultaneously.")
		}

		// If we've made it to this point, we need to have an Okta domain and an AWS path
		if viper.GetString("okta.domain") == "" || viper.GetString("okta.aws_saml_endpoint") == "" {
			return errors.New(`An Okta domain and an AWS SAML Endpoint must be configured for yak to work.
These can be configured either in the [okta] section of ~/.config/yak/config.toml or by passing the --okta-domain and --okta-aws-saml-endpoint arguments.`)
		}

		// If the output format is invalid, exit here to provide consistent UX across all commands
		err = format.ValidateOutputFormat(viper.GetString("output.format"))
		if err != nil {
			return err
		}

		if viper.GetBool("clear-cache") {
			clearCache()

			if !viper.GetBool("list-roles") && len(args) == 0 {
				return nil
			}
		}

		channel := make(chan os.Signal, 2)
		signal.Notify(channel, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-channel
			fmt.Fprintln(os.Stderr, "Recieved termination signal, exiting...")
			os.Exit(1)
		}()

		if viper.GetBool("list-roles") {
			err = listRolesCmd(cmd, args)
		} else if len(args) == 1 {
			err = printCredentialsCmd(cmd, args)
		} else if len(args) > 1 {
			err = shimCmd(cmd, args)
		} else {
			cmd.Help()
		}

		return err
	},
}

func init() {
	cobra.OnInitialize(defaultConfigValues)
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initCache)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Display this help message and exit")
	rootCmd.PersistentFlags().BoolP("list-roles", "l", false, "List available AWS roles and exit")
	rootCmd.PersistentFlags().Bool("clear-cache", false, "Delete all data from yak's cache. If no other arguments are given, exit without error")
	rootCmd.PersistentFlags().Bool("version", false, "Print the current version and exit")
	viper.BindPFlag("list-roles", rootCmd.PersistentFlags().Lookup("list-roles"))
	viper.BindPFlag("clear-cache", rootCmd.PersistentFlags().Lookup("clear-cache"))
	viper.BindPFlag("version", rootCmd.PersistentFlags().Lookup("version"))

	rootCmd.PersistentFlags().StringP("okta-username", "u", "", "Your Okta username")
	rootCmd.PersistentFlags().String("okta-domain", "", "The domain to use for requests to Okta")
	rootCmd.PersistentFlags().String("okta-aws-saml-endpoint", "", "The app embed path for the AWS app within Okta")
	rootCmd.PersistentFlags().String("okta-mfa-type", "", "The Okta MFA type for login")
	rootCmd.PersistentFlags().String("okta-mfa-provider", "", "The Okta MFA provider name for login")
	rootCmd.PersistentFlags().StringP("output-format", "o", "", "Can be set to either 'json' or 'env'. The format in which to output credential data")
	rootCmd.PersistentFlags().Int64P("aws-session-duration", "d", 0, "The session duration to request from AWS (in seconds)")
	rootCmd.PersistentFlags().Bool("no-cache", false, "Ignore cache for this request. Mutually exclusive with --cache-only")
	rootCmd.PersistentFlags().Bool("cache-only", false, "Only use cache, do not make external requests. Mutually exclusive with --no-cache")
	viper.BindPFlag("okta.username", rootCmd.PersistentFlags().Lookup("okta-username"))
	viper.BindPFlag("okta.domain", rootCmd.PersistentFlags().Lookup("okta-domain"))
	viper.BindPFlag("okta.aws_saml_endpoint", rootCmd.PersistentFlags().Lookup("okta-aws-saml-endpoint"))
	viper.BindPFlag("okta.mfa_type", rootCmd.PersistentFlags().Lookup("okta-mfa-type"))
	viper.BindPFlag("okta.mfa_provider", rootCmd.PersistentFlags().Lookup("okta-mfa-provider"))
	viper.BindPFlag("aws.session_duration", rootCmd.PersistentFlags().Lookup("aws-session-duration"))
	viper.BindPFlag("cache.no_cache", rootCmd.PersistentFlags().Lookup("no-cache"))
	viper.BindPFlag("cache.cache_only", rootCmd.PersistentFlags().Lookup("cache-only"))
	viper.BindPFlag("output.format", rootCmd.PersistentFlags().Lookup("output-format"))
}

func versionCmd() {
	fmt.Printf("yak v%s\n", viper.GetString("yak.version"))

	yabytes, _ := base64.StdEncoding.DecodeString(`
IC8gICAgIFwKLyAgICAgICBcCiBcIF9fXyAvCiAgXG8gby9fX19fICAgICAg
eQogICB8dnwgdiB2IFxfX19fLwogICAgVSAgeSAgWSAgdiAgXAogICAgICBc
IFYgICBWIFkgLwogICAgICAgfHxWdlZ2Vnx8CiAgICAgICB8fCAgICAgfHwK`)
	var yascii = string(yabytes)
	fmt.Printf("\n%s\n", yascii)
}

func initCache() {
	viper.SetDefault("cache.file_location", path.Join(getCacheBasePath(), "cache"))
}

func clearCache() {
	os.Remove(viper.GetString("cache.file_location"))
}

func initConfig() {
	viper.AddConfigPath(getConfigPath())
	viper.AddConfigPath(oldConfigPath())

	viper.SetConfigName("config")
	viper.ReadInConfig()
}

func getCacheBasePath() string {
	dataPath := os.Getenv("XDG_CACHE_HOME")

	if dataPath == "" {
		home, err := homedir.Dir()

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		dataPath = path.Join(home, ".cache")
	}

	yakPath := path.Join(dataPath, "yak")

	if _, err := os.Stat(yakPath); os.IsNotExist(err) {
		os.MkdirAll(yakPath, 0700)
	}

	return yakPath
}

func oldConfigPath() string {
	home, err := homedir.Dir()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return path.Join(home, ".yak")
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
	viper.SetDefault("output.format", "env")
}

func Execute() {
	err := rootCmd.Execute()

	if err != nil {
		exitError, isExitError := err.(*exec.ExitError)

		if isExitError {
			// In this case, we had a subprocess and that subprocess returned an error code; we should return the same
			// exit code as it did.
			os.Exit(getExitCode(exitError))
		} else {
			// In this case, something went wrong, but there was either no subprocess or that subprocess didn't return
			// an error code; we should output an  error message because it's likely nothing went to stderr.
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
}

func getExitCode(err *exec.ExitError) int {
	ws := err.Sys().(syscall.WaitStatus)
	return ws.ExitStatus()
}

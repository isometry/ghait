package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/isometry/ghait"
	"github.com/isometry/ghait/provider"
)

var (
	version string = "snapshot"
	commit  string = "unknown"
	date    string = "unknown"
)

func main() {
	_ = rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:          "ghait [flags]",
	Short:        "Generate an ephemeral GitHub App installation token",
	SilenceUsage: true,
	RunE:         runToken,
	Version:      fmt.Sprintf("%s, commit %s, built at %s", version, commit, date),
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().Int64P("app-id", "a", 0, "App ID (required)")
	viper.BindPFlag("app-id", rootCmd.PersistentFlags().Lookup("app-id"))

	rootCmd.PersistentFlags().Int64P("installation-id", "i", 0, "Installation ID (required)")
	viper.BindPFlag("installation-id", rootCmd.PersistentFlags().Lookup("installation-id"))

	rootCmd.PersistentFlags().StringP("key", "k", "", "Private key or identifier (required)")
	viper.BindPFlag("key", rootCmd.PersistentFlags().Lookup("key"))

	rootCmd.PersistentFlags().StringP("provider", "P", "file", fmt.Sprintf("KMS provider (supported: [%s])", strings.Join(provider.Registered(), ",")))
	viper.BindPFlag("provider", rootCmd.PersistentFlags().Lookup("provider"))

	rootCmd.PersistentFlags().StringSliceP("repository", "r", nil, "Repository names to grant access to (default all)")
	viper.BindPFlag("repository", rootCmd.PersistentFlags().Lookup("repository"))

	rootCmd.PersistentFlags().StringToStringP("permission", "p", nil, "Restricted permissions to grant")
	rootCmd.PersistentFlags().Lookup("permission").DefValue = "all"
	viper.BindPFlag("permission", rootCmd.PersistentFlags().Lookup("permission"))

	rootCmd.Flags().SortFlags = false
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("GHAIT")
}

func runToken(cmd *cobra.Command, args []string) error {
	config := ghait.NewConfig(
		viper.GetInt64("app-id"),
		viper.GetInt64("installation-id"),
		strings.ToLower(viper.GetString("provider")),
		viper.GetString("key"),
	)

	if config.GetAppID() == 0 {
		return errors.New("app-id is required")
	}

	if config.GetInstallationID() == 0 {
		return errors.New("installation-id is required")
	}

	factory, err := ghait.NewGHAIT(cmd.Context(), config)
	if err != nil {
		return err
	}

	permissions := &github.InstallationPermissions{}
	if err := mapstructure.Decode(viper.GetStringMapString("permission"), permissions); err != nil {
		return fmt.Errorf("decode permissions: %w", err)
	}

	tokenOptions := &github.InstallationTokenOptions{
		Repositories: viper.GetStringSlice("repository"),
		Permissions:  permissions,
	}

	token, err := factory.NewTokenWithOptions(cmd.Context(), tokenOptions)
	if err != nil {
		return err
	}

	fmt.Println(token.GetToken())
	fmt.Fprintf(cmd.ErrOrStderr(), "Expires at: %s\n", token.GetExpiresAt())

	return nil
}

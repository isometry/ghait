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
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

var rootCmd = &cobra.Command{
	Use:     "ghait [flags]",
	Short:   "Generate an ephemeral GitHub App installation token",
	RunE:    runToken,
	Version: fmt.Sprintf("%s, commit %s, built at %s", version, commit, date),
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().Int64P("app-id", "a", 0, "App ID (required)")
	viper.BindPFlag("app-id", rootCmd.Flags().Lookup("app-id"))

	rootCmd.Flags().Int64P("installation-id", "i", 0, "Installation ID (required)")
	viper.BindPFlag("installation-id", rootCmd.Flags().Lookup("installation-id"))

	rootCmd.Flags().StringP("key", "k", "", "Private key or identifier (required)")
	viper.BindPFlag("key", rootCmd.Flags().Lookup("key"))

	rootCmd.Flags().StringP("provider", "P", "file", fmt.Sprintf("KMS provider (supported: [%s])", strings.Join(provider.Registered(), ",")))
	viper.BindPFlag("provider", rootCmd.Flags().Lookup("provider"))

	rootCmd.Flags().StringSliceP("repository", "r", nil, "Repository names to grant access to (default all)")
	viper.BindPFlag("repository", rootCmd.Flags().Lookup("repository"))

	rootCmd.Flags().StringToStringP("permission", "p", nil, "Restricted permissions to grant")
	rootCmd.Flags().Lookup("permission").DefValue = "all"
	viper.BindPFlag("permission", rootCmd.Flags().Lookup("permission"))

	rootCmd.Flags().SortFlags = false
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("GHAIT")
}

func runToken(cmd *cobra.Command, args []string) error {
	config := ghait.Config{
		AppID:          viper.GetInt64("app-id"),
		InstallationID: viper.GetInt64("installation-id"),
		Provider:       strings.ToLower(viper.GetString("provider")),
		Key:            viper.GetString("key"),
	}

	if config.AppID == 0 {
		return errors.New("app-id is required")
	}

	if config.InstallationID == 0 {
		return errors.New("installation-id is required")
	}

	ghapp, err := ghait.NewGHAIT(cmd.Context(), &config)
	if err != nil {
		return err
	}

	permissions := &github.InstallationPermissions{}
	if err := mapstructure.Decode(viper.GetStringMapString("permission"), permissions); err != nil {
		return fmt.Errorf("decode permissions: %w", err)
	}

	options := &github.InstallationTokenOptions{
		Repositories: viper.GetStringSlice("repository"),
		Permissions:  permissions,
	}

	instToken, err := ghapp.NewInstallationToken(cmd.Context(), 0, options)
	if err != nil {
		return err
	}

	fmt.Println(instToken.GetToken())

	return nil
}

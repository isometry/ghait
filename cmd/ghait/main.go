// Package main is the entrypoint of the application
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v72/github"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/isometry/ghait"
	"github.com/isometry/ghait/provider"
)

var (
	version = "snapshot"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd := New()
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}

func New() *cobra.Command {
	var cmd = &cobra.Command{
		Use:          "ghait [flags]",
		Short:        "Generate an ephemeral GitHub App installation token",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
		RunE:    runToken,
		Version: fmt.Sprintf("%s, commit %s, built at %s", version, commit, date),
	}

	cobra.OnInitialize(initConfig)

	flags := cmd.Flags()

	flags.Int64P("app-id", "a", 0, "App ID (required)")
	flags.Int64P("installation-id", "i", 0, "Installation ID (required)")
	flags.StringP("key", "k", "", "Private key or identifier (required)")
	flags.StringP("provider", "P", "file", fmt.Sprintf("KMS provider (supported: [%s])", strings.Join(provider.Registered(), ",")))
	flags.StringSliceP("repository", "r", nil, "Repository names to grant access to (default all)")
	flags.StringToStringP("permission", "p", nil, "Restricted permissions to grant")
	flags.Lookup("permission").DefValue = "all"

	return cmd
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("GHAIT")
}

func runToken(cmd *cobra.Command, _ []string) error {
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
	if err = mapstructure.Decode(viper.GetStringMapString("permission"), permissions); err != nil {
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
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Expires at: %s\n", token.GetExpiresAt())

	return nil
}

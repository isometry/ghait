// Package ghait provides a simple interface for obtaining GitHub App Installation Tokens.
package ghait

// @Note: Individual providers are registered in separate files,
// allowing them to be conditionally disabled.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/google/go-github/v75/github"

	"github.com/isometry/ghait/provider"
)

// FatalError is returned when an error is considered fatal.
type FatalError struct{}

func (e FatalError) Error() string {
	return "fatal token error"
}

// TransientError is returned when an error is considered transient.
type TransientError struct{}

func (e TransientError) Error() string {
	return "transient token error"
}

func wrapTokenResponseError(resp *github.Response, err error) error {
	if resp != nil && resp.StatusCode == http.StatusGatewayTimeout {
		return errors.Join(TransientError{}, err)
	}
	return errors.Join(FatalError{}, err)
}

// GHAIT is the GitHub App Installation Token interface.
type GHAIT interface {
	GetAppID() int64
	GetInstallationID() int64
	NewInstallationToken(ctx context.Context, installationID int64, options *github.InstallationTokenOptions) (*github.InstallationToken, error)
	NewToken(ctx context.Context) (*github.InstallationToken, error)
	NewTokenWithOptions(ctx context.Context, options *github.InstallationTokenOptions) (*github.InstallationToken, error)
}

type ghait struct {
	appID          int64
	installationID int64
	Client         *github.Client
}

// NewGHAIT returns a new GitHub App Installation Token instance.
func NewGHAIT(ctx context.Context, cfg Config) (*ghait, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if cfg.GetAppID() == 0 {
		return nil, errors.New("no GitHub App ID configured")
	}

	var (
		signer provider.Provider
		err    error
	)

	if slices.Contains[[]string](provider.Registered(), cfg.GetProvider()) {
		signer, err = provider.NewSigner(ctx, cfg.GetProvider(), cfg.GetKey())
		if err != nil {
			return nil, fmt.Errorf("%s signer: %w", cfg.GetProvider(), err)
		}
	} else {
		return nil, fmt.Errorf("unsupported provider: %s", cfg.GetProvider())
	}

	if err := signer.Check(); err != nil {
		return nil, fmt.Errorf("signer check: %w", err)
	}

	appsTransport, err := ghinstallation.NewAppsTransportWithOptions(
		http.DefaultTransport,
		cfg.GetAppID(),
		ghinstallation.WithSigner(signer),
	)
	if err != nil {
		return nil, fmt.Errorf("apps transport: %w", err)
	}

	rateLimitWaiterClient := github_ratelimit.NewClient(appsTransport)

	return &ghait{
		appID:          cfg.GetAppID(),
		installationID: cfg.GetInstallationID(),
		Client:         github.NewClient(rateLimitWaiterClient),
	}, nil
}

// GetAppID returns the GitHub App ID of the ghait instance.
func (g *ghait) GetAppID() int64 {
	return g.appID
}

// GetInstallationID returns the GitHub App Installation ID of the ghait instance.
func (g *ghait) GetInstallationID() int64 {
	return g.installationID
}

// NewInstallationToken returns a new GitHub App installation token for
// the specified installation, with optional override of the
// installation ID. If the installation ID is not provided, it will use
// that of the configured ghait instance.
// All errors are wrapped in a custom error type to allow for easy error
// classification: FatalError for errors that should not be retried,
// TransientError for errors that may be retried.
func (g *ghait) NewInstallationToken(ctx context.Context, installationID int64, options *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	if installationID == 0 {
		if g.installationID == 0 {
			return nil, wrapTokenResponseError(nil, errors.New("no GitHub App Installation ID configured"))
		}
		installationID = g.installationID
	}

	installationToken, resp, err := g.Client.Apps.CreateInstallationToken(ctx, installationID, options)
	if err != nil {
		return nil, fmt.Errorf("create installation token: %w", wrapTokenResponseError(resp, err))
	}

	return installationToken, nil
}

// NewToken returns a new GitHub App Installation Token for the
// configured installation with default options.
func (g *ghait) NewToken(ctx context.Context) (*github.InstallationToken, error) {
	return g.NewInstallationToken(ctx, 0, nil)
}

// NewTokenWithOptions returns a new GitHub App Installation Token for
// the configured installation with optional restrictions.
func (g *ghait) NewTokenWithOptions(ctx context.Context, options *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	return g.NewInstallationToken(ctx, 0, options)
}

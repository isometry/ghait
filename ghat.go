package ghait

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v66/github"

	"github.com/isometry/ghait/provider"
	// Individual providers are registered in separate files,
	// allowing them to be conditionally disabled.
)

type GHAIT interface {
	GetAppID() int64
	GetInstallationID() int64
	NewInstallationToken(ctx context.Context, installationId int64, options *github.InstallationTokenOptions) (*github.InstallationToken, error)
	NewToken(ctx context.Context) (*github.InstallationToken, error)
	NewTokenWithOptions(ctx context.Context, options *github.InstallationTokenOptions) (*github.InstallationToken, error)
}

type ghait struct {
	appID          int64
	installationID int64
	Client         *github.Client
}

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

	return &ghait{
		appID:          cfg.GetAppID(),
		installationID: cfg.GetInstallationID(),
		Client: github.NewClient(&http.Client{
			Transport: appsTransport,
		}),
	}, nil
}

func (g *ghait) GetAppID() int64 {
	return g.appID
}

func (g *ghait) GetInstallationID() int64 {
	return g.installationID
}

// NewInstallationToken returns a new GitHub App installation token for
// the specified installation, with optional override of the
// installation ID. If the installation ID is not provided, it will use
// that of the configured ghait instance.
func (g *ghait) NewInstallationToken(ctx context.Context, installationId int64, options *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	if installationId == 0 {
		if g.installationID == 0 {
			return nil, errors.New("no GitHub App Installation ID configured")
		}
		installationId = g.installationID
	}

	installationToken, _, err := g.Client.Apps.CreateInstallationToken(ctx, installationId, options)
	if err != nil {
		return nil, fmt.Errorf("create installation token: %w", err)
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

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

	_ "github.com/isometry/ghait/provider/file"
	// Additional KMS providers are registered in separate files,
	// allowing them to be conditionally disabled
)

type ghait struct {
	appID          int64
	installationID int64
	client         *github.Client
}

func NewGHAIT(ctx context.Context, cfg *Config) (*ghait, error) {
	var err error

	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	var signer provider.Provider

	if slices.Contains[[]string](provider.Registered(), cfg.Provider) {
		signer, err = provider.NewSigner(ctx, cfg.Provider, cfg.Key)
		if err != nil {
			return nil, fmt.Errorf("%s signer: %w", cfg.Provider, err)
		}
	} else {
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}

	if err := signer.Check(); err != nil {
		return nil, fmt.Errorf("signer check: %w", err)
	}

	appsTransport, err := ghinstallation.NewAppsTransportWithOptions(
		http.DefaultTransport,
		cfg.AppID,
		ghinstallation.WithSigner(signer),
	)
	if err != nil {
		return nil, fmt.Errorf("apps transport: %w", err)
	}

	return &ghait{
		appID:          cfg.AppID,
		installationID: cfg.InstallationID,
		client: github.NewClient(&http.Client{
			Transport: appsTransport,
		}),
	}, nil
}

func (g *ghait) NewInstallationToken(ctx context.Context, installationId int64, options *github.InstallationTokenOptions) (*github.InstallationToken, error) {
	if installationId == 0 {
		if g.installationID == 0 {
			return nil, errors.New("no GitHub App Installation ID configured")
		}
		installationId = g.installationID
	}

	installationToken, _, err := g.client.Apps.CreateInstallationToken(ctx, installationId, options)
	if err != nil {
		return nil, fmt.Errorf("create installation token: %w", err)
	}

	return installationToken, nil
}

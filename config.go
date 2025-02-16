package ghait

// Config represents the configuration for the provider.
type Config interface {
	GetAppID() int64
	GetInstallationID() int64
	GetProvider() string
	GetKey() string
}

type ghaitConfig struct {
	appID          int64  `mapstructure:"appId"`
	installationID int64  `mapstructure:"installationId"`
	provider       string `mapstructure:"provider"`
	key            string `mapstructure:"key"`
}

// NewConfig creates a new Config instance.
func NewConfig(appID int64, installationID int64, provider string, key string) *ghaitConfig {
	return &ghaitConfig{
		appID:          appID,
		installationID: installationID,
		provider:       provider,
		key:            key,
	}
}

// GetAppID returns the App ID.
func (c *ghaitConfig) GetAppID() int64 {
	return c.appID
}

// GetInstallationID returns the Installation ID.
func (c *ghaitConfig) GetInstallationID() int64 {
	return c.installationID
}

// GetProvider returns the provider.
func (c *ghaitConfig) GetProvider() string {
	return c.provider
}

// GetKey returns the key.
func (c *ghaitConfig) GetKey() string {
	return c.key
}

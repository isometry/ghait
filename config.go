package ghait

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

func NewConfig(appId int64, installationId int64, provider string, key string) *ghaitConfig {
	return &ghaitConfig{
		appID:          appId,
		installationID: installationId,
		provider:       provider,
		key:            key,
	}
}

func (c *ghaitConfig) GetAppID() int64 {
	return c.appID
}

func (c *ghaitConfig) GetInstallationID() int64 {
	return c.installationID
}

func (c *ghaitConfig) GetProvider() string {
	return c.provider
}

func (c *ghaitConfig) GetKey() string {
	return c.key
}

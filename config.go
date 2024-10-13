package ghait

type Config struct {
	AppID          int64  `mapstructure:"appId"`
	InstallationID int64  `mapstructure:"installationId"`
	Provider       string `mapstructure:"provider"`
	Key            string `mapstructure:"key"`
}

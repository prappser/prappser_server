package internal

import (
	"fmt"

	"github.com/prappser/prappser_server/internal/owner"
	"github.com/spf13/viper"
)

type Config struct {
	Owners owner.Config `mapstructure:"owners"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile("files/config.yaml")

	// Try to read the config and provide more detailed error information
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &config, nil
}

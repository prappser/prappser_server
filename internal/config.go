package internal

import (
	"fmt"
	"os"

	"github.com/prappser/prappser_server/internal/user"
	"github.com/spf13/viper"
)

type Config struct {
	Users       user.Config `mapstructure:"users"`
	Port        string      `mapstructure:"port"`
	ExternalURL string      `mapstructure:"external_url"`
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

	// Allow PORT environment variable to override config
	if port := os.Getenv("PORT"); port != "" {
		config.Port = port
	}

	// Default port
	if config.Port == "" {
		config.Port = "4545"
	}

	// Allow EXTERNAL_URL environment variable to override config
	if externalURL := os.Getenv("EXTERNAL_URL"); externalURL != "" {
		config.ExternalURL = externalURL
	}

	// If external URL is not set, construct from localhost and port
	if config.ExternalURL == "" {
		config.ExternalURL = fmt.Sprintf("http://localhost:%s", config.Port)
	}

	return &config, nil
}

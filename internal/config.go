package internal

import (
	"fmt"
	"os"

	"github.com/prappser/prappser_server/internal/user"
	"github.com/spf13/viper"
)

type Config struct {
	Users       user.Config `mapstructure:"users"`
	ServerURL   string      `mapstructure:"server_url"`
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

	// Allow SERVER_URL environment variable to override config
	if serverURL := os.Getenv("SERVER_URL"); serverURL != "" {
		config.ServerURL = serverURL
	}

	// Allow EXTERNAL_URL environment variable to override config
	if externalURL := os.Getenv("EXTERNAL_URL"); externalURL != "" {
		config.ExternalURL = externalURL
	}

	// Default to localhost for development
	if config.ServerURL == "" {
		config.ServerURL = "http://localhost:4545"
	}

	// If external URL is not set, use server URL
	// This is useful for development with ngrok/cloudflare tunnels
	if config.ExternalURL == "" {
		config.ExternalURL = config.ServerURL
	}

	return &config, nil
}

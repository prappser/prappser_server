package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/prappser/prappser_server/internal/user"
	"github.com/spf13/viper"
)

type Config struct {
	Users          user.Config `mapstructure:"users"`
	Port           string      `mapstructure:"port"`
	ExternalURL    string      `mapstructure:"external_url"`
	AllowedOrigins []string    `mapstructure:"allowed_origins"`
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

	// Allow ALLOWED_ORIGINS environment variable to override config
	// Format: comma-separated list, e.g., "https://prappser.app,http://localhost:8080"
	if originsEnv := os.Getenv("ALLOWED_ORIGINS"); originsEnv != "" {
		config.AllowedOrigins = strings.Split(originsEnv, ",")
		// Trim whitespace from each origin
		for i := range config.AllowedOrigins {
			config.AllowedOrigins[i] = strings.TrimSpace(config.AllowedOrigins[i])
		}
	}

	// Default to wildcard if no origins are configured
	if len(config.AllowedOrigins) == 0 {
		config.AllowedOrigins = []string{"*"}
	}

	return &config, nil
}

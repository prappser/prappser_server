package internal

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/prappser/prappser_server/internal/user"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	Users          user.Config `mapstructure:"users"`
	Port           string      `mapstructure:"port"`
	ExternalURL    string      `mapstructure:"external_url"`
	AllowedOrigins []string    `mapstructure:"allowed_origins"`
	MasterPassword string      // Original password for key derivation (not exported to config file)
}

func LoadConfig() (*Config, error) {
	viper.SetConfigFile("files/config.yaml")

	// Config file is optional - if it doesn't exist, we use env vars only
	if err := viper.ReadInConfig(); err != nil {
		log.Info().Msg("Config file not found, using environment variables only")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults for user config
	if config.Users.JWTExpirationHours == 0 {
		config.Users.JWTExpirationHours = 24
	}
	if config.Users.ChallengeTTLSec == 0 {
		config.Users.ChallengeTTLSec = 300
	}
	if config.Users.RegistrationTokenTTLSec == 0 {
		config.Users.RegistrationTokenTTLSec = 10
	}

	// Process master password: env var takes priority, then config file
	if password := os.Getenv("MASTER_PASSWORD"); password != "" {
		config.MasterPassword = password
		hash := md5.Sum([]byte(password))
		config.Users.MasterPasswordMD5Hash = hex.EncodeToString(hash[:])
	} else if password := viper.GetString("users.master_password"); password != "" {
		// Config file: plain password
		config.MasterPassword = password
		hash := md5.Sum([]byte(password))
		config.Users.MasterPasswordMD5Hash = hex.EncodeToString(hash[:])
	}

	// Allow JWT_EXPIRATION_HOURS environment variable to override config
	if jwtHours := os.Getenv("JWT_EXPIRATION_HOURS"); jwtHours != "" {
		if hours, err := strconv.Atoi(jwtHours); err == nil {
			config.Users.JWTExpirationHours = hours
		}
	}

	// Allow CHALLENGE_TTL_SEC environment variable to override config
	if ttl := os.Getenv("CHALLENGE_TTL_SEC"); ttl != "" {
		if seconds, err := strconv.Atoi(ttl); err == nil {
			config.Users.ChallengeTTLSec = seconds
		}
	}

	// Allow REGISTRATION_TOKEN_TTL_SEC environment variable to override config
	if ttl := os.Getenv("REGISTRATION_TOKEN_TTL_SEC"); ttl != "" {
		if seconds, err := strconv.Atoi(ttl); err == nil {
			config.Users.RegistrationTokenTTLSec = int32(seconds)
		}
	}

	// Validate required config - master password must be set
	if config.Users.MasterPasswordMD5Hash == "" {
		return nil, fmt.Errorf("MASTER_PASSWORD environment variable is required")
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

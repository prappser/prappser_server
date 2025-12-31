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

// maskString masks a string for safe logging (shows first 2 and last 2 chars)
func maskString(s string) string {
	if s == "" {
		return "(empty)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// maskPassword masks password in connection strings
func maskPassword(connStr string) string {
	if connStr == "" {
		return "(empty)"
	}
	// Simple mask - just show host part
	if strings.Contains(connStr, "@") {
		parts := strings.Split(connStr, "@")
		if len(parts) >= 2 {
			return "****@" + parts[len(parts)-1]
		}
	}
	return "****"
}

// resolveExternalURL resolves the external URL based on hosting provider
// For zeabur: "myapp" -> "https://myapp.zeabur.app"
// For zeabur: "https://myapp" -> "https://myapp.zeabur.app"
// For zeabur: "myapp.custom.com" -> "https://myapp.custom.com"
// For others: use as-is or localhost fallback
func resolveExternalURL(externalURL, hostingProvider, port string) string {
	if externalURL == "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}

	// Strip https:// or http:// prefix for analysis
	urlWithoutScheme := externalURL
	hasHTTPS := strings.HasPrefix(externalURL, "https://")
	hasHTTP := strings.HasPrefix(externalURL, "http://")
	if hasHTTPS {
		urlWithoutScheme = strings.TrimPrefix(externalURL, "https://")
	} else if hasHTTP {
		urlWithoutScheme = strings.TrimPrefix(externalURL, "http://")
	}

	// Check if it's a full domain (contains a dot)
	isFullDomain := strings.Contains(urlWithoutScheme, ".")

	if hostingProvider == "zeabur" && !isFullDomain {
		// Just a subdomain name, append .zeabur.app
		return fmt.Sprintf("https://%s.zeabur.app", urlWithoutScheme)
	}

	// Full domain or non-zeabur hosting - ensure https prefix
	if !hasHTTPS && !hasHTTP {
		return fmt.Sprintf("https://%s", externalURL)
	}

	return externalURL
}

func LoadConfig() (*Config, error) {
	// Debug: Print all relevant environment variables
	log.Info().
		Str("PORT", os.Getenv("PORT")).
		Str("EXTERNAL_URL", os.Getenv("EXTERNAL_URL")).
		Str("HOSTING_PROVIDER", os.Getenv("HOSTING_PROVIDER")).
		Str("DATABASE_URL", maskPassword(os.Getenv("DATABASE_URL"))).
		Str("MASTER_PASSWORD", maskString(os.Getenv("MASTER_PASSWORD"))).
		Str("ALLOWED_ORIGINS", os.Getenv("ALLOWED_ORIGINS")).
		Str("LOG_LEVEL", os.Getenv("LOG_LEVEL")).
		Msg("Environment variables on startup")

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

	// Resolve EXTERNAL_URL based on HOSTING_PROVIDER
	externalURL := os.Getenv("EXTERNAL_URL")
	hostingProvider := os.Getenv("HOSTING_PROVIDER")
	config.ExternalURL = resolveExternalURL(externalURL, hostingProvider, config.Port)

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

	// Debug: Print final config values
	log.Info().
		Str("Port", config.Port).
		Str("ExternalURL", config.ExternalURL).
		Strs("AllowedOrigins", config.AllowedOrigins).
		Int("JWTExpirationHours", config.Users.JWTExpirationHours).
		Int("ChallengeTTLSec", config.Users.ChallengeTTLSec).
		Bool("HasMasterPasswordHash", config.Users.MasterPasswordMD5Hash != "").
		Msg("Final config loaded")

	return &config, nil
}

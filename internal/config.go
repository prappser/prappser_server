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
)

type Config struct {
	Users          user.Config
	Port           string
	ExternalURL    string
	AllowedOrigins []string
	MasterPassword string
}

// Defaults
const (
	defaultPort                   = "4545"
	defaultJWTExpirationHours     = 24
	defaultChallengeTTLSec        = 300
	defaultRegistrationTokenTTLSec = 10
)

var defaultAllowedOrigins = []string{"https://prappser.app", "http://localhost:*", "https://localhost:*"}

func maskString(s string) string {
	if s == "" {
		return "(empty)"
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func maskPassword(connStr string) string {
	if connStr == "" {
		return "(empty)"
	}
	if strings.Contains(connStr, "@") {
		parts := strings.Split(connStr, "@")
		if len(parts) >= 2 {
			return "****@" + parts[len(parts)-1]
		}
	}
	return "****"
}

func resolveExternalURL(externalURL, hostingProvider, port string) string {
	if externalURL == "" {
		return fmt.Sprintf("http://localhost:%s", port)
	}

	urlWithoutScheme := externalURL
	hasHTTPS := strings.HasPrefix(externalURL, "https://")
	hasHTTP := strings.HasPrefix(externalURL, "http://")
	if hasHTTPS {
		urlWithoutScheme = strings.TrimPrefix(externalURL, "https://")
	} else if hasHTTP {
		urlWithoutScheme = strings.TrimPrefix(externalURL, "http://")
	}

	isFullDomain := strings.Contains(urlWithoutScheme, ".")

	if hostingProvider == "zeabur" && !isFullDomain {
		return fmt.Sprintf("https://%s.zeabur.app", urlWithoutScheme)
	}

	if !hasHTTPS && !hasHTTP {
		return fmt.Sprintf("https://%s", externalURL)
	}

	return externalURL
}

func LoadConfig() (*Config, error) {
	// Read all environment variables upfront
	envPort := os.Getenv("PORT")
	envExternalURL := os.Getenv("EXTERNAL_URL")
	envHostingProvider := os.Getenv("HOSTING_PROVIDER")
	envDatabaseURL := os.Getenv("DATABASE_URL")
	envMasterPassword := os.Getenv("MASTER_PASSWORD")
	envAllowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	envLogLevel := os.Getenv("LOG_LEVEL")
	envJWTExpirationHours := os.Getenv("JWT_EXPIRATION_HOURS")
	envChallengeTTLSec := os.Getenv("CHALLENGE_TTL_SEC")
	envRegistrationTokenTTLSec := os.Getenv("REGISTRATION_TOKEN_TTL_SEC")

	log.Info().
		Str("PORT", envPort).
		Str("EXTERNAL_URL", envExternalURL).
		Str("HOSTING_PROVIDER", envHostingProvider).
		Str("DATABASE_URL", maskPassword(envDatabaseURL)).
		Str("MASTER_PASSWORD", maskString(envMasterPassword)).
		Str("ALLOWED_ORIGINS", envAllowedOrigins).
		Str("LOG_LEVEL", envLogLevel).
		Msg("Environment variables on startup")

	// Validate required config
	if envMasterPassword == "" {
		return nil, fmt.Errorf("MASTER_PASSWORD environment variable is required")
	}

	// Build config with defaults and env overrides
	config := &Config{
		MasterPassword: envMasterPassword,
	}

	// Port
	if envPort != "" {
		config.Port = envPort
	} else {
		config.Port = defaultPort
	}

	// External URL
	config.ExternalURL = resolveExternalURL(envExternalURL, envHostingProvider, config.Port)

	// Allowed Origins
	if envAllowedOrigins != "" {
		origins := strings.Split(envAllowedOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		config.AllowedOrigins = origins
	} else {
		config.AllowedOrigins = defaultAllowedOrigins
	}

	// User config
	hash := md5.Sum([]byte(envMasterPassword))
	config.Users.MasterPasswordMD5Hash = hex.EncodeToString(hash[:])

	// JWT Expiration Hours
	if envJWTExpirationHours != "" {
		if hours, err := strconv.Atoi(envJWTExpirationHours); err == nil {
			config.Users.JWTExpirationHours = hours
		} else {
			config.Users.JWTExpirationHours = defaultJWTExpirationHours
		}
	} else {
		config.Users.JWTExpirationHours = defaultJWTExpirationHours
	}

	// Challenge TTL
	if envChallengeTTLSec != "" {
		if seconds, err := strconv.Atoi(envChallengeTTLSec); err == nil {
			config.Users.ChallengeTTLSec = seconds
		} else {
			config.Users.ChallengeTTLSec = defaultChallengeTTLSec
		}
	} else {
		config.Users.ChallengeTTLSec = defaultChallengeTTLSec
	}

	// Registration Token TTL
	if envRegistrationTokenTTLSec != "" {
		if seconds, err := strconv.Atoi(envRegistrationTokenTTLSec); err == nil {
			config.Users.RegistrationTokenTTLSec = int32(seconds)
		} else {
			config.Users.RegistrationTokenTTLSec = defaultRegistrationTokenTTLSec
		}
	} else {
		config.Users.RegistrationTokenTTLSec = defaultRegistrationTokenTTLSec
	}

	log.Info().
		Str("Port", config.Port).
		Str("ExternalURL", config.ExternalURL).
		Strs("AllowedOrigins", config.AllowedOrigins).
		Int("JWTExpirationHours", config.Users.JWTExpirationHours).
		Int("ChallengeTTLSec", config.Users.ChallengeTTLSec).
		Bool("HasMasterPasswordHash", config.Users.MasterPasswordMD5Hash != "").
		Msg("Final config loaded")

	return config, nil
}

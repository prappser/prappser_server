---
name: go-config
description: Configuration loading and structure patterns for the Go server
confidence: 93
scope:
  - "internal/config.go"
  - "internal/**/*.go"
---

# Configuration Convention

## Rules

1. All configuration is loaded from environment variables only — no config files, no `mapstructure`/Viper. The CLAUDE.md references `mapstructure` tags but the actual codebase uses raw `os.Getenv`.
2. The root `Config` struct lives in `internal/config.go`. Domain-specific sub-configs (e.g. `user.Config`) live in their own package and are embedded/referenced from root `Config`.
3. `LoadConfig()` in `internal/config.go` is the single entry point. It validates required vars, applies defaults, and returns `*Config, error`.
4. Required environment variables are validated at startup. Missing required vars return an error from `LoadConfig()` which causes `log.Fatal()` in `main.go`.
5. Optional environment variables use a helper `getEnvOrDefault(key, defaultVal string) string`.
6. Default values for numeric configs are declared as package-level `const` with `default` prefix: `defaultPort`, `defaultJWTExpirationHours`, `defaultChallengeTTLSec`.
7. Numeric env vars are parsed with `strconv.Atoi` / `strconv.ParseInt`. Parse errors are silently ignored and the default is kept — no panic on bad numeric input.
8. Config structs do not have validation methods — validation is inline in `LoadConfig()`.
9. Sensitive config values (passwords, keys) are never logged. They are only referenced by the services that need them.
10. The `Config` struct is passed by pointer to `NewRequestHandler` and used directly. Services receive only the specific sub-config they need (e.g. `user.Config`), not the full root `Config`.
11. No config struct tags (`mapstructure`, `env`, `yaml`) — plain Go structs with PascalCase fields.

## Example

```go
// internal/config.go
type Config struct {
    Users          user.Config
    Storage        StorageConfig
    Port           string
    ExternalURL    string
    AllowedOrigins []string
    MasterPassword string
}

const (
    defaultPort               = "4545"
    defaultJWTExpirationHours = 24
)

func getEnvOrDefault(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}

func LoadConfig() (*Config, error) {
    envMasterPassword := os.Getenv("MASTER_PASSWORD")
    if envMasterPassword == "" {
        return nil, fmt.Errorf("MASTER_PASSWORD environment variable is required")
    }

    config := &Config{MasterPassword: envMasterPassword}
    config.Port = getEnvOrDefault("PORT", defaultPort)
    // ... parse remaining vars with defaults
    return config, nil
}

// Domain sub-config in its own package (internal/user/user.go)
type Config struct {
    MasterPasswordMD5Hash   string
    JWTExpirationHours      int
    ChallengeTTLSec         int
    RegistrationTokenTTLSec int32
}
```

## Anti-pattern

```go
// WRONG: config file loading (no YAML/JSON/TOML config files)
viper.SetConfigFile("config.yaml")

// WRONG: mapstructure tags (not used in this codebase)
type Config struct {
    Port string `mapstructure:"port"`
}

// WRONG: passing full Config to every service
func NewUserService(config *internal.Config) *UserService { ... }
// prefer: pass only the sub-config the service needs
func NewUserService(config user.Config, ...) *UserService { ... }

// WRONG: panicking on missing optional env var
port := os.Getenv("PORT")
if port == "" {
    panic("PORT is required")  // use default instead
}
```

## Scope

- `internal/config.go`
- `internal/**/*.go`

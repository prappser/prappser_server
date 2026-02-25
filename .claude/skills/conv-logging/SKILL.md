---
name: go-logging
description: Logging patterns using zerolog (rs/zerolog) in the Go server
confidence: 95
scope:
  - "internal/**/*.go"
  - "main.go"
---

# Logging Convention

## Rules

1. Use `github.com/rs/zerolog/log` package-level logger throughout. Import as `"github.com/rs/zerolog/log"` and call `log.Info()`, `log.Error()`, etc. directly — no logger instance passed around.
2. Log level selection:
   - `log.Debug()` — detailed flow tracing, per-step progress inside complex operations (e.g. each step of JWT verification, event processing).
   - `log.Info()` — startup messages, successful completion of significant operations (server started, event accepted, keys loaded).
   - `log.Warn()` — recoverable issues, unexpected but non-fatal conditions (database not ready yet, unknown log level, unknown event type).
   - `log.Error()` — all handler-level errors before responding with 4xx/5xx, failed operations that are logged and returned.
   - `log.Fatal()` — only in `main.go` for unrecoverable startup failures. Calls `os.Exit(1)`.
3. Always chain `.Err(err)` before `.Msg(...)` when logging an error: `log.Error().Err(err).Msg("Failed to create user")`.
4. Attach structured fields before `.Msg(...)` using `.Str("key", value)`, `.Int("key", value)`, `.Bool("key", value)`, `.Int64("key", value)`, `.Time("key", value)`.
5. `.Msg(...)` text uses sentence case, no trailing period: `"Failed to get events"`, `"WebSocket hub started"`.
6. Complex multi-step operations (event processing, auth flow) prefix log messages with a bracket tag: `"[EVENT] Validation passed"`, `"[AUTH] Starting user authentication"`, `"[CHALLENGE] Challenge requested for user"`. This makes log filtering easy.
7. Log level is configured at startup from `LOG_LEVEL` environment variable via `zerolog.SetGlobalLevel(...)`. Default is `info`.
8. Never log raw sensitive data (passwords, full private keys, full JWT tokens). Truncate public keys: `publicKey[:min(50, len(publicKey))] + "..."`.
9. Log at `Info` level for: server startup, service initialization, successful key generation/loading, event accepted.
10. Log at `Debug` level for: individual steps in auth/event flows, per-request tracing. These are off by default in production.

## Example

```go
// Structured error log before responding
log.Error().Err(err).Msg("Failed to get events")
ctx.Error("Failed to get events", fasthttp.StatusInternalServerError)

// Structured info log with fields
log.Info().
    Str("eventId", event.ID).
    Str("type", string(event.Type)).
    Int64("sequence", event.SequenceNumber).
    Msg("[EVENT] Accepted successfully")

// Debug flow tracing with bracket prefix
log.Debug().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] Looking up user in database")

// Startup fatal
log.Fatal().Err(err).Msg("Error loading config")

// Warn for recoverable issue
log.Warn().Int("attempt", i+1).Err(err).Msg("Database not ready, retrying...")
```

## Anti-pattern

```go
// WRONG: fmt.Println or fmt.Printf for logging
fmt.Println("Server started")

// WRONG: passing logger as parameter (use package-level log)
func NewService(logger zerolog.Logger) *Service { ... }

// WRONG: logging after response (log first, then respond)
ctx.Error("Unauthorized", 401)
log.Error().Msg("Auth failed")

// WRONG: logging full sensitive values
log.Debug().Str("privateKey", hex.EncodeToString(privateKey)).Msg("Key loaded")

// WRONG: no structured fields on error
log.Error().Msg("failed")  // always attach .Err(err) when an error is available
```

## Scope

- `internal/**/*.go`
- `main.go`

---
name: go-architecture
description: Package structure, layering, and component organization for the Go server
confidence: 95
scope:
  - "internal/**/*.go"
  - "main.go"
---

# Architecture Convention

## Rules

1. All business logic lives under `internal/`. Each domain is a sub-package (e.g. `internal/user`, `internal/event`, `internal/application`).
2. Every domain package follows a strict 3-file layer split:
   - `{domain}.go` — domain model types, constants, and interfaces
   - `{domain}_repository.go` — concrete `*sql.DB` repository struct and SQL methods
   - `{domain}_service.go` — business logic, wires repository + external deps
   - `{domain}_endpoints.go` — fasthttp handler struct with one method per HTTP endpoint
3. Interfaces are defined in `{domain}.go` alongside the types that use them. The concrete implementation lives in a separate file.
4. Dependency injection is manual, done in `main.go` by wiring `New*` constructors top-to-bottom.
5. The HTTP router (`internal/http.go`) is a single `switch` statement on `path`. No external router library is used.
6. Middleware (auth, CORS) is implemented as function wrappers returning `fasthttp.RequestHandler`. Middleware structs live in `internal/middleware/`.
7. `internal/config.go` holds `Config` structs and `LoadConfig()` which reads purely from environment variables (no config files).
8. `internal/db.go` initializes `*sql.DB` and runs `golang-migrate` migrations from `files/migrations/`.
9. `internal/setup/` contains one-off operational endpoints (e.g. Railway token setup).
10. Sub-packages (`internal/user/owner`) are only created when a domain area becomes too large for a single package.

## Example

```
internal/
  config.go          <- Config structs + LoadConfig()
  db.go              <- NewDB() with migration
  http.go            <- NewRequestHandler() with path switch
  user/
    user.go           <- User struct, UserRepository interface, UserEndpoints struct, Config
    user_repository.go <- userRepository struct implementing UserRepository
    user_service.go   <- UserService struct with JWT methods
    owner/            <- sub-package for owner-specific JWE/JWS logic
  event/
    event.go          <- Event types, EventType constants
    event_repository.go
    event_service.go
    event_endpoints.go
    event_validator.go <- standalone validation functions
    event_authorizer.go
    event_cleanup.go
  middleware/
    auth.go
    cors.go
```

## Anti-pattern

```go
// WRONG: mixing all concerns in one file
// event.go that contains Event struct, SQL queries, HTTP handlers, and validation

// WRONG: creating interfaces for testing-only purposes
// Convention: only define an interface when the concrete type will have multiple implementations
// or when needed for cross-package use

// WRONG: using a router library
// The project uses a manual path switch in internal/http.go
```

## Scope

- `internal/**/*.go`
- `main.go`

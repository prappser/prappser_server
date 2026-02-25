---
name: go-middleware
description: Middleware patterns for auth and CORS using fasthttp
confidence: 93
scope:
  - "internal/middleware/*.go"
  - "internal/http.go"
---

# Middleware Convention

## Rules

1. Middleware lives in `internal/middleware/`. Each middleware is one file per concern: `auth.go`, `cors.go`.
2. Middleware is a struct with a `New{Name}Middleware(...)` constructor. It holds its dependencies (e.g. `*user.UserService`).
3. Middleware methods return `fasthttp.RequestHandler` wrapping another `fasthttp.RequestHandler` — the standard decorator pattern.
4. Auth middleware exposes two methods:
   - `RequireAuth(handler fasthttp.RequestHandler) fasthttp.RequestHandler` — validates JWT, sets `"user"` in context, calls handler.
   - `RequireRole(role string, handler fasthttp.RequestHandler) fasthttp.RequestHandler` — calls `RequireAuth` first, then checks role.
5. The authenticated user is stored in fasthttp context via `ctx.SetUserValue("user", authenticatedUser)`. Handlers retrieve it with `ctx.UserValue("user").(*user.User)`.
6. CORS middleware wraps the entire router handler (outermost layer). It is applied in `NewRequestHandler` as `return corsMiddleware.Handle(handler)`.
7. CORS allowed origins are configured at startup from `config.AllowedOrigins`. Localhost wildcard patterns (`http://localhost:*`) are matched via a compiled `*regexp.Regexp`.
8. OPTIONS preflight requests are handled entirely inside CORS middleware — they never reach domain handlers.
9. Middleware constructors are called once in `main.go` / `NewRequestHandler`, not per-request.
10. No third-party middleware libraries are used. All middleware is hand-written.

## Example

```go
// Auth middleware struct and constructor
type AuthMiddleware struct {
    userService *user.UserService
}

func NewAuthMiddleware(userService *user.UserService) *AuthMiddleware {
    return &AuthMiddleware{userService: userService}
}

// Decorator pattern — returns a new handler
func (am *AuthMiddleware) RequireAuth(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
    return func(ctx *fasthttp.RequestCtx) {
        authenticatedUser, err := am.userService.ValidateJWTFromRequest(ctx)
        if err != nil {
            log.Error().Err(err).Msg("Authentication failed")
            ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
            return
        }
        ctx.SetUserValue("user", authenticatedUser)
        handler(ctx)
    }
}

// Role check builds on RequireAuth
func (am *AuthMiddleware) RequireRole(role string, handler fasthttp.RequestHandler) fasthttp.RequestHandler {
    return am.RequireAuth(func(ctx *fasthttp.RequestCtx) {
        authenticatedUser := ctx.UserValue("user").(*user.User)
        if authenticatedUser.Role != role {
            ctx.Error("Forbidden", fasthttp.StatusForbidden)
            return
        }
        handler(ctx)
    })
}

// Applied in router
return corsMiddleware.Handle(handler)
```

## Anti-pattern

```go
// WRONG: middleware as a standalone function instead of a method
func AuthMiddleware(userService *user.UserService, handler fasthttp.RequestHandler) fasthttp.RequestHandler { ... }

// WRONG: creating middleware inside the request handler (per-request allocation)
func NewRequestHandler(...) fasthttp.RequestHandler {
    return func(ctx *fasthttp.RequestCtx) {
        authMiddleware := middleware.NewAuthMiddleware(userService)  // wrong: should be outside
        ...
    }
}

// WRONG: using a third-party middleware library
// All middleware is hand-written fasthttp decorators
```

## Scope

- `internal/middleware/*.go`
- `internal/http.go`

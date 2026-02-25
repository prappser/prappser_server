---
name: go-errors
description: Error handling, wrapping, and sentinel error patterns in the Go server
confidence: 92
scope:
  - "internal/**/*.go"
---

# Error Handling Convention

## Rules

1. Always wrap errors with context using `fmt.Errorf("description: %w", err)`. The description uses lowercase and describes what failed.
2. Return errors up the call stack from repository -> service -> endpoint. Never swallow errors silently.
3. Sentinel errors are declared as package-level `var` using `errors.New(...)` — e.g. `var ErrValidation = errors.New("validation error")`, `var ErrUnauthorized = errors.New("unauthorized")`. These live in the package where the error originates.
4. Sentinel errors are wrapped when returned with additional context: `fmt.Errorf("%w: applicationId is required", ErrValidation)`.
5. Endpoints check for sentinel errors using `errors.Is()` or string comparison (`err.Error() == "..."`) to select HTTP status codes. String comparison is used in this codebase when the service returns plain `fmt.Errorf` strings like `"application not found"`.
6. Repositories return `nil, nil` (not an error) when a row is not found via `sql.ErrNoRows` for single-item lookups where "not found" is a valid business state (e.g. `GetUserByPublicKey`). They return `fmt.Errorf("X not found")` when not-found is an error (e.g. bulk delete expecting a row to exist).
7. Never use `panic` in business logic. Fatal errors in `main.go` use `log.Fatal().Err(err).Msg(...)`.
8. Validation errors use the `ErrValidation` sentinel; authorization errors use `ErrUnauthorized`.
9. HTTP error responses use `ctx.Error(message, statusCode)` for plain text or `json.NewEncoder(ctx).Encode(map[string]interface{}{...})` for JSON error bodies. Endpoints that need structured errors (like `/events`) return JSON with `"accepted": false`.

## Example

```go
// Repository: return nil, nil for "not found" as valid state
func (r *userRepository) GetUserByPublicKey(publicKey string) (*User, error) {
    err := r.db.QueryRow(...).Scan(...)
    if err == sql.ErrNoRows {
        return nil, nil  // caller handles nil user
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user by public key: %w", err)
    }
    return &user, nil
}

// Service: wrap and add context
func (s *EventService) AcceptEvent(...) (*Event, error) {
    if err := ValidateEvent(event); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    ...
}

// Sentinel errors
var ErrValidation = errors.New("validation error")
return fmt.Errorf("%w: event.id is required", ErrValidation)

// Endpoint: map error to HTTP status
switch {
case err == ErrUnauthorized:
    ctx.Error("Forbidden", fasthttp.StatusForbidden)
case err == ErrValidation:
    ctx.Error("Bad Request", fasthttp.StatusBadRequest)
default:
    ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
}
```

## Anti-pattern

```go
// WRONG: swallowing error
result, _ := someFunc()

// WRONG: returning error without context
return err  // prefer: return fmt.Errorf("failed to do X: %w", err)

// WRONG: panic in business logic
panic("unexpected state")

// WRONG: sql.ErrNoRows bubbling up without conversion
return nil, sql.ErrNoRows  // convert to domain error or return nil, nil
```

## Scope

- `internal/**/*.go`

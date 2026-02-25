---
name: go-handlers
description: HTTP endpoint handler patterns using fasthttp
confidence: 95
scope:
  - "internal/**/*_endpoints.go"
  - "internal/**/endpoints.go"
---

# HTTP Handler Convention

## Rules

1. Endpoint structs are named `{Domain}Endpoints` (e.g. `UserEndpoints`, `EventEndpoints`). They hold a pointer to the service and any other dependencies (config, keys).
2. Constructor is `New{Domain}Endpoints(...)` returning a pointer to the concrete struct.
3. Each HTTP operation is one method on the endpoints struct. Method names match the action, not the HTTP verb: `RegisterApplication`, `GetEvents`, `SubmitEvent`, not `Post`, `Get`.
4. Every handler method signature is `func (e *{Domain}Endpoints) MethodName(ctx *fasthttp.RequestCtx)`.
5. The authenticated user is retrieved from context at the top of every protected handler: `authenticatedUser, ok := ctx.UserValue("user").(*user.User)`. If missing, respond 401 and return immediately.
6. Path parameters are set by the router (`internal/http.go`) via `ctx.SetUserValue("key", value)` and read in handlers via `ctx.UserValue("key").(string)`.
7. Request body is parsed with `json.Unmarshal(ctx.PostBody(), &req)`. Use `github.com/goccy/go-json` not `encoding/json` for JSON operations in handlers.
8. Query parameters are read via `ctx.QueryArgs().Peek("param")`.
9. Response pattern: always call `ctx.SetStatusCode(...)` then `ctx.SetContentType("application/json")` then `json.NewEncoder(ctx).Encode(...)`.
10. Error responses use `ctx.Error("Plain text message", fasthttp.StatusXxx)` for simple cases. Structured JSON errors use `ctx.SetStatusCode(...)` + `ctx.SetContentType("application/json")` + `json.NewEncoder(ctx).Encode(map[string]interface{}{...})`.
11. Log every error with `log.Error().Err(err).Msg("...")` before responding. Use `log.Debug()` for flow tracing in complex handlers.
12. Input validation is split: basic presence checks happen in the endpoint, domain rule validation happens in the service.
13. HTTP method dispatch for a single path is done in `internal/http.go` with a `switch method` block, not inside the endpoint method itself.

## Example

```go
type EventEndpoints struct {
    eventService *EventService
}

func NewEventEndpoints(eventService *EventService) *EventEndpoints {
    return &EventEndpoints{eventService: eventService}
}

func (ee *EventEndpoints) GetEvents(ctx *fasthttp.RequestCtx) {
    authenticatedUser, ok := ctx.UserValue("user").(*user.User)
    if !ok || authenticatedUser == nil {
        log.Error().Msg("Failed to get authenticated user from context")
        ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
        return
    }

    sinceEventID := string(ctx.QueryArgs().Peek("since"))

    response, err := ee.eventService.GetEventsSince(authenticatedUser.PublicKey, sinceEventID, 100)
    if err != nil {
        log.Error().Err(err).Msg("Failed to get events")
        ctx.Error("Failed to get events", fasthttp.StatusInternalServerError)
        return
    }

    ctx.SetStatusCode(fasthttp.StatusOK)
    ctx.SetContentType("application/json")
    json.NewEncoder(ctx).Encode(response)
}
```

## Anti-pattern

```go
// WRONG: encoding/json instead of goccy/go-json in handlers
import "encoding/json"  // use "github.com/goccy/go-json"

// WRONG: method dispatch inside handler
func (ee *EventEndpoints) Handle(ctx *fasthttp.RequestCtx) {
    if string(ctx.Method()) == "GET" { ... }  // belongs in http.go router
}

// WRONG: skipping the auth check
func (ee *EventEndpoints) GetEvents(ctx *fasthttp.RequestCtx) {
    // Missing: authenticatedUser check — always required in protected handlers
    response, _ := ee.eventService.GetEventsSince("", "", 100)
    ...
}

// WRONG: logging after responding
ctx.Error("not found", 404)
log.Error().Msg("...")  // log BEFORE responding
```

## Scope

- `internal/**/*_endpoints.go`
- `internal/**/endpoints.go`

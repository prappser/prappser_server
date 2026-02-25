---
name: go-naming
description: Naming conventions for packages, types, functions, variables, and constants in the Go server
confidence: 95
scope:
  - "internal/**/*.go"
  - "main.go"
---

# Naming Convention

## Rules

1. **Packages**: lowercase, single word preferred. Use underscore only when unavoidable (e.g. `bid_filtering`). Examples in this codebase: `user`, `event`, `application`, `middleware`, `websocket`, `keys`, `storage`, `invitation`, `setup`, `health`, `status`.
2. **Exported types**: PascalCase, descriptive of purpose — `UserService`, `EventEndpoints`, `ApplicationRepository`, `CORSMiddleware`.
3. **Unexported types**: camelCase — `userRepository`, `challengeInfo`, `userAuthJWSClaims`.
4. **Exported functions/constructors**: `New{Type}(...)` for constructors, verb-based for actions — `NewUserService`, `NewEventRepository`, `LoadConfig`, `ValidateJWT`, `GenerateChallenge`.
5. **Unexported functions**: camelCase verb-based — `extractJWTFromAuthorizationHeader`, `extractOriginFromURL`, `generateChallenge`.
6. **Method receivers**: short (1–2 char) abbreviation of the struct type — `us` for `UserService`, `ee` for `EventEndpoints`, `ae` for `ApplicationEndpoints`, `r` for repository structs, `am` for `AuthMiddleware`, `cm` for `CORSMiddleware`.
7. **Exported constants**: PascalCase — `RoleOwner`, `MemberRoleOwner`, `SaltSize`, `NonceSize`.
8. **Unexported constants**: camelCase — `defaultPort`, `defaultJWTExpirationHours`, `headerAuthorization`, `headerBearer`.
9. **Interfaces**: named for what the implementer provides, not what the consumer needs — `UserRepository`, `ApplicationRepository`, `EventBroadcaster`. Avoid `I`-prefix or `-er` tacked on mechanically.
10. **Endpoint structs**: named `{Domain}Endpoints` — `UserEndpoints`, `EventEndpoints`, `ApplicationEndpoints`. Exception: `storage.Endpoints` (no domain prefix since it is unambiguous in package scope).
11. **Repository structs**: named `{Domain}Repository` when exported as an interface; the concrete unexported impl is `{domain}Repository` (lowercase first letter) — e.g. interface `UserRepository` / impl `userRepository`.
12. **Service structs**: `{Domain}Service` — `UserService`, `EventService`, `ApplicationService`, `KeyService`.

## Example

```go
// Package: user
type UserRepository interface { ... }   // exported interface
type userRepository struct { db *sql.DB } // unexported impl

func NewUserRepository(db *sql.DB) UserRepository { ... }  // constructor returns interface

type UserService struct { ... }
func NewUserService(...) *UserService { ... }

func (us *UserService) ValidateJWT(token string) (*User, error) { ... }
func (us *UserService) GenerateJWT(user *User) (string, int64, error) { ... }

const (
    RoleOwner           = "owner"      // exported
    headerAuthorization = "Authorization" // unexported
)
```

## Anti-pattern

```go
// WRONG: IUserRepository (I-prefix)
// WRONG: UserRepositoryInterface
// WRONG: srv as receiver for UserService (too generic — use us)
// WRONG: exported constant defaultPort
// WRONG: GetUserSvc (abbreviation in public name)
```

## Scope

- `internal/**/*.go`
- `main.go`

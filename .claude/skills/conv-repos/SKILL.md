---
name: go-repos
description: Repository pattern — struct design, SQL style, and interface placement
confidence: 95
scope:
  - "internal/**/*_repository.go"
  - "internal/**/repository.go"
---

# Repository Convention

## Rules

1. Every domain has one repository file. File is named `{domain}_repository.go` or `repository.go` (e.g. `user_repository.go`, `application/repository.go`).
2. The repository interface is defined in `{domain}.go` alongside the domain types that use it. The concrete struct lives in the repository file.
3. Concrete repository structs are unexported lowercase: `userRepository`, not `UserRepository`. Exception: `application.Repository` and `event.EventRepository` are exported — acceptable when only one implementation exists and no interface ambiguity.
4. Constructors accept `*sql.DB` and return the interface (when one exists) or the concrete pointer: `func NewUserRepository(db *sql.DB) UserRepository`.
5. All SQL is raw string literals — no ORM. Multi-line queries use backtick raw strings with consistent indentation.
6. Query constants are inline (not extracted to package-level vars) unless reused in multiple methods.
7. Parameterized queries use PostgreSQL `$1, $2, ...` placeholders. Never string-interpolate values into SQL.
8. `rows.Close()` is always deferred immediately after a successful `db.Query(...)` call.
9. `rows.Err()` is always checked after the scan loop and returned.
10. For single-row queries: use `db.QueryRow(...).Scan(...)`. Check `sql.ErrNoRows` explicitly.
11. "Not found" handling: return `nil, nil` when the caller is expected to handle absence as a valid state (e.g. `GetUserByPublicKey`). Return `fmt.Errorf("X not found")` when the row must exist (e.g. delete, update expecting rowsAffected > 0).
12. After `db.Exec(...)` for update/delete operations, check `result.RowsAffected()` and return an error if 0 rows were affected.
13. JSON columns (`data`, `avatar_bytes`) are marshaled/unmarshaled manually in the repository using `encoding/json` or `encoding/base64`. Domain structs use Go types (`map[string]interface{}`, `string`); the DB layer handles conversion.

## Example

```go
// Interface in domain file (user.go)
type UserRepository interface {
    CreateUser(user *User) error
    GetUserByPublicKey(publicKey string) (*User, error)
    UpdateUserRole(publicKey string, role string) error
}

// Concrete impl in user_repository.go
type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) GetUserByPublicKey(publicKey string) (*User, error) {
    var user User
    err := r.db.QueryRow(
        "SELECT public_key, username, role, created_at FROM users WHERE public_key = $1",
        publicKey,
    ).Scan(&user.PublicKey, &user.Username, &user.Role, &user.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user by public key: %w", err)
    }
    return &user, nil
}

func (r *userRepository) DeleteMember(memberID string) error {
    result, err := r.db.Exec("DELETE FROM members WHERE id = $1", memberID)
    if err != nil {
        return err
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rowsAffected == 0 {
        return fmt.Errorf("member not found")
    }
    return nil
}
```

## Anti-pattern

```go
// WRONG: string interpolation in SQL (SQL injection risk)
query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", id)

// WRONG: forgetting rows.Close() or rows.Err()
rows, _ := r.db.Query(query, args...)
for rows.Next() { ... }
// missing: defer rows.Close() and return nil, rows.Err()

// WRONG: using an ORM or query builder
// This codebase uses raw SQL only via database/sql + lib/pq

// WRONG: defining interface in the repository file instead of the domain file
```

## Scope

- `internal/**/*_repository.go`
- `internal/**/repository.go`

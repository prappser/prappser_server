---
name: go-database
description: Database initialization, migrations, and connection patterns for the Go server
confidence: 92
scope:
  - "internal/db.go"
  - "files/migrations/*.sql"
---

# Database Convention

## Rules

1. Database initialization lives in `internal/db.go`. The single exported function `NewDB() (*sql.DB, error)` opens the connection and runs migrations before returning.
2. PostgreSQL is the only supported database. Driver is `github.com/lib/pq`, imported as a blank import `_ "github.com/lib/pq"` in `main.go`.
3. Connection string is read from `DATABASE_URL` environment variable only. No other connection config.
4. Migrations use `github.com/golang-migrate/migrate/v4` with SQL files stored in `files/migrations/`. Migration files follow the naming pattern `{6-digit-sequence}_{description}.{up|down}.sql` (e.g. `000001_init.up.sql`).
5. `NewDB()` retries the PostgreSQL connection up to 10 times with linear backoff (`time.Sleep(time.Duration(i+1) * time.Second)`) to handle slow container startup.
6. `migrate.Up()` is called on every startup. `migrate.ErrNoChange` is not an error — it is explicitly ignored.
7. `*sql.DB` is passed directly to repository constructors. It is not wrapped or abstracted beyond the repository interface.
8. No connection pool configuration is set explicitly — Go's `database/sql` defaults are used.
9. SQL schema is NOT dropped and recreated on startup (unlike the Flutter app). Migrations are additive. This is a real server with persistent data.
10. JSON data in columns (e.g. `data` in `events`, `components`) is stored as `TEXT`/`JSONB` and marshaled/unmarshaled in repository methods — not in the DB layer.
11. Timestamps are stored as `int64` Unix epoch seconds, not as `TIMESTAMP` columns with timezone. Models use `int64` fields for `CreatedAt`, `UpdatedAt`.

## Example

```go
// internal/db.go
func NewDB() (*sql.DB, error) {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        return nil, fmt.Errorf("DATABASE_URL environment variable is required")
    }

    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return nil, fmt.Errorf("failed to open db: %w", err)
    }

    // Retry with backoff for container startup
    var driver database.Driver
    for i := 0; i < 10; i++ {
        driver, err = postgres.WithInstance(db, &postgres.Config{})
        if err == nil {
            break
        }
        log.Warn().Int("attempt", i+1).Err(err).Msg("Database not ready, retrying...")
        time.Sleep(time.Duration(i+1) * time.Second)
    }

    dbMigrate, err := migrate.NewWithDatabaseInstance("file://files/migrations", "postgres", driver)
    if err != nil {
        return nil, fmt.Errorf("failed to create migrate: %w", err)
    }

    err = dbMigrate.Up()
    if err != nil && err != migrate.ErrNoChange {
        return nil, fmt.Errorf("failed to migrate: %w", err)
    }

    return db, nil
}

// Migration file naming: files/migrations/000004_add_storage.up.sql
// Passed to repository constructors directly:
keyRepo := keys.NewKeyRepository(db)
userRepo := user.NewUserRepository(db)
```

## Anti-pattern

```go
// WRONG: running migrations outside of NewDB
// Migrations must always run inside NewDB before returning *sql.DB

// WRONG: dropping and recreating tables on startup
// This is the server — data must persist across restarts
// (Unlike the Flutter app which drops tables in DEV mode)

// WRONG: wrapping *sql.DB in a custom type
type Database struct { db *sql.DB }
// Pass *sql.DB directly to repository constructors

// WRONG: hardcoded connection string
db, _ := sql.Open("postgres", "host=localhost user=postgres ...")
// Always read from DATABASE_URL env var
```

## Scope

- `internal/db.go`
- `files/migrations/*.sql`

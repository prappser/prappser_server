package internal

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const dbMigrationsPath = "file://files/migrations"

func NewDB() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	dbMigrate, err := migrate.NewWithDatabaseInstance(dbMigrationsPath, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate: %w", err)
	}

	err = dbMigrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return db, nil
}

package internal

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
)

type DB struct {
	db *sql.DB
}

const (
	sqlite3DBPath    = "files/prappser.db"
	dbMigrationsPath = "file://files/migrations"
)

func NewDB() (*DB, error) {
	db, err := sql.Open("sqlite3", sqlite3DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}
	migrate, err := migrate.NewWithDatabaseInstance(dbMigrationsPath, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate: %w", err)
	}
	err = migrate.Up()
	if err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}
	return &DB{db: db}, nil
}

package internal

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	sqlite3DBPath    = "files/prappser.db"
	dbMigrationsPath = "file://files/migrations"
)

func NewDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", sqlite3DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}
	dbMigrate, err := migrate.NewWithDatabaseInstance(dbMigrationsPath, "sqlite3", driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate: %w", err)
	}
	err = dbMigrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}
	return db, nil
}

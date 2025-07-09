package owner

import (
	"database/sql"
	"fmt"
)

type sqlite3OwnerRepository struct {
	db *sql.DB
}

func (r *sqlite3OwnerRepository) CreateOwner(owner *Owner) error {
	_, err := r.db.Query("INSERT INTO owners (public_key) VALUES (?)", owner.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create owner: %w", err)
	}
	return nil
}

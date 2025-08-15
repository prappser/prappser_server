package user

import (
	"database/sql"
	"fmt"
)

type sqlite3UserRepository struct {
	db *sql.DB
}

func NewSQLite3UserRepository(db *sql.DB) UserRepository {
	return &sqlite3UserRepository{db: db}
}

func (r *sqlite3UserRepository) CreateUser(user *User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (id, public_key, username, role, created_at) VALUES (?, ?, ?, ?, ?)",
		user.ID, user.PublicKey, user.Username, user.Role, user.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *sqlite3UserRepository) GetUserByPublicKey(publicKey string) (*User, error) {
	var user User
	err := r.db.QueryRow(
		"SELECT id, public_key, username, role, created_at FROM users WHERE public_key = ?",
		publicKey,
	).Scan(&user.ID, &user.PublicKey, &user.Username, &user.Role, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by public key: %w", err)
	}
	return &user, nil
}

func (r *sqlite3UserRepository) GetUserByID(id string) (*User, error) {
	var user User
	err := r.db.QueryRow(
		"SELECT id, public_key, username, role, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.PublicKey, &user.Username, &user.Role, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &user, nil
}

func (r *sqlite3UserRepository) GetUserByUsername(username string) (*User, error) {
	var user User
	err := r.db.QueryRow(
		"SELECT id, public_key, username, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.PublicKey, &user.Username, &user.Role, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

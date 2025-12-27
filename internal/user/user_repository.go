package user

import (
	"database/sql"
	"fmt"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(user *User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (public_key, username, role, created_at) VALUES ($1, $2, $3, $4)",
		user.PublicKey, user.Username, user.Role, user.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetUserByPublicKey(publicKey string) (*User, error) {
	var user User
	err := r.db.QueryRow(
		"SELECT public_key, username, role, created_at FROM users WHERE public_key = $1",
		publicKey,
	).Scan(&user.PublicKey, &user.Username, &user.Role, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by public key: %w", err)
	}
	return &user, nil
}

func (r *userRepository) GetUserByUsername(username string) (*User, error) {
	var user User
	err := r.db.QueryRow(
		"SELECT public_key, username, role, created_at FROM users WHERE username = $1",
		username,
	).Scan(&user.PublicKey, &user.Username, &user.Role, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func (r *userRepository) UpdateUserRole(publicKey string, role string) error {
	_, err := r.db.Exec(
		"UPDATE users SET role = $1 WHERE public_key = $2",
		role, publicKey,
	)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}

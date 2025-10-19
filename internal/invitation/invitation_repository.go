package invitation

import (
	"database/sql"
	"fmt"
)

// InvitationRepository defines the interface for invitation data access
type InvitationRepository interface {
	Create(invite *Invitation) error
	GetByID(id string) (*Invitation, error)
	Delete(id string) error
	IncrementUseCount(id string) error
	RecordUse(inviteID, userPublicKey string, useID string) error
	GetByApplicationID(appID string) ([]*Invitation, error)
}

// SQLiteInvitationRepository implements InvitationRepository using SQLite
type SQLiteInvitationRepository struct {
	db *sql.DB
}

// NewSQLiteInvitationRepository creates a new SQLiteInvitationRepository
func NewSQLiteInvitationRepository(db *sql.DB) *SQLiteInvitationRepository {
	return &SQLiteInvitationRepository{db: db}
}

// Create inserts a new invitation into the database
func (r *SQLiteInvitationRepository) Create(invite *Invitation) error {
	query := `
		INSERT INTO invitations (
			id, application_id, created_by_public_key,
			role, max_uses, used_count, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		invite.ID,
		invite.ApplicationID,
		invite.CreatedByPublicKey,
		invite.Role,
		invite.MaxUses,
		invite.UsedCount,
		invite.CreatedAt,
	)

	return err
}

// GetByID retrieves an invitation by its ID
func (r *SQLiteInvitationRepository) GetByID(id string) (*Invitation, error) {
	query := `
		SELECT id, application_id, created_by_public_key,
		       role, max_uses, used_count, created_at
		FROM invitations
		WHERE id = ?
	`

	invite := &Invitation{}
	err := r.db.QueryRow(query, id).Scan(
		&invite.ID,
		&invite.ApplicationID,
		&invite.CreatedByPublicKey,
		&invite.Role,
		&invite.MaxUses,
		&invite.UsedCount,
		&invite.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, err
	}

	return invite, nil
}

// Delete removes an invitation from the database (hard delete for revocation)
func (r *SQLiteInvitationRepository) Delete(id string) error {
	query := `DELETE FROM invitations WHERE id = ?`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}

	return nil
}

// IncrementUseCount increments the used_count for an invitation
func (r *SQLiteInvitationRepository) IncrementUseCount(id string) error {
	query := `
		UPDATE invitations
		SET used_count = used_count + 1
		WHERE id = ?
	`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}

	return nil
}

// RecordUse records that a user joined via an invitation
func (r *SQLiteInvitationRepository) RecordUse(inviteID, userPublicKey string, useID string) error {
	query := `
		INSERT INTO invitation_uses (id, invitation_id, user_public_key, used_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, useID, inviteID, userPublicKey, 0) // 0 will be replaced with actual timestamp in service
	return err
}

// GetByApplicationID retrieves all invitations for an application
func (r *SQLiteInvitationRepository) GetByApplicationID(appID string) ([]*Invitation, error) {
	query := `
		SELECT id, application_id, created_by_public_key,
		       role, max_uses, used_count, created_at
		FROM invitations
		WHERE application_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*Invitation
	for rows.Next() {
		invite := &Invitation{}
		err := rows.Scan(
			&invite.ID,
			&invite.ApplicationID,
			&invite.CreatedByPublicKey,
			&invite.Role,
			&invite.MaxUses,
			&invite.UsedCount,
			&invite.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, invite)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return invitations, nil
}

package application

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) CreateApplication(app *Application) error {
	query := `INSERT INTO applications (id, owner_public_key, user_public_key, name, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.Exec(query, app.ID, app.OwnerPublicKey, app.UserPublicKey, app.Name, app.CreatedAt, app.UpdatedAt)
	return err
}

func (r *SQLiteRepository) GetApplicationByID(id string) (*Application, error) {
	query := `SELECT id, owner_public_key, user_public_key, name, created_at, updated_at 
			  FROM applications WHERE id = ?`
	
	app := &Application{}
	err := r.db.QueryRow(query, id).Scan(
		&app.ID, &app.OwnerPublicKey, &app.UserPublicKey, &app.Name, &app.CreatedAt, &app.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("application not found")
	}
	if err != nil {
		return nil, err
	}
	
	// Load component groups
	groups, err := r.GetComponentGroupsByApplicationID(id)
	if err != nil {
		return nil, err
	}
	
	// Convert pointers to values and load components for each group
	app.ComponentGroups = make([]ComponentGroup, len(groups))
	for i, group := range groups {
		app.ComponentGroups[i] = *group
		
		components, err := r.GetComponentsByGroupID(group.ID)
		if err != nil {
			return nil, err
		}
		
		// Convert component pointers to values
		app.ComponentGroups[i].Components = make([]Component, len(components))
		for j, comp := range components {
			app.ComponentGroups[i].Components[j] = *comp
		}
	}
	
	// Load members
	members, err := r.GetMembersByApplicationID(id)
	if err != nil {
		return nil, err
	}
	
	// Convert member pointers to values
	app.Members = make([]Member, len(members))
	for i, member := range members {
		app.Members[i] = *member
	}
	
	return app, nil
}

func (r *SQLiteRepository) GetApplicationsByOwnerPublicKey(ownerPublicKey string) ([]*Application, error) {
	query := `SELECT id, owner_public_key, user_public_key, name, created_at, updated_at 
			  FROM applications WHERE owner_public_key = ? ORDER BY created_at DESC`
	
	rows, err := r.db.Query(query, ownerPublicKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var applications []*Application
	for rows.Next() {
		app := &Application{}
		err := rows.Scan(&app.ID, &app.OwnerPublicKey, &app.UserPublicKey, &app.Name, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, err
		}
		
		// Load component groups for this application
		groups, err := r.GetComponentGroupsByApplicationID(app.ID)
		if err != nil {
			return nil, err
		}
		
		// Convert pointers to values and load components for each group
		app.ComponentGroups = make([]ComponentGroup, len(groups))
		for i, group := range groups {
			app.ComponentGroups[i] = *group
			
			components, err := r.GetComponentsByGroupID(group.ID)
			if err != nil {
				return nil, err
			}
			
			// Convert component pointers to values
			app.ComponentGroups[i].Components = make([]Component, len(components))
			for j, comp := range components {
				app.ComponentGroups[i].Components[j] = *comp
			}
		}
		
		// Load members for this application
		members, err := r.GetMembersByApplicationID(app.ID)
		if err != nil {
			return nil, err
		}
		
		// Convert member pointers to values
		app.Members = make([]Member, len(members))
		for i, member := range members {
			app.Members[i] = *member
		}
		
		applications = append(applications, app)
	}
	
	return applications, rows.Err()
}

func (r *SQLiteRepository) GetApplicationState(id string) (*ApplicationState, error) {
	query := `SELECT id, name, updated_at FROM applications WHERE id = ?`
	
	state := &ApplicationState{}
	err := r.db.QueryRow(query, id).Scan(&state.ID, &state.Name, &state.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("application not found")
	}
	
	return state, err
}

func (r *SQLiteRepository) UpdateApplicationTimestamp(id string) error {
	query := `UPDATE applications SET updated_at = ? WHERE id = ?`
	
	_, err := r.db.Exec(query, time.Now().Unix(), id)
	return err
}

func (r *SQLiteRepository) DeleteApplication(id string) error {
	// Due to CASCADE constraints, deleting the application will automatically delete
	// all associated component_groups and components
	query := `DELETE FROM applications WHERE id = ?`
	
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("application not found")
	}
	
	return nil
}

func (r *SQLiteRepository) CreateComponentGroup(group *ComponentGroup) error {
	query := `INSERT INTO component_groups (id, application_id, name, index_order) 
			  VALUES (?, ?, ?, ?)`
	
	_, err := r.db.Exec(query, group.ID, group.ApplicationID, group.Name, group.Index)
	return err
}

func (r *SQLiteRepository) GetComponentGroupsByApplicationID(appID string) ([]*ComponentGroup, error) {
	query := `SELECT id, application_id, name, index_order 
			  FROM component_groups WHERE application_id = ? ORDER BY index_order`
	
	rows, err := r.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var groups []*ComponentGroup
	for rows.Next() {
		group := &ComponentGroup{}
		err := rows.Scan(&group.ID, &group.ApplicationID, &group.Name, &group.Index)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	
	return groups, rows.Err()
}

func (r *SQLiteRepository) CreateComponent(component *Component) error {
	query := `INSERT INTO components (id, component_group_id, application_id, name, data, index_order) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	var dataJSON string
	if component.Data != nil {
		dataBytes, err := json.Marshal(component.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal component data: %w", err)
		}
		dataJSON = string(dataBytes)
	}
	
	_, err := r.db.Exec(query, 
		component.ID, 
		component.ComponentGroupID, 
		component.ApplicationID, 
		component.Name, 
		dataJSON, 
		component.Index,
	)
	return err
}

func (r *SQLiteRepository) GetComponentsByGroupID(groupID string) ([]*Component, error) {
	query := `SELECT id, component_group_id, application_id, name, data, index_order 
			  FROM components WHERE component_group_id = ? ORDER BY index_order`
	
	rows, err := r.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var components []*Component
	for rows.Next() {
		comp := &Component{}
		var dataJSON sql.NullString
		err := rows.Scan(
			&comp.ID, 
			&comp.ComponentGroupID, 
			&comp.ApplicationID, 
			&comp.Name, 
			&dataJSON, 
			&comp.Index,
		)
		if err != nil {
			return nil, err
		}
		
		// Parse JSON data if present
		if dataJSON.Valid && dataJSON.String != "" {
			if err := json.Unmarshal([]byte(dataJSON.String), &comp.Data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal component data: %w", err)
			}
		}
		
		components = append(components, comp)
	}
	
	return components, rows.Err()
}

func (r *SQLiteRepository) GetComponentsByApplicationID(appID string) ([]*Component, error) {
	query := `SELECT id, component_group_id, application_id, name, data, index_order 
			  FROM components WHERE application_id = ? ORDER BY index_order`
	
	rows, err := r.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var components []*Component
	for rows.Next() {
		comp := &Component{}
		var dataJSON sql.NullString
		err := rows.Scan(
			&comp.ID, 
			&comp.ComponentGroupID, 
			&comp.ApplicationID, 
			&comp.Name, 
			&dataJSON, 
			&comp.Index,
		)
		if err != nil {
			return nil, err
		}
		
		// Parse JSON data if present
		if dataJSON.Valid && dataJSON.String != "" {
			if err := json.Unmarshal([]byte(dataJSON.String), &comp.Data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal component data: %w", err)
			}
		}
		
		components = append(components, comp)
	}
	
	return components, rows.Err()
}

func (r *SQLiteRepository) CreateMember(member *Member) error {
	query := `INSERT INTO members (id, application_id, name, role, public_key, avatar_bytes) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	_, err := r.db.Exec(query, member.ID, member.ApplicationID, member.Name, string(member.Role), member.PublicKey, member.AvatarBytes)
	return err
}

func (r *SQLiteRepository) GetMembersByApplicationID(appID string) ([]*Member, error) {
	query := `SELECT id, application_id, name, role, public_key, avatar_bytes 
			  FROM members WHERE application_id = ? ORDER BY role, name`
	
	rows, err := r.db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var members []*Member
	for rows.Next() {
		member := &Member{}
		var roleStr string
		var avatarBytes sql.NullString
		
		err := rows.Scan(
			&member.ID,
			&member.ApplicationID,
			&member.Name,
			&roleStr,
			&member.PublicKey,
			&avatarBytes,
		)
		if err != nil {
			return nil, err
		}
		
		member.Role = MemberRole(roleStr)
		if avatarBytes.Valid {
			member.AvatarBytes = []byte(avatarBytes.String)
		}
		
		members = append(members, member)
	}
	
	return members, rows.Err()
}

func (r *SQLiteRepository) GetMemberByID(memberID string) (*Member, error) {
	query := `SELECT id, application_id, name, role, public_key, avatar_bytes 
			  FROM members WHERE id = ?`
	
	member := &Member{}
	var roleStr string
	var avatarBytes sql.NullString
	
	err := r.db.QueryRow(query, memberID).Scan(
		&member.ID,
		&member.ApplicationID,
		&member.Name,
		&roleStr,
		&member.PublicKey,
		&avatarBytes,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}
	
	member.Role = MemberRole(roleStr)
	if avatarBytes.Valid {
		member.AvatarBytes = []byte(avatarBytes.String)
	}
	
	return member, nil
}

func (r *SQLiteRepository) UpdateMember(member *Member) error {
	query := `UPDATE members SET name = ?, role = ?, public_key = ?, avatar_bytes = ? 
			  WHERE id = ?`
	
	result, err := r.db.Exec(query, member.Name, string(member.Role), member.PublicKey, member.AvatarBytes, member.ID)
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

func (r *SQLiteRepository) DeleteMember(memberID string) error {
	query := `DELETE FROM members WHERE id = ?`

	result, err := r.db.Exec(query, memberID)
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

// GetMemberByPublicKey returns a member by public key for a specific application
func (r *SQLiteRepository) GetMemberByPublicKey(appID, publicKey string) (*Member, error) {
	query := `SELECT id, application_id, name, role, public_key, avatar_bytes
			  FROM members WHERE application_id = ? AND public_key = ?`

	member := &Member{}
	var roleStr string
	var avatarBytes sql.NullString

	err := r.db.QueryRow(query, appID, publicKey).Scan(
		&member.ID,
		&member.ApplicationID,
		&member.Name,
		&roleStr,
		&member.PublicKey,
		&avatarBytes,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}

	member.Role = MemberRole(roleStr)
	if avatarBytes.Valid {
		member.AvatarBytes = []byte(avatarBytes.String)
	}

	return member, nil
}

// GetApplicationsByMemberPublicKey returns all applications where the user is a member
func (r *SQLiteRepository) GetApplicationsByMemberPublicKey(publicKey string) ([]*Application, error) {
	query := `SELECT DISTINCT a.id, a.owner_public_key, a.user_public_key, a.name, a.created_at, a.updated_at
			  FROM applications a
			  INNER JOIN members m ON a.id = m.application_id
			  WHERE m.public_key = ?
			  ORDER BY a.created_at DESC`

	rows, err := r.db.Query(query, publicKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var applications []*Application
	for rows.Next() {
		app := &Application{}
		err := rows.Scan(&app.ID, &app.OwnerPublicKey, &app.UserPublicKey, &app.Name, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Load component groups for this application
		groups, err := r.GetComponentGroupsByApplicationID(app.ID)
		if err != nil {
			return nil, err
		}

		// Convert pointers to values and load components for each group
		app.ComponentGroups = make([]ComponentGroup, len(groups))
		for i, group := range groups {
			app.ComponentGroups[i] = *group

			components, err := r.GetComponentsByGroupID(group.ID)
			if err != nil {
				return nil, err
			}

			// Convert component pointers to values
			app.ComponentGroups[i].Components = make([]Component, len(components))
			for j, comp := range components {
				app.ComponentGroups[i].Components[j] = *comp
			}
		}

		// Load members
		members, err := r.GetMembersByApplicationID(app.ID)
		if err != nil {
			return nil, err
		}

		// Convert member pointers to values
		app.Members = make([]Member, len(members))
		for i, member := range members {
			app.Members[i] = *member
		}

		applications = append(applications, app)
	}

	return applications, rows.Err()
}

// IsMember checks if a user is a member of an application
func (r *SQLiteRepository) IsMember(appID, publicKey string) (bool, error) {
	query := `SELECT COUNT(*) FROM members WHERE application_id = ? AND public_key = ?`

	var count int
	err := r.db.QueryRow(query, appID, publicKey).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetMemberCount returns the number of members in an application
func (r *SQLiteRepository) GetMemberCount(appID string) (int, error) {
	query := `SELECT COUNT(*) FROM members WHERE application_id = ?`

	var count int
	err := r.db.QueryRow(query, appID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
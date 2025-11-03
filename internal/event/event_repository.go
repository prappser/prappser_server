package event

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// GetNextSequence retrieves the next sequence number for an application
func (r *EventRepository) GetNextSequence(applicationID string) (int64, error) {
	var maxSeq int64
	query := `SELECT COALESCE(MAX(sequence_number), 0)
			  FROM events
			  WHERE application_id = ?`

	err := r.db.QueryRow(query, applicationID).Scan(&maxSeq)
	if err != nil {
		return 0, fmt.Errorf("failed to get next sequence: %w", err)
	}

	return maxSeq + 1, nil
}

// Create inserts a new event into the database
func (r *EventRepository) Create(event *Event) error {
	// Set created_at if not already set
	if event.CreatedAt == 0 {
		event.CreatedAt = time.Now().Unix()
	}

	// Extract and set application ID from event data if not already set
	if event.ApplicationID == "" {
		appID, ok := event.Data["applicationId"].(string)
		if ok && appID != "" {
			event.ApplicationID = appID
		}
	}

	// Assign sequence number if not already set
	if event.SequenceNumber == 0 {
		if event.ApplicationID != "" {
			seq, err := r.GetNextSequence(event.ApplicationID)
			if err != nil {
				return fmt.Errorf("failed to get next sequence: %w", err)
			}
			event.SequenceNumber = seq
		}
	}

	// Marshal data to JSON string
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `INSERT INTO events (id, created_at, application_id, sequence_number, type, creator_public_key, version, data)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.Exec(query,
		event.ID,
		event.CreatedAt,
		event.ApplicationID,
		event.SequenceNumber,
		string(event.Type),
		event.CreatorPublicKey,
		event.Version,
		string(dataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// GetByID retrieves a single event by ID
func (r *EventRepository) GetByID(id string) (*Event, error) {
	query := `SELECT id, created_at, application_id, sequence_number, type, creator_public_key, version, data
			  FROM events WHERE id = ?`

	event := &Event{}
	var eventType string
	var dataJSON string

	err := r.db.QueryRow(query, id).Scan(
		&event.ID,
		&event.CreatedAt,
		&event.ApplicationID,
		&event.SequenceNumber,
		&eventType,
		&event.CreatorPublicKey,
		&event.Version,
		&dataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query event: %w", err)
	}

	event.Type = EventType(eventType)

	// Unmarshal data JSON
	if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return event, nil
}

// GetSince retrieves events after a given event ID for applications the user is a member of
// Filters events by user's application memberships for security and correctness
func (r *EventRepository) GetSince(userPublicKey string, sinceEventID string, limit int) ([]*Event, bool, error) {
	if limit <= 0 || limit > 500 {
		limit = 100 // Default limit
	}

	var query string
	var args []interface{}

	if sinceEventID == "" {
		// Get latest events for user's applications
		query = `SELECT DISTINCT e.id, e.created_at, e.application_id, e.sequence_number,
				        e.type, e.creator_public_key, e.version, e.data
				 FROM events e
				 INNER JOIN members m ON e.application_id = m.application_id
				 WHERE m.public_key = ?
				 ORDER BY e.application_id ASC, e.sequence_number ASC, e.created_at ASC
				 LIMIT ?`
		args = []interface{}{userPublicKey, limit + 1}
	} else {
		// Get events after sinceEventID for user's applications
		// First, get the application_id, sequence_number and created_at of sinceEventID
		var sinceAppID string
		var sinceSequence int64
		var sinceCreatedAt int64
		err := r.db.QueryRow("SELECT application_id, sequence_number, created_at FROM events WHERE id = ?", sinceEventID).Scan(&sinceAppID, &sinceSequence, &sinceCreatedAt)
		if err == sql.ErrNoRows {
			// Event not found - might have been cleaned up, return all events for user
			return r.GetSince(userPublicKey, "", limit)
		}
		if err != nil {
			return nil, false, fmt.Errorf("failed to get since event: %w", err)
		}

		// Query events after the cursor for ALL user's applications
		// For the same application as sinceEventID, get events with higher sequence
		// For other applications, get all events
		query = `SELECT DISTINCT e.id, e.created_at, e.application_id, e.sequence_number,
				        e.type, e.creator_public_key, e.version, e.data
				 FROM events e
				 INNER JOIN members m ON e.application_id = m.application_id
				 WHERE m.public_key = ?
				   AND (
				     e.application_id != ?
				     OR (e.application_id = ? AND (
				       e.sequence_number > ?
				       OR (e.sequence_number = ? AND e.created_at > ?)
				       OR (e.sequence_number = ? AND e.created_at = ? AND e.id > ?)
				     ))
				   )
				 ORDER BY e.application_id ASC, e.sequence_number ASC, e.created_at ASC
				 LIMIT ?`
		args = []interface{}{
			userPublicKey,
			sinceAppID, sinceAppID, sinceSequence, sinceSequence, sinceCreatedAt, sinceSequence, sinceCreatedAt, sinceEventID,
			limit + 1,
		}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		var eventType string
		var dataJSON string

		err := rows.Scan(
			&event.ID,
			&event.CreatedAt,
			&event.ApplicationID,
			&event.SequenceNumber,
			&eventType,
			&event.CreatorPublicKey,
			&event.Version,
			&dataJSON,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to scan event: %w", err)
		}

		event.Type = EventType(eventType)

		// Unmarshal data JSON
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, false, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		events = append(events, event)

		// Stop if we've fetched limit + 1 (to check hasMore)
		if len(events) > limit {
			break
		}
	}

	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("error iterating events: %w", err)
	}

	// Check if there are more events
	hasMore := len(events) > limit
	if hasMore {
		// Remove the extra event
		events = events[:limit]
	}

	return events, hasMore, nil
}

// GetByApplicationID retrieves all events for a specific application
func (r *EventRepository) GetByApplicationID(appID string, limit int) ([]*Event, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, created_at, application_id, sequence_number, type, creator_public_key, version, data
			  FROM events
			  WHERE application_id = ?
			  ORDER BY sequence_number ASC, created_at ASC
			  LIMIT ?`

	rows, err := r.db.Query(query, appID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		var eventType string
		var dataJSON string

		err := rows.Scan(
			&event.ID,
			&event.CreatedAt,
			&event.ApplicationID,
			&event.SequenceNumber,
			&eventType,
			&event.CreatorPublicKey,
			&event.Version,
			&dataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		event.Type = EventType(eventType)

		// Unmarshal data JSON
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		events = append(events, event)
	}

	return events, rows.Err()
}

// DeleteOlderThan deletes events older than the specified timestamp
// Used by cleanup cronjob to enforce retention policy
func (r *EventRepository) DeleteOlderThan(timestamp int64) (int64, error) {
	query := `DELETE FROM events WHERE created_at < ?`

	result, err := r.db.Exec(query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetOldestEventID returns the ID of the oldest event in the database
func (r *EventRepository) GetOldestEventID() (string, error) {
	query := `SELECT id FROM events ORDER BY created_at ASC, id ASC LIMIT 1`

	var id string
	err := r.db.QueryRow(query).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil // No events
	}
	if err != nil {
		return "", fmt.Errorf("failed to get oldest event ID: %w", err)
	}

	return id, nil
}

// Count returns the total number of events in the database
func (r *EventRepository) Count() (int64, error) {
	query := `SELECT COUNT(*) FROM events`

	var count int64
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}

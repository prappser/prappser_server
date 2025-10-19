package event

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EventService struct {
	repo *EventRepository
}

func NewEventService(repo *EventRepository) *EventService {
	return &EventService{
		repo: repo,
	}
}

// ProduceMemberAdded creates a member_added event
func (s *EventService) ProduceMemberAdded(appID, creatorPublicKey, memberPublicKey, role, inviteID string) error {
	data := MemberAddedData{
		MemberPublicKey: memberPublicKey,
		Role:            role,
		InviteID:        inviteID,
	}

	dataMap, err := MarshalData(data)
	if err != nil {
		return fmt.Errorf("failed to marshal member_added data: %w", err)
	}

	event := NewEvent(
		uuid.New().String(), // TODO: Use UUID v7
		EventTypeMemberAdded,
		appID,
		creatorPublicKey,
		dataMap,
	)

	return s.repo.Create(event)
}

// ProduceMemberRemoved creates a member_removed event
func (s *EventService) ProduceMemberRemoved(appID, creatorPublicKey, memberPublicKey, reason string) error {
	data := MemberRemovedData{
		MemberPublicKey: memberPublicKey,
		Reason:          reason,
	}

	dataMap, err := MarshalData(data)
	if err != nil {
		return fmt.Errorf("failed to marshal member_removed data: %w", err)
	}

	event := NewEvent(
		uuid.New().String(), // TODO: Use UUID v7
		EventTypeMemberRemoved,
		appID,
		creatorPublicKey,
		dataMap,
	)

	return s.repo.Create(event)
}

// ProduceMemberRoleChanged creates a member_role_changed event
func (s *EventService) ProduceMemberRoleChanged(appID, creatorPublicKey, memberPublicKey, oldRole, newRole string) error {
	data := MemberRoleChangedData{
		MemberPublicKey: memberPublicKey,
		OldRole:         oldRole,
		NewRole:         newRole,
	}

	dataMap, err := MarshalData(data)
	if err != nil {
		return fmt.Errorf("failed to marshal member_role_changed data: %w", err)
	}

	event := NewEvent(
		uuid.New().String(), // TODO: Use UUID v7
		EventTypeMemberRoleChanged,
		appID,
		creatorPublicKey,
		dataMap,
	)

	return s.repo.Create(event)
}

// ProduceApplicationDataChanged creates an application_data_changed event
func (s *EventService) ProduceApplicationDataChanged(appID, creatorPublicKey string, version int, changedFields []string) error {
	data := ApplicationDataChangedData{
		Version:       version,
		ChangedFields: changedFields,
	}

	dataMap, err := MarshalData(data)
	if err != nil {
		return fmt.Errorf("failed to marshal application_data_changed data: %w", err)
	}

	event := NewEvent(
		uuid.New().String(), // TODO: Use UUID v7
		EventTypeApplicationDataChanged,
		appID,
		creatorPublicKey,
		dataMap,
	)

	return s.repo.Create(event)
}

// ProduceApplicationDeleted creates an application_deleted event
func (s *EventService) ProduceApplicationDeleted(appID, creatorPublicKey string) error {
	data := ApplicationDeletedData{
		DeletedAt: time.Now().Unix(),
	}

	dataMap, err := MarshalData(data)
	if err != nil {
		return fmt.Errorf("failed to marshal application_deleted data: %w", err)
	}

	event := NewEvent(
		uuid.New().String(), // TODO: Use UUID v7
		EventTypeApplicationDeleted,
		appID,
		creatorPublicKey,
		dataMap,
	)

	return s.repo.Create(event)
}

// ProduceInviteRevoked creates an invite_revoked event
func (s *EventService) ProduceInviteRevoked(appID, creatorPublicKey, inviteID string) error {
	data := InviteRevokedData{
		InviteID: inviteID,
	}

	dataMap, err := MarshalData(data)
	if err != nil {
		return fmt.Errorf("failed to marshal invite_revoked data: %w", err)
	}

	event := NewEvent(
		uuid.New().String(), // TODO: Use UUID v7
		EventTypeInviteRevoked,
		appID,
		creatorPublicKey,
		dataMap,
	)

	return s.repo.Create(event)
}

// GetEventsSince retrieves events for a user since a given event ID
func (s *EventService) GetEventsSince(userPublicKey string, sinceEventID string, limit int) (*EventsResponse, error) {
	events, hasMore, err := s.repo.GetSince(userPublicKey, sinceEventID, limit)
	if err != nil {
		// Check if the error is because sinceEventID was not found (might have been cleaned up)
		if err.Error() == "since event not found" {
			return &EventsResponse{
				FullResyncRequired: true,
				Reason:             "Events expired or gap detected",
			}, nil
		}
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	return &EventsResponse{
		Events:  events,
		HasMore: hasMore,
	}, nil
}

// CleanupOldEvents deletes events older than the retention period (7 days)
func (s *EventService) CleanupOldEvents(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 7 // Default 7 days
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays).Unix()
	return s.repo.DeleteOlderThan(cutoffTime)
}

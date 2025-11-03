package event

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prappser/prappser_server/internal/application"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/rs/zerolog/log"
)

type EventService struct {
	repo    *EventRepository
	appRepo application.ApplicationRepository
}

func NewEventService(repo *EventRepository, appRepo application.ApplicationRepository) *EventService {
	return &EventService{
		repo:    repo,
		appRepo: appRepo,
	}
}

func (s *EventService) AcceptEvent(ctx context.Context, event *Event, submitter *user.User) (*Event, error) {
	if err := ValidateEvent(event); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	appID, ok := event.Data["applicationId"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("%w: applicationId not found in event data", ErrValidation)
	}

	// Set ApplicationID field from data
	event.ApplicationID = appID

	app, err := s.appRepo.GetApplicationByID(appID)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	if err := AuthorizeEvent(event, submitter, app); err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	seq, err := s.repo.GetNextSequence(appID)
	if err != nil {
		return nil, fmt.Errorf("sequence generation failed: %w", err)
	}
	event.SequenceNumber = seq

	event.CreatedAt = time.Now().Unix()

	if err := s.repo.Create(event); err != nil {
		return nil, fmt.Errorf("persistence failed: %w", err)
	}

	// Execute event to update database state
	if err := s.executeEvent(ctx, event); err != nil {
		// Log error but don't fail event acceptance
		// Event is already persisted and sequenced
		log.Printf("[WARN] Failed to execute event %s (type: %s): %v", event.ID, event.Type, err)
	}

	return event, nil
}

// ProduceEvent creates an event without authorization checks.
// Used for server-generated events where the action was already validated by the endpoint.
// The event is sequenced, persisted, and executed but authorization is skipped.
func (s *EventService) ProduceEvent(ctx context.Context, event *Event) (*Event, error) {
	if err := ValidateEvent(event); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	appID, ok := event.Data["applicationId"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("%w: applicationId not found in event data", ErrValidation)
	}

	// Set ApplicationID field from data
	event.ApplicationID = appID

	seq, err := s.repo.GetNextSequence(appID)
	if err != nil {
		return nil, fmt.Errorf("sequence generation failed: %w", err)
	}
	event.SequenceNumber = seq

	event.CreatedAt = time.Now().Unix()

	if err := s.repo.Create(event); err != nil {
		return nil, fmt.Errorf("persistence failed: %w", err)
	}

	// Execute event to update database state
	if err := s.executeEvent(ctx, event); err != nil {
		// Log error but don't fail event acceptance
		// Event is already persisted and sequenced
		log.Error().
			Str("eventId", event.ID).
			Str("eventType", string(event.Type)).
			Str("applicationId", event.ApplicationID).
			Err(err).
			Msg("[EVENT_EXECUTION] Failed to execute server-produced event - event persisted but database state not updated")
	} else {
		log.Debug().
			Str("eventId", event.ID).
			Str("eventType", string(event.Type)).
			Str("applicationId", event.ApplicationID).
			Msg("[EVENT_EXECUTION] Successfully executed server-produced event")
	}

	return event, nil
}

// GetEventsSince retrieves events since a given event ID for the authenticated user's applications
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

// executeEvent executes an event by updating the database state
func (s *EventService) executeEvent(ctx context.Context, event *Event) error {
	switch event.Type {
	case "member_added":
		return s.executeMemberAdded(ctx, event)
	case "member_removed":
		return s.executeMemberRemoved(ctx, event)
	case "application_deleted":
		return s.executeApplicationDeleted(ctx, event)
	case "member_role_changed":
		return s.executeMemberRoleChanged(ctx, event)
	default:
		// Unknown event types are not executed (forward compatibility)
		return nil
	}
}

// executeMemberAdded creates a member record in the database
func (s *EventService) executeMemberAdded(ctx context.Context, event *Event) error {
	appID, ok := event.Data["applicationId"].(string)
	if !ok || appID == "" {
		return fmt.Errorf("missing applicationId in member_added event")
	}

	memberPublicKey, ok := event.Data["memberPublicKey"].(string)
	if !ok || memberPublicKey == "" {
		return fmt.Errorf("missing memberPublicKey in member_added event")
	}

	memberName, ok := event.Data["memberName"].(string)
	if !ok || memberName == "" {
		return fmt.Errorf("missing memberName in member_added event")
	}

	roleStr, ok := event.Data["role"].(string)
	if !ok || roleStr == "" {
		roleStr = "member" // Default role
	}

	member := &application.Member{
		ID:            uuid.New().String(),
		ApplicationID: appID,
		Name:          memberName,
		Role:          application.MemberRole(roleStr),
		PublicKey:     memberPublicKey,
	}

	return s.appRepo.CreateMember(member)
}

// executeMemberRemoved deletes a member record from the database
func (s *EventService) executeMemberRemoved(ctx context.Context, event *Event) error {
	appID, ok := event.Data["applicationId"].(string)
	if !ok || appID == "" {
		return fmt.Errorf("missing applicationId in member_removed event")
	}

	memberPublicKey, ok := event.Data["memberPublicKey"].(string)
	if !ok || memberPublicKey == "" {
		return fmt.Errorf("missing memberPublicKey in member_removed event")
	}

	log.Debug().
		Str("applicationId", appID).
		Str("memberPublicKey", memberPublicKey[:20]+"...").
		Msg("[MEMBER_REMOVED] Executing member_removed event - looking up member")

	// Get member by publicKey to get the member ID
	member, err := s.appRepo.GetMemberByPublicKey(appID, memberPublicKey)
	if err != nil {
		log.Error().
			Str("applicationId", appID).
			Str("memberPublicKey", memberPublicKey[:20]+"...").
			Err(err).
			Msg("[MEMBER_REMOVED] Failed to find member for deletion")
		return fmt.Errorf("member not found: %w", err)
	}

	log.Debug().
		Str("applicationId", appID).
		Str("memberId", member.ID).
		Str("memberPublicKey", memberPublicKey[:20]+"...").
		Msg("[MEMBER_REMOVED] Found member, deleting from database")

	// Delete member by ID
	if err := s.appRepo.DeleteMember(member.ID); err != nil {
		log.Error().
			Str("applicationId", appID).
			Str("memberId", member.ID).
			Err(err).
			Msg("[MEMBER_REMOVED] Failed to delete member from database")
		return err
	}

	log.Info().
		Str("applicationId", appID).
		Str("memberId", member.ID).
		Str("memberPublicKey", memberPublicKey[:20]+"...").
		Msg("[MEMBER_REMOVED] Successfully deleted member from database")

	return nil
}

// executeApplicationDeleted deletes an application (cascades to members, groups, components)
func (s *EventService) executeApplicationDeleted(ctx context.Context, event *Event) error {
	appID, ok := event.Data["applicationId"].(string)
	if !ok || appID == "" {
		return fmt.Errorf("missing applicationId in application_deleted event")
	}

	return s.appRepo.DeleteApplication(appID)
}

// executeMemberRoleChanged updates a member's role in the database
func (s *EventService) executeMemberRoleChanged(ctx context.Context, event *Event) error {
	appID, ok := event.Data["applicationId"].(string)
	if !ok || appID == "" {
		return fmt.Errorf("missing applicationId in member_role_changed event")
	}

	memberPublicKey, ok := event.Data["memberPublicKey"].(string)
	if !ok || memberPublicKey == "" {
		return fmt.Errorf("missing memberPublicKey in member_role_changed event")
	}

	newRole, ok := event.Data["newRole"].(string)
	if !ok || newRole == "" {
		return fmt.Errorf("missing newRole in member_role_changed event")
	}

	// Get member by publicKey
	member, err := s.appRepo.GetMemberByPublicKey(appID, memberPublicKey)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}

	// Update role
	member.Role = application.MemberRole(newRole)
	return s.appRepo.UpdateMember(member)
}

// CleanupOldEvents deletes events older than the retention period (7 days)
func (s *EventService) CleanupOldEvents(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 7 // Default 7 days
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays).Unix()
	return s.repo.DeleteOlderThan(cutoffTime)
}

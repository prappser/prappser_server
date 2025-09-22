package application

import (
	"fmt"
	"time"

	"github.com/prappser/prappser_server/internal/user"
)

type ApplicationService struct {
	appRepo ApplicationRepository
}

func NewApplicationService(appRepo ApplicationRepository) *ApplicationService {
	return &ApplicationService{
		appRepo: appRepo,
	}
}

func (s *ApplicationService) RegisterApplication(ownerPublicKey string, app *Application) (*Application, error) {
	// Validate application
	if app.ID == "" {
		return nil, fmt.Errorf("application ID cannot be empty")
	}
	if app.Name == "" {
		return nil, fmt.Errorf("application name cannot be empty")
	}
	if app.UserPublicKey == "" {
		return nil, fmt.Errorf("user public key cannot be empty")
	}
	if app.OwnerPublicKey == "" {
		return nil, fmt.Errorf("owner public key cannot be empty")
	}

	// Set timestamps
	now := time.Now().Unix()
	app.CreatedAt = now
	app.UpdatedAt = now

	// Create application
	if err := s.appRepo.CreateApplication(app); err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	// Create members
	for _, member := range app.Members {
		if member.ID == "" {
			return nil, fmt.Errorf("member ID cannot be empty")
		}
		member.ApplicationID = app.ID

		if err := s.appRepo.CreateMember(&member); err != nil {
			return nil, fmt.Errorf("failed to create member: %w", err)
		}
	}

	// Create component groups and components
	for _, group := range app.ComponentGroups {
		if group.ID == "" {
			return nil, fmt.Errorf("component group ID cannot be empty")
		}
		group.ApplicationID = app.ID

		if err := s.appRepo.CreateComponentGroup(&group); err != nil {
			return nil, fmt.Errorf("failed to create component group: %w", err)
		}

		// Create components for this group
		for _, component := range group.Components {
			if component.ID == "" {
				return nil, fmt.Errorf("component ID cannot be empty")
			}
			component.ComponentGroupID = group.ID
			component.ApplicationID = app.ID

			if err := s.appRepo.CreateComponent(&component); err != nil {
				return nil, fmt.Errorf("failed to create component: %w", err)
			}
		}
	}

	// Return the complete application
	return s.appRepo.GetApplicationByID(app.ID)
}

func (s *ApplicationService) GetApplication(appID string, requestingUser *user.User) (*Application, error) {
	app, err := s.appRepo.GetApplicationByID(appID)
	if err != nil {
		return nil, err
	}
	
	// Verify ownership
	if app.OwnerPublicKey != requestingUser.PublicKey {
		return nil, fmt.Errorf("unauthorized")
	}
	
	return app, nil
}

func (s *ApplicationService) GetApplicationState(appID string, requestingUser *user.User) (*ApplicationState, error) {
	state, err := s.appRepo.GetApplicationState(appID)
	if err != nil {
		return nil, err
	}
	
	// Verify ownership by fetching the full app (state doesn't include userID)
	app, err := s.appRepo.GetApplicationByID(appID)
	if err != nil {
		return nil, err
	}
	
	if app.OwnerPublicKey != requestingUser.PublicKey {
		return nil, fmt.Errorf("unauthorized")
	}
	
	return state, nil
}

func (s *ApplicationService) ListApplications(ownerPublicKey string) ([]*Application, error) {
	return s.appRepo.GetApplicationsByOwnerPublicKey(ownerPublicKey)
}

func (s *ApplicationService) DeleteApplication(appID string, requestingUser *user.User) error {
	// First verify the application exists and the user owns it
	app, err := s.appRepo.GetApplicationByID(appID)
	if err != nil {
		return err
	}
	
	// Verify ownership
	if app.OwnerPublicKey != requestingUser.PublicKey {
		return fmt.Errorf("unauthorized")
	}
	
	// Delete the application
	return s.appRepo.DeleteApplication(appID)
}


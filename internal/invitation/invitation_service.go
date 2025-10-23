package invitation

import (
	"crypto/rsa"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prappser/prappser_server/internal/application"
)

const (
	MaxExpirationHours = 48
)

// EventProducer interface for producing events (to avoid circular dependency)
type EventProducer interface {
	ProduceMemberAdded(appID, creatorPublicKey, memberPublicKey, role, inviteID string) error
}

type InvitationService struct {
	repo          InvitationRepository
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	appRepo       application.ApplicationRepository
	eventProducer EventProducer
	db            *sql.DB
	serverURL     string
}

func NewInvitationService(repo InvitationRepository, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, appRepo application.ApplicationRepository, eventProducer EventProducer, db *sql.DB, serverURL string) *InvitationService {
	return &InvitationService{
		repo:          repo,
		privateKey:    privateKey,
		publicKey:     publicKey,
		appRepo:       appRepo,
		eventProducer: eventProducer,
		db:            db,
		serverURL:     serverURL,
	}
}

// CreateInvitationOptions contains options for creating an invitation
type CreateInvitationOptions struct {
	ApplicationID      string
	CreatedByPublicKey string
	Role               string
	MaxUses            *int
	ExpiresInHours     *int
}

// CreateInvitation creates a new invitation and generates a JWT token
func (s *InvitationService) CreateInvitation(opts CreateInvitationOptions) (*InvitationResponse, error) {
	// Validate inputs
	if opts.ApplicationID == "" {
		return nil, fmt.Errorf("application ID is required")
	}
	if opts.CreatedByPublicKey == "" {
		return nil, fmt.Errorf("creator public key is required")
	}
	if opts.Role == "" {
		opts.Role = "member" // default
	}

	// Validate expiration
	if opts.ExpiresInHours != nil {
		if *opts.ExpiresInHours < 0 || *opts.ExpiresInHours > MaxExpirationHours {
			return nil, fmt.Errorf("expiration hours must be between 0 and %d", MaxExpirationHours)
		}
	}

	// Validate max uses
	if opts.MaxUses != nil && *opts.MaxUses < 1 {
		return nil, fmt.Errorf("max uses must be at least 1")
	}

	// Create invitation
	now := time.Now().Unix()
	invite := &Invitation{
		ID:                 uuid.New().String(), // TODO: Use UUID v7
		ApplicationID:      opts.ApplicationID,
		CreatedByPublicKey: opts.CreatedByPublicKey,
		Role:               opts.Role,
		MaxUses:            opts.MaxUses,
		UsedCount:          0,
		CreatedAt:          now,
	}

	// Save to database
	if err := s.repo.Create(invite); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Generate JWT token
	var expiresAt *int64
	if opts.ExpiresInHours != nil {
		exp := time.Now().Add(time.Duration(*opts.ExpiresInHours) * time.Hour).Unix()
		expiresAt = &exp
	}

	token, err := s.GenerateToken(invite.ID, invite.ApplicationID, invite.Role, s.serverURL, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Build response - use HTTPS PWA URL for sharing
	pwaURL := "https://prappser-app.netlify.app"
	response := &InvitationResponse{
		ID:        invite.ID,
		Token:     token,
		URL:       fmt.Sprintf("%s/join?token=%s", pwaURL, token),
		DeepLink:  fmt.Sprintf("prappser://join?token=%s", token),
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	return response, nil
}

// GenerateToken creates a signed JWT token for an invitation
func (s *InvitationService) GenerateToken(inviteID, appID, role, serverURL string, expiresAt *int64) (string, error) {
	now := time.Now()

	claims := InviteTokenClaims{
		InviteID:      inviteID,
		ApplicationID: appID,
		Role:          role,
		ServerURL:     serverURL,
		IssuedAt:      now.Unix(),
		ExpiresAt:     expiresAt,
	}

	// Build registered claims for JWT
	registeredClaims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	if expiresAt != nil {
		registeredClaims.ExpiresAt = jwt.NewNumericDate(time.Unix(*expiresAt, 0))
	}

	// Create token with custom claims
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"inviteId":  claims.InviteID,
		"appId":     claims.ApplicationID,
		"role":      claims.Role,
		"serverUrl": claims.ServerURL,
		"iat":       claims.IssuedAt,
		"exp":       claims.ExpiresAt,
		"nbf":       registeredClaims.NotBefore.Unix(),
	})

	// Sign token
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken verifies a JWT token and returns the claims
func (s *InvitationService) ValidateToken(tokenString string) (*InviteTokenClaims, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		inviteClaims := &InviteTokenClaims{
			InviteID:      claims["inviteId"].(string),
			ApplicationID: claims["appId"].(string),
			Role:          claims["role"].(string),
			IssuedAt:      int64(claims["iat"].(float64)),
		}

		// ServerURL is required (added in this version)
		if serverURL, ok := claims["serverUrl"].(string); ok {
			inviteClaims.ServerURL = serverURL
		}

		// ExpiresAt is optional
		if exp, ok := claims["exp"]; ok && exp != nil {
			expInt := int64(exp.(float64))
			inviteClaims.ExpiresAt = &expInt
		}

		return inviteClaims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// GetInviteInfo returns public information about an invitation (no auth required)
// This is used by the join screen to display invite details before authentication
func (s *InvitationService) GetInviteInfo(tokenString string) (*InviteInfo, error) {
	// Validate token
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check expiration from JWT
	isExpired := false
	if claims.ExpiresAt != nil {
		isExpired = time.Now().Unix() > *claims.ExpiresAt
	}

	// Get invitation from database
	invite, err := s.repo.GetByID(claims.InviteID)
	if err != nil {
		return &InviteInfo{
			InviteID:  claims.InviteID,
			IsExpired: isExpired,
			IsValid:   false,
		}, nil
	}

	// Check max uses
	isMaxUsesReached := false
	if invite.MaxUses != nil && invite.UsedCount >= *invite.MaxUses {
		isMaxUsesReached = true
	}

	// TODO: Get application and host details
	// For now, return minimal info
	info := &InviteInfo{
		InviteID:        invite.ID,
		ApplicationName: "Application", // TODO: Fetch from ApplicationRepository
		HostName:        "Host",        // TODO: Fetch from UserRepository
		MemberCount:     1,             // TODO: Fetch from MemberRepository
		Role:            invite.Role,
		ExpiresAt:       claims.ExpiresAt,
		IsExpired:       isExpired,
		IsValid:         !isExpired && !isMaxUsesReached,
	}

	return info, nil
}

// RevokeInvitation deletes an invitation (hard delete)
func (s *InvitationService) RevokeInvitation(inviteID string) error {
	return s.repo.Delete(inviteID)
}

// GetInvitesForApp returns all active invitations for an application
func (s *InvitationService) GetInvitesForApp(appID string) ([]*Invitation, error) {
	return s.repo.GetByApplicationID(appID)
}

// JoinResult contains the result of a successful join operation
type JoinResult struct {
	Application *application.Application `json:"application"`
	Member      *application.Member      `json:"member"`
	IsNewMember bool                     `json:"isNewMember"`
}

// Join handles the complete join flow with transaction
func (s *InvitationService) Join(tokenString, userPublicKey, userName string) (*JoinResult, error) {
	// Validate token
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check expiration
	if claims.ExpiresAt != nil && time.Now().Unix() > *claims.ExpiresAt {
		return nil, fmt.Errorf("invitation expired")
	}

	// Get invitation
	invite, err := s.repo.GetByID(claims.InviteID)
	if err != nil {
		return nil, fmt.Errorf("invitation not found or revoked: %w", err)
	}

	// Check max uses
	if invite.MaxUses != nil && invite.UsedCount >= *invite.MaxUses {
		return nil, fmt.Errorf("invitation has reached maximum uses")
	}

	// Check if user is already a member (idempotent)
	isMember, err := s.appRepo.IsMember(invite.ApplicationID, userPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check membership: %w", err)
	}

	if isMember {
		// User is already a member - return success with existing data
		app, err := s.appRepo.GetApplicationByID(invite.ApplicationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get application: %w", err)
		}

		member, err := s.appRepo.GetMemberByPublicKey(invite.ApplicationID, userPublicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get member: %w", err)
		}

		return &JoinResult{
			Application: app,
			Member:      member,
			IsNewMember: false,
		}, nil
	}

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create member record
	member := &application.Member{
		ID:            uuid.New().String(),
		ApplicationID: invite.ApplicationID,
		Name:          userName,
		Role:          application.MemberRole(invite.Role),
		PublicKey:     userPublicKey,
		AvatarBytes:   nil,
	}

	if err := s.appRepo.CreateMember(member); err != nil {
		return nil, fmt.Errorf("failed to create member: %w", err)
	}

	// Increment invitation used count
	if err := s.repo.IncrementUseCount(invite.ID); err != nil {
		return nil, fmt.Errorf("failed to increment use count: %w", err)
	}

	// Record usage in invitation_uses table
	useID := uuid.New().String()
	if err := s.repo.RecordUse(invite.ID, userPublicKey, useID); err != nil {
		return nil, fmt.Errorf("failed to record invitation use: %w", err)
	}

	// Produce member_added event
	if s.eventProducer != nil {
		if err := s.eventProducer.ProduceMemberAdded(
			invite.ApplicationID,
			userPublicKey,      // The joiner is the creator of this event
			userPublicKey,      // The member being added
			invite.Role,
			invite.ID,
		); err != nil {
			// Log error but don't fail the transaction
			// Event production is not critical for join success
			fmt.Printf("WARNING: Failed to produce member_added event: %v\n", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get full application details
	app, err := s.appRepo.GetApplicationByID(invite.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get application after join: %w", err)
	}

	return &JoinResult{
		Application: app,
		Member:      member,
		IsNewMember: true,
	}, nil
}

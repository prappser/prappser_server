package invitation

import (
	"github.com/goccy/go-json"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type InvitationEndpoints struct {
	invitationService *InvitationService
}

func NewInvitationEndpoints(invitationService *InvitationService) *InvitationEndpoints {
	return &InvitationEndpoints{
		invitationService: invitationService,
	}
}

// CreateInviteRequest represents the request body for creating an invitation
type CreateInviteRequest struct {
	ExpiresInHours *int   `json:"expiresInHours,omitempty"`
	Role           string `json:"role,omitempty"`
	MaxUses        *int   `json:"maxUses,omitempty"`
}

// CreateInvite handles POST /applications/{id}/invites
func (ie *InvitationEndpoints) CreateInvite(ctx *fasthttp.RequestCtx) {
	// Get authenticated user from context
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok {
		log.Error().Msg("Failed to get authenticated user from context")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	// Extract application ID from path
	appID := ctx.UserValue("appID").(string)
	if appID == "" {
		ctx.Error("Application ID is required", fasthttp.StatusBadRequest)
		return
	}

	// Parse request body
	var req CreateInviteRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		ctx.Error("Invalid request body", fasthttp.StatusBadRequest)
		return
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = "member"
	}

	// TODO: Verify user is owner of the application

	// Create invitation
	opts := CreateInvitationOptions{
		ApplicationID:      appID,
		CreatedByPublicKey: authenticatedUser.PublicKey,
		Role:               req.Role,
		MaxUses:            req.MaxUses,
		ExpiresInHours:     req.ExpiresInHours,
	}

	response, err := ie.invitationService.CreateInvitation(opts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create invitation")
		ctx.Error("Failed to create invitation", fasthttp.StatusInternalServerError)
		return
	}

	// Return created invitation
	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(response)
}

// GetInviteInfo handles GET /invites/{token}/info
// This is a public endpoint (no auth required)
func (ie *InvitationEndpoints) GetInviteInfo(ctx *fasthttp.RequestCtx) {
	// Extract token from path
	token := ctx.UserValue("token").(string)
	if token == "" {
		ctx.Error("Token is required", fasthttp.StatusBadRequest)
		return
	}

	// Get invite info
	info, err := ie.invitationService.GetInviteInfo(token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get invite info")
		ctx.Error("Invalid or expired invite", fasthttp.StatusBadRequest)
		return
	}

	// Return invite info
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(info)
}

// JoinRequest represents the request body for joining via invitation
type JoinRequest struct {
	Token string `json:"token"`
}

// JoinApplication handles POST /invites/{token}/join
func (ie *InvitationEndpoints) JoinApplication(ctx *fasthttp.RequestCtx) {
	// Get authenticated user from context
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok {
		log.Error().Msg("Failed to get authenticated user from context")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	// Extract token from path
	token := ctx.UserValue("token").(string)
	if token == "" {
		ctx.Error("Token is required", fasthttp.StatusBadRequest)
		return
	}

	// Join via invitation service (handles validation, transaction, event production)
	result, err := ie.invitationService.Join(token, authenticatedUser.PublicKey, authenticatedUser.Username)
	if err != nil {
		log.Error().Err(err).Msg("Failed to join application")

		// Determine appropriate status code based on error message
		errorMsg := err.Error()
		switch {
		case errorMsg == "invalid token: failed to parse token: token is expired":
			ctx.Error("Invitation expired", fasthttp.StatusGone)
		case errorMsg == "invitation expired":
			ctx.Error("Invitation expired", fasthttp.StatusGone)
		case errorMsg == "invitation has reached maximum uses":
			ctx.Error("Invitation has reached maximum uses", fasthttp.StatusGone)
		case errorMsg == "invitation not found or revoked: invitation not found":
			ctx.Error("Invitation not found or revoked", fasthttp.StatusNotFound)
		default:
			ctx.Error("Failed to join application", fasthttp.StatusInternalServerError)
		}
		return
	}

	// Return success response
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(result)
}

// RevokeInvite handles DELETE /applications/{appID}/invites/{inviteID}
func (ie *InvitationEndpoints) RevokeInvite(ctx *fasthttp.RequestCtx) {
	// Get authenticated user from context
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok {
		log.Error().Msg("Failed to get authenticated user from context")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	// Extract IDs from path
	appID := ctx.UserValue("appID").(string)
	inviteID := ctx.UserValue("inviteID").(string)

	if appID == "" || inviteID == "" {
		ctx.Error("Application ID and Invite ID are required", fasthttp.StatusBadRequest)
		return
	}

	// TODO: Verify user is owner of the application using authenticatedUser.PublicKey
	_ = authenticatedUser // Will be used in owner verification TODO

	// Get invitation to verify it exists
	invite, err := ie.invitationService.repo.GetByID(inviteID)
	if err != nil {
		log.Error().Err(err).Msg("Invitation not found")
		ctx.Error("Invitation not found", fasthttp.StatusNotFound)
		return
	}

	// Verify invite belongs to the application
	if invite.ApplicationID != appID {
		ctx.Error("Invitation does not belong to this application", fasthttp.StatusBadRequest)
		return
	}

	// Revoke invitation (hard delete)
	if err := ie.invitationService.RevokeInvitation(inviteID); err != nil {
		log.Error().Err(err).Msg("Failed to revoke invitation")
		ctx.Error("Failed to revoke invitation", fasthttp.StatusInternalServerError)
		return
	}

	// TODO: Produce invite_revoked event

	// Return success (204 No Content)
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// ListInvites handles GET /applications/{appID}/invites
func (ie *InvitationEndpoints) ListInvites(ctx *fasthttp.RequestCtx) {
	// Get authenticated user from context
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok {
		log.Error().Msg("Failed to get authenticated user from context")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	// Extract application ID from path
	appID := ctx.UserValue("appID").(string)
	if appID == "" {
		ctx.Error("Application ID is required", fasthttp.StatusBadRequest)
		return
	}

	// TODO: Verify user is owner or admin of the application

	_ = authenticatedUser // Will be used in TODO above

	// Get invites for application
	invites, err := ie.invitationService.GetInvitesForApp(appID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get invites")
		ctx.Error("Failed to get invites", fasthttp.StatusInternalServerError)
		return
	}

	// Return invites
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(invites)
}

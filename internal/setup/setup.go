package setup

import (
	"database/sql"
	"fmt"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type SetupEndpoints struct {
	db *sql.DB
}

func NewSetupEndpoints(db *sql.DB) *SetupEndpoints {
	return &SetupEndpoints{
		db: db,
	}
}

// HasOwner checks if an owner has been registered
func (s *SetupEndpoints) HasOwner() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'owner'").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check owner: %w", err)
	}
	return count > 0, nil
}

// SetRailwayToken stores the Railway API token for server self-management
// This endpoint requires owner authentication
func (s *SetupEndpoints) SetRailwayToken(ctx *fasthttp.RequestCtx) {
	// Parse request body
	var req struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		log.Error().Err(err).Msg("Failed to parse railway token request")
		ctx.Error("Invalid request body", fasthttp.StatusBadRequest)
		return
	}

	if req.Token == "" {
		ctx.Error("Token is required", fasthttp.StatusBadRequest)
		return
	}

	// Store the token in the setup_config table
	_, err := s.db.Exec(`
		INSERT INTO setup_config (id, railway_token) VALUES ('default', $1)
		ON CONFLICT(id) DO UPDATE SET railway_token = excluded.railway_token
	`, req.Token)

	if err != nil {
		log.Error().Err(err).Msg("Failed to store railway token")
		ctx.Error("Failed to store token", fasthttp.StatusInternalServerError)
		return
	}

	log.Info().Msg("Railway token stored successfully")

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(map[string]string{
		"message": "Railway token stored successfully",
	})
}

// GetRailwayToken retrieves the stored Railway token (for internal use)
func (s *SetupEndpoints) GetRailwayToken() (string, error) {
	var token sql.NullString
	err := s.db.QueryRow("SELECT railway_token FROM setup_config WHERE id = 'default'").Scan(&token)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to get railway token: %w", err)
	}
	if !token.Valid {
		return "", nil
	}
	return token.String, nil
}

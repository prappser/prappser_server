package user

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/goccy/go-json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/prappser/prappser_server/internal/user/owner"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

const (
	headerAuthorization = "Authorization"
	headerBearer        = "Bearer"

	RoleOwner = "owner"
)

type User struct {
	PublicKey string `json:"publicKey"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"createdAt"`
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByPublicKey(publicKey string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	UpdateUserRole(publicKey string, role string) error
}

type UserEndpoints struct {
	userRepository UserRepository
	config         Config
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
	userService    *UserService
	// Add challenge storage for verification
	challenges map[string]challengeInfo
}

type Config struct {
	MasterPasswordMD5Hash   string
	RegistrationTokenTTLSec int32
	JWTExpirationHours      int
	ChallengeTTLSec         int
}

// JWS claims for user authentication
type userAuthJWSClaims struct {
	PublicKey string `json:"publicKey"` // User's public key (unique identifier)
	Challenge string `json:"challenge"` // Prevents replay attacks
	IssuedAt  int64  `json:"iat"`       // For TTL validation
}

type JWTClaims struct {
	UserPublicKey string `json:"userPublicKey"`
	Username      string `json:"username"`
	Role          string `json:"role"`
	jwt.RegisteredClaims
}

type ChallengeResponse struct {
	Challenge       string `json:"challenge"`
	ExpiresAt       int64  `json:"expiresAt"`
	ServerPublicKey string `json:"serverPublicKey"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
}

type challengeInfo struct {
	challenge string
	expiresAt time.Time
}

var timeNowFunc = time.Now

func NewEndpoints(userRepository UserRepository, config Config, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, userService *UserService) *UserEndpoints {
	return &UserEndpoints{
		userRepository: userRepository,
		config:         config,
		privateKey:     privateKey,
		publicKey:      publicKey,
		userService:    userService,
		challenges:     make(map[string]challengeInfo),
	}
}

// OwnerRegister handles owner registration with JWE/JWS (existing flow)
func (ue UserEndpoints) OwnerRegister(ctx *fasthttp.RequestCtx) {
	authHeader := ctx.Request.Header.Peek(headerAuthorization)
	if authHeader == nil {
		log.Error().Msg("Missing authorization header")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	jwe, err := owner.ExtractJWEFromAuthorizationHeader(string(authHeader))
	if err != nil {
		log.Error().Err(err).Msg("Invalid authorization header")
		ctx.Error("Invalid authorization header", fasthttp.StatusBadRequest)
		return
	}

	registerJWEClaims, err := owner.DecryptJWE(jwe, ue.config.MasterPasswordMD5Hash)
	if err != nil {
		log.Error().Err(err).Msg("Failed to decrypt JWE")
		ctx.Error("Failed to decrypt JWE", fasthttp.StatusBadRequest)
		return
	}

	registerJWSClaims, err := owner.VerifyJWS(registerJWEClaims.JWS, ue.config.RegistrationTokenTTLSec)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify JWS")
		ctx.Error("Failed to verify JWS", fasthttp.StatusBadRequest)
		return
	}

	// Validate that public key is not empty
	if registerJWSClaims.PublicKey == "" {
		log.Error().Msg("Public key is empty")
		ctx.Error("Public key cannot be empty", fasthttp.StatusBadRequest)
		return
	}

	// Check if user already exists
	existingUser, err := ue.userRepository.GetUserByPublicKey(registerJWSClaims.PublicKey)
	if err == nil && existingUser != nil {
		// If user exists but is not an owner, upgrade them to owner
		if existingUser.Role != RoleOwner {
			log.Debug().
				Str("publicKey", existingUser.PublicKey).
				Str("oldRole", existingUser.Role).
				Msg("Upgrading user to owner role")

			err := ue.userRepository.UpdateUserRole(existingUser.PublicKey, RoleOwner)
			if err != nil {
				log.Error().Err(err).Msg("Failed to upgrade user to owner")
				ctx.Error("Failed to upgrade user to owner", fasthttp.StatusInternalServerError)
				return
			}
		}

		ctx.SetStatusCode(fasthttp.StatusCreated)
		ctx.SetContentType("application/json")
		response := map[string]string{"message": "Owner registered successfully", "publicKey": existingUser.PublicKey}
		json.NewEncoder(ctx).Encode(response)
		return
	}

	newUser := &User{
		PublicKey: registerJWSClaims.PublicKey,
		Username:  registerJWSClaims.Username,
		Role:      RoleOwner,
		CreatedAt: time.Now().Unix(),
	}

	err = ue.userRepository.CreateUser(newUser)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create owner")
		ctx.Error("Failed to create owner", fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetContentType("application/json")
	response := map[string]string{"message": "Owner registered successfully", "publicKey": registerJWSClaims.PublicKey}
	json.NewEncoder(ctx).Encode(response)
}

// GetChallenge generates a challenge for user login
func (ue UserEndpoints) GetChallenge(ctx *fasthttp.RequestCtx) {
	publicKey := ctx.QueryArgs().Peek("publicKey")
	if publicKey == nil {
		log.Error().Msg("[CHALLENGE] Missing publicKey parameter")
		ctx.Error("Missing publicKey parameter", fasthttp.StatusBadRequest)
		return
	}

	publicKeyStr := string(publicKey)
	log.Debug().Str("publicKey", publicKeyStr[:min(50, len(publicKeyStr))]+"...").Msg("[CHALLENGE] Challenge requested for user")

	// Check if user exists
	user, err := ue.userRepository.GetUserByPublicKey(publicKeyStr)
	if err != nil {
		log.Error().Err(err).Str("publicKey", publicKeyStr[:min(50, len(publicKeyStr))]+"...").Msg("[CHALLENGE] User not found")
		ctx.Error("User not found", fasthttp.StatusNotFound)
		return
	}

	log.Debug().Str("username", user.Username).Str("publicKey", publicKeyStr[:min(50, len(publicKeyStr))]+"...").Msg("[CHALLENGE] User found, generating challenge")

	// Generate random challenge
	challenge, err := generateChallenge()
	if err != nil {
		log.Error().Err(err).Msg("[CHALLENGE] Failed to generate challenge")
		ctx.Error("Internal server error", fasthttp.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(time.Duration(ue.config.ChallengeTTLSec) * time.Second)

	// Store challenge for verification (keyed by publicKey)
	ue.challenges[publicKeyStr] = challengeInfo{
		challenge: challenge,
		expiresAt: expiresAt,
	}

	log.Debug().Str("publicKey", publicKeyStr[:min(50, len(publicKeyStr))]+"...").Time("expiresAt", expiresAt).Msg("[CHALLENGE] Challenge generated and stored")

	// Convert server's public key to PEM format
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(ue.publicKey),
	}
	serverPublicKeyString := string(pem.EncodeToMemory(publicKeyPEM))

	response := ChallengeResponse{
		Challenge:       challenge,
		ExpiresAt:       expiresAt.Unix(),
		ServerPublicKey: serverPublicKeyString,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(response)
}

// UserAuth handles user authentication with challenge verification
func (ue UserEndpoints) UserAuth(ctx *fasthttp.RequestCtx) {
	log.Debug().Msg("[AUTH] Starting user authentication")

	authHeader := ctx.Request.Header.Peek(headerAuthorization)
	if authHeader == nil {
		log.Error().Msg("[AUTH] Missing authorization header")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	jws, err := extractJWSFromAuthorizationHeader(string(authHeader))
	if err != nil {
		log.Error().Err(err).Msg("[AUTH] Invalid authorization header")
		ctx.Error("Invalid authorization header", fasthttp.StatusBadRequest)
		return
	}

	log.Debug().Msg("[AUTH] Verifying JWS signature")
	claims, err := ue.verifyUserAuthJWS(jws, ue.config.ChallengeTTLSec)
	if err != nil {
		log.Error().Err(err).Msg("[AUTH] Failed to verify JWS")
		ctx.Error("Failed to verify JWS", fasthttp.StatusBadRequest)
		return
	}

	publicKeyPrefix := claims.PublicKey[:min(50, len(claims.PublicKey))] + "..."
	log.Debug().Str("publicKey", publicKeyPrefix).Msg("[AUTH] JWS verified, fetching user")

	// Get user by public key (already verified in verifyUserAuthJWS, but need full user object)
	user, err := ue.userRepository.GetUserByPublicKey(claims.PublicKey)
	if err != nil {
		log.Error().Err(err).Str("publicKey", publicKeyPrefix).Msg("[AUTH] User not found")
		ctx.Error("User not found", fasthttp.StatusNotFound)
		return
	}

	log.Debug().Str("username", user.Username).Str("role", user.Role).Msg("[AUTH] User found, generating JWT")

	// Generate JWT token
	token, expiresAt, err := ue.userService.GenerateJWT(user)
	if err != nil {
		log.Error().Err(err).Msg("[AUTH] Failed to generate JWT")
		ctx.Error("Internal server error", fasthttp.StatusInternalServerError)
		return
	}

	// Clean up used challenge (keyed by publicKey)
	delete(ue.challenges, claims.PublicKey)

	log.Debug().Str("username", user.Username).Msg("[AUTH] Authentication successful")

	response := LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(response)
}

// VerifyJWT middleware for protecting routes
func (ue UserEndpoints) VerifyJWT(ctx *fasthttp.RequestCtx) (*User, error) {
	return ue.userService.ValidateJWTFromRequest(ctx)
}

// Helper functions
func extractJWSFromAuthorizationHeader(authHeader string) (string, error) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid Authorization header format")
	}
	return parts[1], nil
}


func (ue UserEndpoints) verifyUserAuthJWS(signedJWS string, ttlSec int) (*userAuthJWSClaims, error) {
	log.Debug().Msg("[VERIFY] Parsing JWS")
	// Parse the JWS without verification to extract the claims
	msg, err := jws.Parse([]byte(signedJWS))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWS: %w", err)
	}

	claimsBytes := msg.Payload()
	var claims userAuthJWSClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWS claims: %w", err)
	}

	publicKeyPrefix := claims.PublicKey[:min(50, len(claims.PublicKey))] + "..."
	log.Debug().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] JWS claims extracted")

	// Check if the JWS is expired
	var timeNow = timeNowFunc()
	var issuedAtTime = time.Unix(claims.IssuedAt, 0)
	if issuedAtTime.Add(time.Duration(ttlSec) * time.Second).Before(timeNow) {
		log.Debug().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] JWS has expired")
		return nil, fmt.Errorf("JWS has expired")
	}

	log.Debug().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] Looking up user in database")
	// 1. Get the user by public key (unique identifier) - fixed security issue
	user, err := ue.userRepository.GetUserByPublicKey(claims.PublicKey)
	if err != nil {
		log.Debug().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] User not found in database")
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 2. Validate that the user has a public key
	if user.PublicKey == "" {
		log.Error().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] User has no public key registered")
		return nil, fmt.Errorf("user has no public key registered")
	}

	log.Debug().Str("username", user.Username).Str("publicKey", publicKeyPrefix).Int("publicKeyLen", len(user.PublicKey)).Msg("[VERIFY] User found, validating public key")

	// 3. Parse the user's public key
	block, _ := pem.Decode([]byte(user.PublicKey))
	if block == nil {
		log.Error().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] Failed to decode public key PEM")
		return nil, fmt.Errorf("failed to decode public key PEM")
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		log.Error().Err(err).Str("publicKey", publicKeyPrefix).Msg("[VERIFY] Failed to parse public key")
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	log.Debug().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] Public key parsed, verifying JWS signature")

	// 4. Verify the JWS signature using their public key
	verified, err := jws.Verify([]byte(signedJWS), jws.WithKey(jwa.RS256(), publicKey))
	if err != nil {
		log.Error().Err(err).Str("publicKey", publicKeyPrefix).Msg("[VERIFY] JWS signature verification failed")
		return nil, fmt.Errorf("failed to verify JWS signature: %w", err)
	}

	if len(verified) == 0 {
		log.Error().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] JWS signature verification returned empty result")
		return nil, fmt.Errorf("JWS signature verification failed")
	}

	log.Debug().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] JWS signature verified, checking challenge")

	// 5. Verify that the challenge matches what was issued (keyed by publicKey)
	storedChallenge, exists := ue.challenges[claims.PublicKey]
	if !exists {
		log.Error().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] No challenge found for user")
		return nil, fmt.Errorf("no challenge found for user")
	}

	if storedChallenge.challenge != claims.Challenge {
		log.Error().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] Challenge mismatch")
		return nil, fmt.Errorf("challenge mismatch")
	}

	// Check if challenge has expired
	if storedChallenge.expiresAt.Before(timeNow) {
		log.Error().Str("publicKey", publicKeyPrefix).Msg("[VERIFY] Challenge has expired")
		delete(ue.challenges, claims.PublicKey)
		return nil, fmt.Errorf("challenge has expired")
	}

	log.Debug().Str("username", user.Username).Str("publicKey", publicKeyPrefix).Msg("[VERIFY] All verifications passed successfully")
	return &claims, nil
}

func generateChallenge() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}


// GetServerPublicKey returns the server's public key for JWT verification
func (ue UserEndpoints) GetServerPublicKey(ctx *fasthttp.RequestCtx) {
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(ue.publicKey),
	}

	response := map[string]string{
		"publicKey": string(pem.EncodeToMemory(publicKeyPEM)),
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(response)
}

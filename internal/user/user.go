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
	MasterPasswordMD5Hash   string `mapstructure:"master_password_md5_hash"`
	RegistrationTokenTTLSec int32  `mapstructure:"registration_token_ttl_sec"`
	JWTExpirationHours      int    `mapstructure:"jwt_expiration_hours"`
	ChallengeTTLSec         int    `mapstructure:"challenge_ttl_sec"`
}

// JWS claims for user authentication
type userAuthJWSClaims struct {
	Username  string `json:"username"`
	Challenge string `json:"challenge"`
	IssuedAt  int64  `json:"iat"`
}

type JWTClaims struct {
	UserPublicKey string `json:"userPublicKey"`
	Username      string `json:"username"`
	Role          string `json:"role"`
	jwt.RegisteredClaims
}

type ChallengeResponse struct {
	Challenge string `json:"challenge"`
	ExpiresAt int64  `json:"expiresAt"`
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

	// Check if owner already exists
	existingUser, err := ue.userRepository.GetUserByPublicKey(registerJWSClaims.PublicKey)
	if err == nil && existingUser != nil {
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
	username := ctx.QueryArgs().Peek("username")
	if username == nil {
		log.Error().Msg("Missing username parameter")
		ctx.Error("Missing username parameter", fasthttp.StatusBadRequest)
		return
	}

	// Check if user exists
	_, err := ue.userRepository.GetUserByUsername(string(username))
	if err != nil {
		log.Error().Err(err).Msg("User not found")
		ctx.Error("User not found", fasthttp.StatusNotFound)
		return
	}

	// Generate random challenge
	challenge, err := generateChallenge()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate challenge")
		ctx.Error("Internal server error", fasthttp.StatusInternalServerError)
		return
	}

	expiresAt := time.Now().Add(time.Duration(ue.config.ChallengeTTLSec) * time.Second)

	// Store challenge for verification
	ue.challenges[string(username)] = challengeInfo{
		challenge: challenge,
		expiresAt: expiresAt,
	}

	response := ChallengeResponse{
		Challenge: challenge,
		ExpiresAt: expiresAt.Unix(),
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(response)
}

// UserAuth handles user authentication with challenge verification
func (ue UserEndpoints) UserAuth(ctx *fasthttp.RequestCtx) {
	authHeader := ctx.Request.Header.Peek(headerAuthorization)
	if authHeader == nil {
		log.Error().Msg("Missing authorization header")
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}

	jws, err := extractJWSFromAuthorizationHeader(string(authHeader))
	if err != nil {
		log.Error().Err(err).Msg("Invalid authorization header")
		ctx.Error("Invalid authorization header", fasthttp.StatusBadRequest)
		return
	}

	claims, err := ue.verifyUserAuthJWS(jws, ue.config.ChallengeTTLSec)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify JWS")
		ctx.Error("Failed to verify JWS", fasthttp.StatusBadRequest)
		return
	}

	// Get user by username
	user, err := ue.userRepository.GetUserByUsername(claims.Username)
	if err != nil {
		log.Error().Err(err).Msg("User not found")
		ctx.Error("User not found", fasthttp.StatusNotFound)
		return
	}

	// Generate JWT token
	token, expiresAt, err := ue.userService.GenerateJWT(user)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate JWT")
		ctx.Error("Internal server error", fasthttp.StatusInternalServerError)
		return
	}

	// Clean up used challenge
	delete(ue.challenges, claims.Username)

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

	// Check if the JWS is expired
	var timeNow = timeNowFunc()
	var issuedAtTime = time.Unix(claims.IssuedAt, 0)
	if issuedAtTime.Add(time.Duration(ttlSec) * time.Second).Before(timeNow) {
		return nil, fmt.Errorf("JWS has expired")
	}

	// 1. Get the user by username to retrieve their public key
	user, err := ue.userRepository.GetUserByUsername(claims.Username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 2. Parse the user's public key
	block, _ := pem.Decode([]byte(user.PublicKey))
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	// 3. Verify the JWS signature using their public key
	verified, err := jws.Verify([]byte(signedJWS), jws.WithKey(jwa.RS256(), publicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to verify JWS signature: %w", err)
	}

	if len(verified) == 0 {
		return nil, fmt.Errorf("JWS signature verification failed")
	}

	// 4. Verify that the challenge matches what was issued
	storedChallenge, exists := ue.challenges[claims.Username]
	if !exists {
		return nil, fmt.Errorf("no challenge found for user")
	}

	if storedChallenge.challenge != claims.Challenge {
		return nil, fmt.Errorf("challenge mismatch")
	}

	// Check if challenge has expired
	if storedChallenge.expiresAt.Before(timeNow) {
		delete(ue.challenges, claims.Username)
		return nil, fmt.Errorf("challenge has expired")
	}

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

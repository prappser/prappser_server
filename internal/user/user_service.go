package user

import (
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
)

type UserService struct {
	userRepository UserRepository
	config         Config
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
}

func NewUserService(userRepository UserRepository, config Config, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) *UserService {
	return &UserService{
		userRepository: userRepository,
		config:         config,
		privateKey:     privateKey,
		publicKey:      publicKey,
	}
}

func (us *UserService) ValidateJWTFromRequest(ctx *fasthttp.RequestCtx) (*User, error) {
	authHeader := ctx.Request.Header.Peek(headerAuthorization)
	if authHeader == nil {
		return nil, fmt.Errorf("missing authorization header")
	}

	tokenString, err := extractJWTFromAuthorizationHeader(string(authHeader))
	if err != nil {
		return nil, fmt.Errorf("invalid authorization header: %w", err)
	}

	return us.ValidateJWT(tokenString)
}

func (us *UserService) GenerateJWT(user *User) (string, int64, error) {
	expiresAt := time.Now().Add(time.Duration(us.config.JWTExpirationHours) * time.Hour).Unix()

	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(expiresAt, 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(us.privateKey)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt, nil
}

func (us *UserService) ValidateJWT(tokenString string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return us.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		user, err := us.userRepository.GetUserByID(claims.UserID)
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func extractJWTFromAuthorizationHeader(authHeader string) (string, error) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != headerBearer {
		return "", fmt.Errorf("invalid Authorization header format")
	}
	return parts[1], nil
}


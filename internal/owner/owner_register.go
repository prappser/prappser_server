package owner

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwe"
	"github.com/lestrrat-go/jwx/v3/jws"
)

func decryptJWE(encryptedJWE string, masterPasswordMD5Hash string) (*registerJWEClaims, error) {
	// Decrypt the JWE
	decodedKey, err := hex.DecodeString(masterPasswordMD5Hash)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}
	decrypted, err := jwe.Decrypt([]byte(encryptedJWE), jwe.WithKey(jwa.DIRECT(), decodedKey))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt JWE: %v", err)
	}
	var registerJWEClaims registerJWEClaims
	if err := json.Unmarshal(decrypted, &registerJWEClaims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWE claims: %w", err)
	}
	return &registerJWEClaims, nil
}

type registerJWEClaims struct {
	JWS string `json:"jws"`
}

type registerJWSClaims struct {
	PublicKey string `json:"publicKey"`
	IssuedAt  int64  `json:"iat"`
}

var timeNowFunc = time.Now

func extractJWEFromAuthorizationHeader(authHeader string) (string, error) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid Authorization header format")
	}
	return parts[1], nil
}

// VerifyJWS verifies the JWS using the public key from its claims
func verifyJWS(signedJWS string, registrationTokenTTLSec int32) (*registerJWSClaims, error) {
	// Parse the JWS without verification to extract the claims
	msg, err := jws.Parse([]byte(signedJWS))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWS: %w", err)
	}

	// Extract the public key from the claims
	claimsBytes := msg.Payload()

	var registerJWSClaims registerJWSClaims
	if err := json.Unmarshal(claimsBytes, &registerJWSClaims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWS claims: %w", err)
	}

	// Check if the JWS is expired
	var timeNow = timeNowFunc()
	var issuedAtTime = time.Unix(registerJWSClaims.IssuedAt, 0)
	if issuedAtTime.Add(time.Duration(registrationTokenTTLSec) * time.Second).Before(timeNow) {
		return nil, fmt.Errorf("JWS has expired")
	}

	// Decode the public key
	block, _ := pem.Decode([]byte(registerJWSClaims.PublicKey))
	if block == nil {
		return nil, fmt.Errorf("invalid public key format")
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	// Verify the JWS signature
	_, err = jws.Verify([]byte(signedJWS), jws.WithKey(jwa.RS256(), publicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to verify JWS: %w", err)
	}

	return &registerJWSClaims, nil
}

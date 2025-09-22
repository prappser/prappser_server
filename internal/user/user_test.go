package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateChallenge_ShouldGenerateUniqueChallenge(t *testing.T) {
	// when
	challenge1, err1 := generateChallenge()
	challenge2, err2 := generateChallenge()

	// then
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEmpty(t, challenge1)
	assert.NotEmpty(t, challenge2)
	assert.NotEqual(t, challenge1, challenge2)
}

func TestExtractJWSFromAuthorizationHeader_ShouldExtractValidly(t *testing.T) {
	// given
	authHeader := "Bearer eyJhbGciOiJSUzI1NiJ9.eyJ1c2VybmFtZSI6ImFsaWNlIiwiY2hhbGxlbmdlIjoiYTF..."

	// when
	jws, err := extractJWSFromAuthorizationHeader(authHeader)

	// then
	assert.NoError(t, err)
	assert.NotEmpty(t, jws)
	assert.Contains(t, jws, "eyJhbGciOiJSUzI1NiJ9")
}

func TestExtractJWSFromAuthorizationHeader_ShouldFailWithInvalidFormat(t *testing.T) {
	// given
	authHeader := "InvalidFormat"

	// when
	jws, err := extractJWSFromAuthorizationHeader(authHeader)

	// then
	assert.Error(t, err)
	assert.Empty(t, jws)
}

func TestExtractJWTFromAuthorizationHeader_ShouldExtractValidly(t *testing.T) {
	// given
	authHeader := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

	// when
	jwt, err := extractJWTFromAuthorizationHeader(authHeader)

	// then
	assert.NoError(t, err)
	assert.NotEmpty(t, jwt)
	assert.Contains(t, jwt, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
}

func TestExtractJWTFromAuthorizationHeader_ShouldFailWithInvalidFormat(t *testing.T) {
	// given
	authHeader := "InvalidFormat"

	// when
	jwt, err := extractJWTFromAuthorizationHeader(authHeader)

	// then
	assert.Error(t, err)
	assert.Empty(t, jwt)
}

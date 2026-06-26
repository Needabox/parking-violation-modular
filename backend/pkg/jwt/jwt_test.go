package jwtutil_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jwtutil "github.com/parking-violation-portal/backend/pkg/jwt"
)

func TestJWTManager_GenerateAndParse(t *testing.T) {
	manager := jwtutil.NewManager("test-secret", time.Hour)
	userID := uuid.New()

	token, expiresIn, err := manager.Generate(userID, "member@example.com", "member")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, int64(3600), expiresIn)

	claims, err := manager.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, "member@example.com", claims.Email)
	assert.Equal(t, "member", claims.Role)

	parsedID, err := jwtutil.SubjectUUID(claims)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedID)
}

func TestJWTManager_InvalidToken(t *testing.T) {
	manager := jwtutil.NewManager("test-secret", time.Hour)
	_, err := manager.Parse("invalid.token.value")
	assert.Error(t, err)
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	manager := jwtutil.NewManager("test-secret", -time.Hour)
	userID := uuid.New()

	token, _, err := manager.Generate(userID, "member@example.com", "member")
	require.NoError(t, err)

	_, err = manager.Parse(token)
	assert.Error(t, err)
}

func TestJWTManager_WrongSecret(t *testing.T) {
	managerA := jwtutil.NewManager("secret-a", time.Hour)
	managerB := jwtutil.NewManager("secret-b", time.Hour)
	userID := uuid.New()

	token, _, err := managerA.Generate(userID, "member@example.com", "member")
	require.NoError(t, err)

	_, err = managerB.Parse(token)
	assert.Error(t, err)
}

func TestJWTClaimsRegisteredFields(t *testing.T) {
	manager := jwtutil.NewManager("test-secret", time.Hour)
	userID := uuid.New()

	token, _, err := manager.Generate(userID, "member@example.com", "member")
	require.NoError(t, err)

	parser := jwt.NewParser()
	parsed, _, err := parser.ParseUnverified(token, &jwtutil.Claims{})
	require.NoError(t, err)

	claims := parsed.Claims.(*jwtutil.Claims)
	assert.Equal(t, userID.String(), claims.Subject)
	assert.NotNil(t, claims.ExpiresAt)
}

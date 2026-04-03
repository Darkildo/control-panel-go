package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_Generate_And_Verify(t *testing.T) {
	manager := NewManager("test-secret-key", time.Hour)

	userID := int64(42)
	login := "admin"

	token, err := manager.Generate(userID, login)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, login, claims.Login)
}

func TestManager_Verify_ExpiredToken(t *testing.T) {
	manager := NewManager("test-secret-key", -time.Hour)

	token, err := manager.Generate(1, "user")
	require.NoError(t, err)

	_, err = manager.Verify(token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token")
}

func TestManager_Verify_InvalidToken(t *testing.T) {
	manager := NewManager("test-secret-key", time.Hour)

	_, err := manager.Verify("totally-invalid-token")
	require.Error(t, err)
}

func TestManager_Verify_WrongSecret(t *testing.T) {
	manager1 := NewManager("secret-1", time.Hour)
	manager2 := NewManager("secret-2", time.Hour)

	token, err := manager1.Generate(1, "user")
	require.NoError(t, err)

	_, err = manager2.Verify(token)
	require.Error(t, err)
}

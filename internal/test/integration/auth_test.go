//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_Success(t *testing.T) {
	truncate(t)

	w := registerUser("alice", "password123")

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["id"])
	assert.Equal(t, "alice", data["username"])
	assert.NotEmpty(t, data["created_at"])
}

func TestRegister_DuplicateUsername(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)

	w2 := registerUser("alice", "different-password")

	assert.Equal(t, http.StatusConflict, w2.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "USERNAME_TAKEN", errObj["code"])
}

func TestRegister_InvalidBody(t *testing.T) {
	truncate(t)

	// Missing password
	w := do("POST", "/api/v1/auth/register", `{"username":"alice"}`, "")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "INVALID_REQUEST", errObj["code"])
}

func TestLogin_Success(t *testing.T) {
	truncate(t)

	w1 := registerUser("bob", "mypassword")
	require.Equal(t, http.StatusCreated, w1.Code)

	w := do("POST", "/api/v1/auth/login", `{"username":"bob","password":"mypassword"}`, "")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["access_token"])
	assert.NotZero(t, data["expires_at"])
}

func TestLogin_WrongPassword(t *testing.T) {
	truncate(t)

	w1 := registerUser("charlie", "correctpass")
	require.Equal(t, http.StatusCreated, w1.Code)

	w := do("POST", "/api/v1/auth/login", `{"username":"charlie","password":"wrongpass"}`, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "INVALID_CREDENTIALS", errObj["code"])
}

func TestLogin_UnknownUser(t *testing.T) {
	truncate(t)

	w := do("POST", "/api/v1/auth/login", `{"username":"nobody","password":"pass"}`, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "INVALID_CREDENTIALS", errObj["code"])
}

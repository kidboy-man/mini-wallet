//go:build integration

package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBalance_AfterRegister(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)

	token := loginAs(t, "alice", "password123")
	w := do("GET", "/api/v1/wallets/me/balance", "", token)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]any)
	assert.Equal(t, "0.00", data["balance"])
	assert.Equal(t, "0.00", data["locked_amount"])
	assert.Equal(t, "0.00", data["available_balance"])
}

func TestGetBalance_RequiresAuth(t *testing.T) {
	truncate(t)

	w := do("GET", "/api/v1/wallets/me/balance", "", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "UNAUTHORIZED", errObj["code"])
}

func TestTopUp_Success(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	token := loginAs(t, "alice", "password123")

	body := `{"amount":"500","reference_id":"ref-topup-1"}`
	w := do("POST", "/api/v1/wallets/topup", body, token)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["transaction_id"])
	assert.Equal(t, "500.00", data["balance"])
}

func TestTopUp_Idempotent(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	token := loginAs(t, "alice", "password123")

	body := `{"amount":"500","reference_id":"ref-idem-1"}`

	w1r := do("POST", "/api/v1/wallets/topup", body, token)
	require.Equal(t, http.StatusCreated, w1r.Code)

	var resp1 map[string]any
	require.NoError(t, json.Unmarshal(w1r.Body.Bytes(), &resp1))
	txID1 := resp1["data"].(map[string]any)["transaction_id"]

	// Second call with same reference_id
	w2r := do("POST", "/api/v1/wallets/topup", body, token)
	assert.Equal(t, http.StatusCreated, w2r.Code)

	var resp2 map[string]any
	require.NoError(t, json.Unmarshal(w2r.Body.Bytes(), &resp2))
	txID2 := resp2["data"].(map[string]any)["transaction_id"]

	// Same transaction returned, balance not doubled
	assert.Equal(t, txID1, txID2)
	assert.Equal(t, "500.00", resp2["data"].(map[string]any)["balance"])
}

func TestWithdraw_Success(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	token := loginAs(t, "alice", "password123")

	// TopUp 500 first
	doTopUp := do("POST", "/api/v1/wallets/topup", `{"amount":"500","reference_id":"ref-tu-1"}`, token)
	require.Equal(t, http.StatusCreated, doTopUp.Code)

	// Withdraw 200
	body := `{"amount":"200","reference_id":"ref-wd-1"}`
	w := do("POST", "/api/v1/wallets/withdraw", body, token)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["transaction_id"])
	assert.Equal(t, "300.00", data["available_balance"])
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	token := loginAs(t, "alice", "password123")

	// TopUp 100, try to withdraw 500
	do("POST", "/api/v1/wallets/topup", `{"amount":"100","reference_id":"ref-tu-1"}`, token)

	body := `{"amount":"500","reference_id":"ref-wd-1"}`
	w := do("POST", "/api/v1/wallets/withdraw", body, token)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "INSUFFICIENT_BALANCE", errObj["code"])
}

func TestWithdraw_DuplicateReference(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	token := loginAs(t, "alice", "password123")

	do("POST", "/api/v1/wallets/topup", `{"amount":"500","reference_id":"ref-tu-1"}`, token)

	body := `{"amount":"100","reference_id":"ref-wd-dup"}`

	w1r := do("POST", "/api/v1/wallets/withdraw", body, token)
	require.Equal(t, http.StatusCreated, w1r.Code)

	var resp1 map[string]any
	require.NoError(t, json.Unmarshal(w1r.Body.Bytes(), &resp1))
	txID1 := resp1["data"].(map[string]any)["transaction_id"]

	// Duplicate
	w2r := do("POST", "/api/v1/wallets/withdraw", body, token)
	assert.Equal(t, http.StatusCreated, w2r.Code)

	var resp2 map[string]any
	require.NoError(t, json.Unmarshal(w2r.Body.Bytes(), &resp2))
	txID2 := resp2["data"].(map[string]any)["transaction_id"]

	assert.Equal(t, txID1, txID2)
}

func TestTransfer_Success(t *testing.T) {
	truncate(t)

	// Register sender (alice) and receiver (bob)
	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	w2 := registerUser("bob", "password123")
	require.Equal(t, http.StatusCreated, w2.Code)

	aliceToken := loginAs(t, "alice", "password123")
	bobToken := loginAs(t, "bob", "password123")

	// Top up alice 500
	do("POST", "/api/v1/wallets/topup", `{"amount":"500","reference_id":"ref-tu-alice"}`, aliceToken)

	// Transfer 100 from alice to bob
	body := `{"to_username":"bob","amount":"100","reference_id":"ref-tf-1"}`
	w := do("POST", "/api/v1/wallets/transfer", body, aliceToken)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["transaction_id"])
	assert.Equal(t, "400.00", data["available_balance"]) // alice: 500 - 100

	// Verify bob's balance
	wBob := do("GET", "/api/v1/wallets/me/balance", "", bobToken)
	require.Equal(t, http.StatusOK, wBob.Code)

	var bobResp map[string]any
	require.NoError(t, json.Unmarshal(wBob.Body.Bytes(), &bobResp))
	bobData := bobResp["data"].(map[string]any)
	assert.Equal(t, "100.00", bobData["balance"])
}

func TestTransfer_RecipientNotFound(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	aliceToken := loginAs(t, "alice", "password123")

	do("POST", "/api/v1/wallets/topup", `{"amount":"500","reference_id":"ref-tu-1"}`, aliceToken)

	body := fmt.Sprintf(`{"to_username":"nonexistent","amount":"100","reference_id":"ref-tf-1"}`)
	w := do("POST", "/api/v1/wallets/transfer", body, aliceToken)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "RECIPIENT_NOT_FOUND", errObj["code"])
}

func TestTransfer_InsufficientBalance(t *testing.T) {
	truncate(t)

	w1 := registerUser("alice", "password123")
	require.Equal(t, http.StatusCreated, w1.Code)
	w2 := registerUser("bob", "password123")
	require.Equal(t, http.StatusCreated, w2.Code)

	aliceToken := loginAs(t, "alice", "password123")

	// Alice has 0 balance; try to transfer 100 to bob
	body := `{"to_username":"bob","amount":"100","reference_id":"ref-tf-1"}`
	w := do("POST", "/api/v1/wallets/transfer", body, aliceToken)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "INSUFFICIENT_BALANCE", errObj["code"])
}

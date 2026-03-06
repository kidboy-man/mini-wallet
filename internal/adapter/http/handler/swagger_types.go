package handler

// SuccessResponse wraps all successful API responses.
type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

// ErrorDetail is the API error payload.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps all failed API responses.
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type RegisterResponseData struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type RegisterSuccessResponse struct {
	Success bool                 `json:"success"`
	Data    RegisterResponseData `json:"data"`
}

type LoginResponseData struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   string `json:"expires_at"`
}

type LoginSuccessResponse struct {
	Success bool              `json:"success"`
	Data    LoginResponseData `json:"data"`
}

type BalanceResponseData struct {
	Balance          string `json:"balance"`
	LockedAmount     string `json:"locked_amount"`
	AvailableBalance string `json:"available_balance"`
}

type BalanceSuccessResponse struct {
	Success bool                `json:"success"`
	Data    BalanceResponseData `json:"data"`
}

type TopUpResponseData struct {
	TransactionID string `json:"transaction_id"`
	Balance       string `json:"balance"`
}

type TopUpSuccessResponse struct {
	Success bool              `json:"success"`
	Data    TopUpResponseData `json:"data"`
}

type WithdrawTransferResponseData struct {
	TransactionID    string `json:"transaction_id"`
	AvailableBalance string `json:"available_balance"`
}

type WithdrawTransferSuccessResponse struct {
	Success bool                         `json:"success"`
	Data    WithdrawTransferResponseData `json:"data"`
}

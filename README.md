# Mini Wallet API

Mini Wallet is a RESTful backend service for digital wallet operations.
It is built with Go + PostgreSQL using Hexagonal Architecture (Ports and Adapters), with JWT authentication, idempotent transaction handling, optimistic locking, and an immutable transaction ledger.

## Features

- User registration and login (`JWT` access token)
- One wallet per user, created atomically at registration
- Balance inquiry (`balance`, `locked_amount`, `available_balance`)
- Top up, withdraw, and transfer operations
- Idempotency with `reference_id` for transaction requests
- Concurrency protection with optimistic locking + retry
- Race-enabled test commands by default (`go test --race` via Make targets)
- Swagger/OpenAPI docs with interactive UI

## Tech Stack

- Go `1.25+`
- Gin (HTTP router)
- PostgreSQL (`pgx/v5`)
- golang-migrate
- JWT (`golang-jwt/v5`)
- shopspring/decimal for money arithmetic
- Swaggo (Swagger docs)

## Project Structure

```text
cmd/server/main.go                 # bootstrap, migrations, DI wiring, server startup
config/config.go                   # env config loader
docs/                              # generated Swagger/OpenAPI artifacts
internal/
	adapter/
		http/                          # Gin router, handlers, middleware
		repository/                    # PostgreSQL repository implementations
	core/
		domain/                        # entities and app errors
		port/                          # interfaces (repositories/services)
		service/                       # business use-cases
	infrastructure/
		db/                            # pgx pool + tx context helpers
		token/                         # JWT implementation
	test/
		unit/                          # unit tests
		integration/                   # integration tests
migrations/                        # SQL migrations
```

## Architecture and Flow

The service follows Hexagonal Architecture:

- `core/domain`: pure business entities and error types
- `core/service`: use-case orchestration and business rules
- `core/port`: interfaces for dependencies
- `adapter/repository`: DB adapters implementing repository ports
- `adapter/http`: transport adapter (Gin handlers)

Request flow:

1. HTTP request enters Gin route (`/api/v1/...`)
2. JWT middleware validates token for protected routes
3. Handler validates input and calls service layer
4. Service executes business logic inside transaction manager when needed
5. Repository executes SQL using ambient transaction from context
6. Standard JSON envelope response is returned

## Money and Consistency Rules

- DB precision: `DECIMAL(20,4)` for persisted money values
- API response formatting: fixed to `2` decimals (for client-facing output)
- No floating point math (`float64`) for monetary operations
- `available_balance = balance - locked_amount`

Consistency patterns:

- Optimistic locking on wallet updates via `version`
- Retry up to 3 times for lock contention on debit flows
- Virtual lock for debits using `locked_amount` to prevent overspending
- Immutable transaction ledger (status transitions and audit trail)

## API Overview

Base URL: `http://localhost:8080/api/v1`

Public endpoints:

- `POST /auth/register`
- `POST /auth/login`

Protected endpoints (`Authorization: Bearer <token>`):

- `GET /wallets/me/balance`
- `POST /wallets/topup`
- `POST /wallets/withdraw`
- `POST /wallets/transfer`

Response envelope:

```json
{
	"success": true,
	"data": {}
}
```

Error envelope:

```json
{
	"success": false,
	"error": {
		"code": "ERROR_CODE",
		"message": "human readable message"
	}
}
```

## Quick Start

### 1. Environment

Create `.env` (minimum required):

```env
JWT_SECRET=your-secret-key
```

Optional values (defaults are applied):

- `SERVER_PORT=8080`
- `DB_HOST=localhost`
- `DB_PORT=5432`
- `DB_USER=postgres`
- `DB_PASSWORD=`
- `DB_NAME=mini_wallet`
- `DATABASE_URL=postgres://...` (overrides DB_* values)
- `JWT_EXPIRY_MINS=15`
- `BCRYPT_COST=12`
- `APP_ENV=development`

### 2. Run with Docker (recommended)

```bash
make docker-up
```

Stop services:

```bash
make docker-down
```

### 3. Run locally

Ensure PostgreSQL is running and env vars are set, then:

```bash
make run
```

The server applies DB migrations automatically on startup.

## Swagger / OpenAPI

Generate docs:

```bash
make swagger
```

Open Swagger UI:

- `http://localhost:8080/swagger/index.html`

## Common Commands

```bash
# Build
make build

# Run app
make run

# Run all tests (race-enabled)
make test

# Run integration tests only (race-enabled)
make test-integration

# Coverage (race-enabled)
make test-cover

# Migrations
make migrate-up
make migrate-down
make migrate-create name=add_some_table

# Generate mocks
make mock-gen

# Generate Swagger docs
make swagger
```

## Authentication + Wallet Flow Example

1. Register user

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
	-H "Content-Type: application/json" \
	-d '{"username":"alice","password":"password123"}'
```

2. Login and get token

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
	-H "Content-Type: application/json" \
	-d '{"username":"alice","password":"password123"}'
```

3. Top up wallet

```bash
curl -X POST http://localhost:8080/api/v1/wallets/topup \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer <access_token>" \
	-d '{"amount":"500.00","reference_id":"topup-001"}'
```

4. Withdraw from wallet

```bash
curl -X POST http://localhost:8080/api/v1/wallets/withdraw \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer <access_token>" \
	-d '{"amount":"100.00","reference_id":"wd-001"}'
```

5. Check balance

```bash
curl -X GET http://localhost:8080/api/v1/wallets/me/balance \
	-H "Authorization: Bearer <access_token>"
```

## Idempotency Behavior

- `reference_id` is used to protect against duplicate client retries.
- If the same authenticated user sends the same operation again with the same `reference_id`, the API returns the original successful transaction result instead of applying the balance mutation twice.
- This behavior is implemented for top up, withdraw, and transfer flows.

## Testing Strategy

- Unit tests: `internal/test/unit/`
- Integration tests: `internal/test/integration/` (tag: `integration`)
- All Make test targets run with race detection enabled.

Run manually:

```bash
go test ./internal/test/unit/... -v --race
go test -tags=integration ./internal/test/integration/... -v --race
```

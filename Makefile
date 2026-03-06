BINARY_NAME=mini-wallet
MAIN_PATH=./cmd/server
MIGRATE_PATH=./migrations
DATABASE_URL?=$(shell grep DATABASE_URL .env 2>/dev/null | cut -d '=' -f2)

.PHONY: build run test test-cover docker-up docker-down migrate-up migrate-down migrate-create mock-gen tidy

## Build the binary
build:
	go build -ldflags="-s -w" -o bin/$(BINARY_NAME) $(MAIN_PATH)

## Run the server locally (requires .env)
run:
	go run $(MAIN_PATH)

## Run all tests
test:
	go test ./... -v -count=1

## Run tests with coverage report
test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Apply all pending migrations
migrate-up:
	migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up

## Roll back the last migration
migrate-down:
	migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down 1

## Create a new migration (usage: make migrate-create name=add_some_table)
migrate-create:
	migrate create -ext sql -dir $(MIGRATE_PATH) -seq $(name)

## Generate mocks for port interfaces
mock-gen:
	go generate ./...

## Start Docker services (postgres + app)
docker-up:
	docker compose up --build -d

## Stop Docker services
docker-down:
	docker compose down

## Remove Docker services and volumes
docker-clean:
	docker compose down -v

## Run integration tests (requires Docker)
test-integration:
	go test -v -count=1 -tags=integration -timeout=120s ./internal/test/integration/...

## Tidy go modules
tidy:
	go mod tidy

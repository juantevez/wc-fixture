.PHONY: run build test test-unit test-integration lint migrate-up migrate-down

SERVICE = fixture-core
BINARY  = bin/$(SERVICE)
MAIN    = ./cmd/server

# ── Desarrollo ────────────────────────────────────────────────────────────────
run:
	@go run $(MAIN)/...

build:
	@mkdir -p bin
	@go build -ldflags="-s -w" -o $(BINARY) $(MAIN)/...
	@echo "Binary: $(BINARY)"

# ── Tests ─────────────────────────────────────────────────────────────────────
test:
	@go test ./... -v -race -timeout 120s

test-unit:
	@go test ./internal/domain/... -v -race -short

test-integration:
	@go test ./internal/infrastructure/... -v -race -timeout 120s

# ── Calidad ───────────────────────────────────────────────────────────────────
lint:
	@golangci-lint run ./...

# ── Migraciones ───────────────────────────────────────────────────────────────
migrate-up:
	@migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	@migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-status:
	@migrate -path ./migrations -database "$(DATABASE_URL)" version

# ── Docker ────────────────────────────────────────────────────────────────────
docker-build:
	@docker build -t $(SERVICE):latest .

docker-run:
	@docker run --rm --env-file .env -p 8080:8080 $(SERVICE):latest

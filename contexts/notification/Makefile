.PHONY: run build test lint

SERVICE = notification
BINARY  = bin/$(SERVICE)
MAIN    = ./cmd/server

run:
	@go run $(MAIN)/...

build:
	@mkdir -p bin
	@go build -ldflags="-s -w" -o $(BINARY) $(MAIN)/...

test:
	@go test ./... -v -race -timeout 120s

test-unit:
	@go test ./internal/domain/... -v -race -short

lint:
	@golangci-lint run ./...

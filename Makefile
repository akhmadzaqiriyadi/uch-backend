# Variables for build metadata
VERSION ?= 1.0.0
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "development")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -w -s -X 'gozaq/config.Version=$(VERSION)' -X 'gozaq/config.GitCommit=$(GIT_COMMIT)' -X 'gozaq/config.BuildTime=$(BUILD_TIME)'

# Run application directly
run:
	go run cmd/api/main.go

# Run with live hot-reloading (auto-restart on file save)
dev:
	air

# Install Git pre-commit hooks
hook-install:
	chmod +x .githooks/pre-commit
	git config core.hooksPath .githooks

# Run database seeder in Development mode (Admin + 20 sample users)
seed: seed-dev

seed-dev:
	go run cmd/seed/main.go -env=dev

# Run database seeder in Production mode (Root Admin only, no fake users)
seed-prod:
	go run cmd/seed/main.go -env=prod

# Build binary with metadata
build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/api cmd/api/main.go

# Run unit tests with race detector and coverage
test:
	go test -v -race -cover ./...

# Auto-format code and clean imports (like Prettier)
fmt:
	gofmt -s -w .
	which goimports > /dev/null && goimports -w . || true

# Run standard linter (like ESLint)
lint:
	golangci-lint run ./...

# Scan for known vulnerabilities (like npm audit)
audit:
	govulncheck ./...

# Download and clean dependencies
tidy:
	go mod tidy

# Start docker compose containers
docker-up:
	docker compose up -d --build

# Stop docker compose containers
docker-down:
	docker compose down

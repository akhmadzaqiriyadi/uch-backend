.PHONY: run dev build test tidy fmt lint audit docker-up docker-down

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

# Run database seeder (Admin & sample users)
seed:
	go run cmd/seed/main.go

# Build binary
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/api cmd/api/main.go

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

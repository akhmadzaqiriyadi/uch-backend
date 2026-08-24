# Stage 1: Build binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically compiled binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api ./cmd/api

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates and tzdata for SSL & timezones
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/api /app/api
COPY --from=builder /app/migrations /app/migrations

# Expose port
EXPOSE 8080

# Run binary
ENTRYPOINT ["/app/api"]

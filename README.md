<div align="center">

# ⚡ Gozaq - Production-Grade Go Backend Boilerplate

**An enterprise-grade, clean architecture Go backend boilerplate packed with PostgreSQL, Redis Caching, Distributed Locking & Rate Limiting, SingleFlight Anti-Stampede, Circuit Breakers, RBAC, Scalar OpenAPI Docs, Prometheus Observability, Pprof Profiling, and Automated CI/CD.**

[![CI Pipeline](https://github.com/akhmadzaqiriyadi/gozaq/actions/workflows/ci.yml/badge.svg)](https://github.com/akhmadzaqiriyadi/gozaq/actions)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![Clean Architecture](https://img.shields.io/badge/Architecture-Clean%20Arch-FF6B6B?style=flat)](https://blog.cleancoder.com)
[![Scalar Docs](https://img.shields.io/badge/API%20Docs-Scalar%20UI-7C3AED?style=flat)](http://localhost:8080/docs)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL%20%2B%20pgx-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Cache%20%26%20Lock-Redis-DC382D?style=flat&logo=redis)](https://redis.io)
[![Faker](https://img.shields.io/badge/Seeder-gofakeit%20v7-10B981?style=flat)](https://github.com/brianvoe/gofakeit)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## 🌟 High-Concurrency & Enterprise Features

* **🏛️ Clean & Layered Architecture**: Domain-driven design with strict separation of concerns (`domain` $\to$ `service` $\to$ `repository` $\to$ `handler`).
* **🐘 High-Performance PostgreSQL**: Powered by `pgx/v5` with connection pooling, statement caching, **automatic startup migrations**, and Prometheus pool metrics.
* **🛡️ SingleFlight (`golang.org/x/sync/singleflight`) - Anti Cache Stampede**:
  - Eliminates the *Thundering Herd* problem when cache expires under high load by deduplicating concurrent database queries.
* **🔒 Redis Distributed Stack**:
  - **Cache-Aside Engine**: Sub-millisecond profile caching with automatic cache invalidation.
  - **Token Rotation & Revocation**: Instant token invalidation on logout.
  - **Distributed Lock (`pkg/cache/lock.go`)**: Multi-instance mutex with atomic Lua release scripts (prevents double-spending / race conditions).
  - **Distributed Sliding-Window Rate Limiter (`pkg/cache/rate_limiter.go`)**: Cluster-wide rate limiting via Redis ZSETs.
* **🔌 Go Runtime Profiler (`/debug/pprof`)**: Built-in endpoints for real-time CPU flamegraphs, heap memory analysis, and goroutine leak detection.
* **⚡ Resilient Circuit Breaker (`pkg/resilience/circuit_breaker.go`)**: Protects the application against cascading failures from third-party APIs / external services.
* **🛡️ Security & Guarding**:
  - **OWASP Security Headers** (Go's Helmet: CSP, HSTS, X-Frame-Options, No-Sniff).
  - **Role-Based Access Control (RBAC)** (`admin` vs `user` roles).
  - **Structured Panic Recovery** logging full stack-traces with unified JSON 500 responses.
* **📖 Interactive Scalar API Reference (`/docs`)**: Modern, interactive documentation embedded directly from OpenAPI 3.1 schema.
* **📊 Observability & Diagnostics**:
  - Real-time diagnostics endpoint (`GET /healthz` checking PostgreSQL, Redis, goroutines, RAM alloc, CPU cores, uptime).
  - Prometheus metrics exporter (`GET /metrics`) with HTTP latency histograms and PostgreSQL connection pool gauges.
  - Structured JSON logging (`log/slog`) exposing `X-Request-ID` and `X-Response-Time` headers.
* **📁 File Upload & Static Storage**: Multipart upload handler with MIME type sniffing (anti-spoofing) and 5MB size limit.
* **📧 Async Worker Pool & Mailer**: 5 concurrent background workers processing async tasks with graceful drain on server shutdown.
* **🎭 Realistic Faker Database Seeder**: Powered by `gofakeit/v7` (`make seed`) to seed default Admin (`admin@gozaq.com`) + 20 sample users.
* **🧪 Quality & Dev Tooling**:
  - Git Pre-Commit Hook (auto formats with `gofmt`, runs `golangci-lint`, and executes `go test -race`).
  - Live Hot-Reloading (`air`) via `make dev`.
  - Comprehensive unit & HTTP integration tests with `testify/mock`.

---

## 🏗️ Project Architecture Layout

```text
gozaq/
├── cmd/
│   ├── api/              # HTTP Server bootstrap & DI wiring
│   └── seed/             # Realistic Database Seeder using gofakeit v7
├── config/               # Environment variables & build-time metadata (ldflags)
├── docs/                 # OpenAPI 3.1 specification schema
├── internal/
│   ├── domain/           # Core Entities, DTOs, and Repository/Service Interfaces
│   ├── handler/          # HTTP Handlers (User, Upload, Docs, Diagnostics)
│   │   └── middleware/   # Auth, Security Headers, Rate Limiter, Logger, Metrics, Recovery
│   ├── repository/       # Data Access Layer (PostgreSQL with pgxpool & DBTX)
│   └── service/          # Business Logic with SingleFlight & Async Tasks
├── migrations/           # SQL Migration files (.up.sql / .down.sql)
├── pkg/
│   ├── cache/            # Redis client, Distributed Lock, & Distributed Rate Limiter
│   ├── database/         # Transactor (ACID transactions) & AutoMigrator
│   ├── mailer/           # HTML Email Service
│   ├── pagination/       # Request query parser & pagination metadata
│   ├── resilience/       # Circuit Breaker pattern
│   ├── response/         # Unified JSON response envelope (200, 201, 400, 401, 403, 404, 409, 422, 429, 500)
│   ├── storage/          # Multipart file storage manager with MIME sniffing
│   ├── validator/        # Struct payload validation (go-playground/validator)
│   └── worker/           # Async Goroutine Worker Pool with graceful shutdown
├── .air.toml             # Live hot-reload config
├── .github/workflows/    # GitHub Actions CI pipeline
├── .githooks/            # Git Pre-commit hook script
├── .golangci.yml         # Strict GolangCI-Lint configuration (v2)
├── docker-compose.yml    # Complete stack (Postgres + Redis + Go API)
├── Dockerfile            # Multi-stage production container image
└── Makefile              # Developer command suite
```

---

## 🚀 Quick Start Guide

### Prerequisites
* **Go** 1.22+
* **PostgreSQL** 14+ & **Redis** (or run via Docker)

### 1. Clone & Setup Environment
```bash
git clone https://github.com/akhmadzaqiriyadi/gozaq.git
cd gozaq

# Copy environment variables
cp .env.example .env
```

### 2. Install Git Pre-Commit Hooks
```bash
make hook-install
```

### 3. Seed Realistic Database
Populate database with default Admin (`admin@gozaq.com` / `Admin123!`) and 20 sample users generated by `gofakeit`:
```bash
make seed
```

### 4. Run Development Server (with Live Hot-Reload)
```bash
make dev
```
The server will start on `http://localhost:8080`.

---

## 💡 Advanced Technical Patterns & Code Examples

### 1. Anti Cache-Stampede with SingleFlight
```go
// Only 1 concurrent DB query per key; other concurrent callers wait and share the result
v, err, _ := s.sfGroup.Do(cacheKey, func() (any, error) {
    user, err := s.repo.GetByID(ctx, userID)
    if err != nil {
        return nil, err
    }
    resp := user.ToResponse()
    _ = s.cache.Set(ctx, cacheKey, resp, 15*time.Minute)
    return &resp, nil
})
```

### 2. Redis Distributed Locking (Multi-Instance Mutex)
```go
locker := cache.NewRedisLocker(redisClient)

// Safely execute critical operations across multiple server instances
err := locker.WithLock(ctx, "order:checkout:user-123", 5*time.Second, func(ctx context.Context) error {
    return processPayment()
})
```

### 3. Circuit Breaker for External Service Calls
```go
cb := resilience.NewCircuitBreaker(5, 30*time.Second)

err := cb.Execute(func() error {
    return thirdPartyPaymentAPI.Charge(payload)
})
```

### 4. Atomic Database Transactions (Unit of Work)
```go
err := database.WithinTransaction(ctx, dbPool, func(txCtx context.Context) error {
    if err := userRepo.Create(txCtx, user); err != nil {
        return err // Automatically ROLLBACK
    }
    return walletRepo.Create(txCtx, wallet) // Automatically COMMIT if all succeed
})
```

---

## 🐳 Running with Docker Compose

To launch the complete stack (PostgreSQL + Redis + API) in one command:

```bash
# Start all services in background
make docker-up

# View logs
docker compose logs -f app

# Stop services
make docker-down
```

---

## 📖 Interactive API Documentation

Once the server is running, open your browser:
👉 **[http://localhost:8080/docs](http://localhost:8080/docs)**

Rendered by **Scalar UI**, you can test every endpoint directly from the browser, inspect exact JSON request/response schemas, and explore all status codes (`200`, `201`, `400`, `401`, `403`, `404`, `409`, `422`, `429`, `500`).

---

## 🛣️ API Endpoints Summary

| Method | Endpoint | Description | Auth / Role |
| :--- | :--- | :--- | :--- |
| **`GET`** | `/healthz` | System & Dependencies Diagnostics | Public |
| **`GET`** | `/metrics` | Prometheus Metrics Exporter | Public |
| **`GET`** | `/docs` | Scalar Interactive API Reference | Public |
| **`GET`** | `/debug/pprof/*`| Go Runtime Profiler (CPU, Heap, Goroutines) | Public / Internal |
| **`POST`** | `/api/v1/auth/register` | Register new account & generate tokens | Public (Rate Limited 5 RPS) |
| **`POST`** | `/api/v1/auth/login` | Login with email & password | Public (Rate Limited 5 RPS) |
| **`POST`** | `/api/v1/auth/refresh` | Refresh Access Token (Token Rotation) | Public |
| **`POST`** | `/api/v1/auth/logout` | Logout & Revoke Redis Refresh Token | Bearer JWT |
| **`GET`** | `/api/v1/auth/profile` | Get user profile (Redis + SingleFlight) | Bearer JWT |
| **`PUT`** | `/api/v1/auth/profile` | Update profile & invalidate Redis cache | Bearer JWT |
| **`POST`** | `/api/v1/uploads` | Upload avatar/file (JPEG/PNG/PDF max 5MB) | Bearer JWT |
| **`GET`** | `/uploads/*` | Static file server for uploaded media | Public |
| **`GET`** | `/api/v1/users` | List users with pagination & search | **Admin Only** |
| **`DELETE`**| `/api/v1/users/{id}` | Delete user by UUID | **Admin Only** |

---

## 🛠️ Makefile Commands Reference

| Command | Description |
| :--- | :--- |
| **`make dev`** | Runs server with live hot-reloading (`air`) |
| **`make run`** | Runs server directly (`go run cmd/api/main.go`) |
| **`make seed`** | Seeds database with Admin & 20 sample users using `gofakeit` |
| **`make hook-install`** | Installs Git pre-commit hook in `.git/hooks` |
| **`make build`** | Builds optimized binary with version & commit SHA metadata in `bin/api` |
| **`make test`** | Runs unit & integration tests with race detector and coverage |
| **`make lint`** | Runs `golangci-lint` with strict rules (0 issues) |
| **`make fmt`** | Formats code with `gofmt` and `goimports` |
| **`make audit`** | Scans dependencies with `govulncheck` (0 vulnerabilities) |
| **`make tidy`** | Cleans and optimizes `go.mod` and `go.sum` |
| **`make docker-up`** | Starts PostgreSQL, Redis, and API via Docker Compose |
| **`make docker-down`**| Stops all Docker Compose containers |

---

## 📄 License

This project is licensed under the **MIT License** - feel free to use it for your personal projects, startups, or enterprise applications!

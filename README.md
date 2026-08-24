<div align="center">

# ⚡ Gozaq - Production-Grade Go Backend Boilerplate

**An enterprise-grade, clean architecture Go backend boilerplate packed with PostgreSQL, SQLC Configuration, Redis Caching, Distributed Locking & Rate Limiting, Dynamic Multi-Role RBAC & PBAC Permissions, SingleFlight Anti-Stampede, Circuit Breakers, Scalar OpenAPI Docs, Prometheus Observability, and Automated CI/CD.**

[![CI Pipeline](https://github.com/akhmadzaqiriyadi/gozaq/actions/workflows/ci.yml/badge.svg)](https://github.com/akhmadzaqiriyadi/gozaq/actions)
[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![Clean Architecture](https://img.shields.io/badge/Architecture-Clean%20Arch-FF6B6B?style=flat)](https://blog.cleancoder.com)
[![SQLC Ready](https://img.shields.io/badge/SQL-SQLC%20Ready-00ADD8?style=flat)](https://sqlc.dev)
[![Scalar Docs](https://img.shields.io/badge/API%20Docs-Scalar%20UI-7C3AED?style=flat)](http://localhost:8080/docs)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL%20%2B%20pgx-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Cache%20%26%20Lock-Redis-DC382D?style=flat&logo=redis)](https://redis.io)
[![Faker](https://img.shields.io/badge/Seeder-gofakeit%20v7-10B981?style=flat)](https://github.com/brianvoe/gofakeit)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## 🌟 High-Concurrency & Enterprise Features

* **🏛️ Clean & Layered Architecture**: Domain-driven design with strict separation of concerns (`domain` $\to$ `service` $\to$ `repository` $\to$ `handler`).
* **🐘 High-Performance PostgreSQL & SQLC Ready**:
  - Powered by `pgx/v5` with connection pooling, statement caching, **automatic startup migrations**, and Prometheus pool metrics.
  - Includes **SQLC (`sqlc.yaml` & `sql/queries/*.sql`)** ready for type-safe code generation.
* **👑 Dynamic Multi-Role RBAC & PBAC (Granular Permissions)**:
  - Default Roles: `admin`, `manager`, `user`.
  - Granular Permissions: `users:read`, `users:create`, `users:update`, `users:delete`, `roles:read`, `roles:manage`, `uploads:create`, `audit:read`.
  - **Permission Guards**: `middleware.RequirePermission("users:delete")` and `middleware.RequireAnyPermission(...)`.
  - **Dynamic Role Assignment Endpoint**: `PUT /api/v1/users/{id}/role` (instantly recalculates permissions & evicts cache).
  - Sub-millisecond Redis permission caching (`role:permissions:<role_id>`).
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
│   ├── domain/           # Core Entities (User, RBAC), DTOs, and Repository/Service Interfaces
│   ├── handler/          # HTTP Handlers (User, Upload, RBAC, Docs, Diagnostics)
│   │   └── middleware/   # Auth (JWT & PBAC), Security Headers, Rate Limiter, Logger, Metrics, Recovery
│   ├── repository/       # Data Access Layer (PostgreSQL with pgxpool, DBTX, & RBAC)
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
├── sql/
│   └── queries/          # Type-safe SQL query files for SQLC (users.sql, rbac.sql)
├── sqlc.yaml             # SQLC codegen configuration
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

## 💡 Dynamic RBAC & PBAC Usage

```go
// 1. Guard endpoint with granular permission
r.With(middleware.RequirePermission("users:delete")).
    Delete("/users/{id}", userHandler.DeleteUser)

// 2. Guard endpoint with multiple permission options
r.With(middleware.RequireAnyPermission("roles:read", "roles:manage")).
    Get("/roles", rbacHandler.ListRoles)
```

---

## 🐳 Running with Docker Compose

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

---

## 🛣️ API Endpoints Summary

| Method | Endpoint | Description | Permission / Guard |
| :--- | :--- | :--- | :--- |
| **`GET`** | `/healthz` | System & Dependencies Diagnostics | Public |
| **`GET`** | `/metrics` | Prometheus Metrics Exporter | Public |
| **`GET`** | `/docs` | Scalar Interactive API Reference | Public |
| **`GET`** | `/debug/pprof/*`| Go Runtime Profiler (CPU, Heap, Goroutines) | Public / Internal |
| **`POST`** | `/api/v1/auth/register` | Register new account & generate tokens | Public (5 RPS) |
| **`POST`** | `/api/v1/auth/login` | Login with email & password | Public (5 RPS) |
| **`POST`** | `/api/v1/auth/refresh` | Refresh Access Token (Token Rotation) | Public |
| **`POST`** | `/api/v1/auth/logout` | Logout & Revoke Redis Refresh Token | Bearer JWT |
| **`GET`** | `/api/v1/auth/profile` | Get user profile with permissions | Bearer JWT |
| **`PUT`** | `/api/v1/auth/profile` | Update profile & invalidate Redis cache | Bearer JWT |
| **`POST`** | `/api/v1/uploads` | Upload avatar/file (JPEG/PNG/PDF max 5MB) | `uploads:create` |
| **`GET`** | `/uploads/*` | Static file server for uploaded media | Public |
| **`GET`** | `/api/v1/users` | List users with pagination & search | `users:read` |
| **`PUT`** | `/api/v1/users/{id}/role` | Update user role dynamically | `roles:manage` |
| **`DELETE`**| `/api/v1/users/{id}` | Delete user by UUID | `users:delete` |
| **`GET`** | `/api/v1/roles` | List all system roles & permissions | `roles:read` |
| **`GET`** | `/api/v1/permissions` | List all master system permissions | `roles:read` |
| **`POST`** | `/api/v1/roles/{role_id}/permissions/{permission_id}` | Assign permission to role | `roles:manage` |
| **`DELETE`**| `/api/v1/roles/{role_id}/permissions/{permission_id}` | Revoke permission from role | `roles:manage` |

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

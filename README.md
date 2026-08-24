# Go Backend Production-Grade Clean Architecture Template

Template backend Golang standar industri berbasis **Clean Architecture**, **Redis Caching**, **RBAC**, **Token Rotation**, dan **Prometheus Observability**.

---

## 📁 Struktur Direktori & Layer

```text
gozaq/
├── cmd/
│   └── api/
│       └── main.go                 # Entrypoint: Dependency Injection & Graceful Shutdown
├── config/
│   └── config.go                   # Environment configuration loader (.env & OS Env)
├── internal/                       # Private code (Compiler-protected)
│   ├── domain/                     # Core Business Entities & Interface Contracts
│   │   ├── user.go                 # Models, DTOs (Auth, Register, CRUD), & Interfaces
│   │   └── errors.go               # Standard Domain Errors
│   ├── handler/                    # Transport Layer (HTTP)
│   │   ├── router.go               # Chi router, Middleware, RBAC Guard, Prometheus
│   │   ├── user_handler.go         # Controller HTTP (Auth & User CRUD)
│   │   ├── docs_handler.go         # Scalar OpenAPI 3.1 Documentation UI Handler
│   │   ├── openapi.json            # Embedded OpenAPI 3.1 Spec
│   │   └── middleware/
│   │       ├── auth.go             # JWT Authentication & RBAC RequireRole Guard
│   │       ├── security_headers.go # OWASP Security Headers (Go's Helmet)
│   │       ├── rate_limiter.go     # Token Bucket Rate Limiter per IP
│   │       ├── logger.go           # Structured JSON slog request logger + Latency ms
│   │       ├── recovery.go         # Structured Panic Recovery (Returns 500 JSON)
│   │       └── metrics.go          # Prometheus HTTP Metrics Collector
│   ├── service/                    # Business Logic Layer (Use cases, bcrypt, JWT, Cache)
│   │   ├── user_service.go
│   │   └── user_service_test.go    # Unit tests with Mock repository
│   └── repository/                 # Data Access Layer
│       └── postgres/
│           └── user_repo.go        # PostgreSQL implementation using pgxpool
├── pkg/                            # Reusable cross-project utilities
│   ├── cache/                      # Redis Cache Layer (Cache-Aside + Noop fallback)
│   ├── pagination/                 # Generic pagination, sorting, & metadata builder
│   ├── response/                   # Standardized JSON response helper
│   ├── validator/                  # Request payload validation helper
│   └── worker/                     # Async Goroutine Worker Pool (Non-blocking tasks)
├── migrations/                     # SQL database migration scripts
│   ├── 000001_create_users_table.up.sql
│   └── 000001_create_users_table.down.sql
├── .github/workflows/ci.yml        # GitHub Actions CI/CD Pipeline
├── .air.toml                       # Live hot-reloading configuration
├── .golangci.yml                   # Strict linter configuration (0 issues)
├── Dockerfile                      # Multi-stage production Alpine image
├── docker-compose.yml              # PostgreSQL + Redis + API stack
├── Makefile                        # Shortcuts for development tasks
├── .env.example & .env             # Environment configuration
├── go.mod & go.sum
└── README.md
```

---

## 🚀 Fitur Utama & Teknologi

1. **Autentikasi Modern (Access Token + Refresh Token di Redis)**:
   - Access Token (JWT 15 menit) untuk authorization cepat.
   - Refresh Token (7 hari di Redis) dengan mekanisme **Token Rotation & Revocation (Logout)**.
2. **Role-Based Access Control (RBAC)**:
   - Role `user` dan `admin` yang dilindungi dengan `middleware.RequireRole("admin")`.
3. **Redis Caching (Cache-Aside Pattern)**:
   - Profil user otomatis dicache di Redis (15 menit) dan di-evict otomatis saat update / delete.
   - Dilengkapi **Graceful Fallback (*NoopCache*)** jika Redis offline.
4. **Asynchronous Background Worker Pool**:
   - Menjalankan proses non-blocking (kirim email, analytics) via antrean goroutine pool.
5. **Observability & Docs**:
   - Interactive Docs: **Scalar UI** di `/docs`.
   - Metrics: **Prometheus** di `/metrics`.

---

## 📡 Daftar Endpoint API

| Method | Endpoint | Auth / Role | Deskripsi |
| :--- | :--- | :--- | :--- |
| `GET` | `/docs` | Publik | **Scalar Interactive API Documentation UI** |
| `GET` | `/openapi.json` | Publik | OpenAPI 3.1 Raw JSON |
| `GET` | `/healthz` | Publik | Health check & Liveness probe |
| `GET` | `/metrics` | Publik | Prometheus Metrics endpoint |
| `POST` | `/api/v1/auth/register` | Publik | Registrasi user baru |
| `POST` | `/api/v1/auth/login` | Publik | Login (kembalikan access + refresh token) |
| `POST` | `/api/v1/auth/refresh` | Publik | Refresh token (Token Rotation) |
| `POST` | `/api/v1/auth/logout` | Bearer Token | Logout & revoke refresh token di Redis |
| `GET` | `/api/v1/auth/profile` | Bearer Token | Ambil profil user (Redis Cache-Aside) |
| `PUT` | `/api/v1/auth/profile` | Bearer Token | Update data profil & auto-evict Redis |
| `GET` | `/api/v1/users` | **Admin Only** | List semua user + pagination & search |
| `DELETE` | `/api/v1/users/{id}` | **Admin Only** | Hapus user & invalidate cache |

---

## 🛠️ Perintah Terminal ([`Makefile`](file:///Users/zaq/gozaq/Makefile))

```bash
# 1. Jalankan development server dengan LIVE HOT-RELOAD (Air)
make dev

# 2. Jalankan server biasa
make run

# 3. Jalankan unit test dengan race detector & coverage
make test

# 4. Jalankan linter (golangci-lint)
make lint

# 5. Format kode otomatis (Prettier Go)
make fmt

# 6. Jalankan Security Audit dependensi (govulncheck)
make audit

# 7. Start container PostgreSQL + Redis + API
make docker-up
```

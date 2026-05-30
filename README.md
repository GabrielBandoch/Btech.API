# BTech Fleet Control Platform — Go Backend API

This repository contains the Go backend API for **BTech**, a multi-tenant SaaS fleet management platform. The system is designed to provide high-performance, secure, and extensible fleet operational tracking, compliance management, and auditing.

---

## Technical Stack

- **Core Language**: Go 1.25
- **HTTP Routing**: `github.com/go-chi/chi/v5`
- **Database**: PostgreSQL (Driver: `github.com/jackc/pgx/v5` with connection pool management)
- **Migrations**: Goose (`github.com/pressly/goose/v3` SQL migrations)
- **Authentication**: JWT (HS256) with Refresh Token Rotation (RTR) and bcrypt hashing
- **Entitlements & Quotas**: Billing and plan gating foundations
- **Logging**: Structured JSON Logging via slog

---

## Features

- **Authentication**: Secure login, registration, password policy verification, and active session management.
- **Multi-Tenancy**: Dynamic tenant isolation using a context-propagated `organization_id` at every layer.
- **Role-Based Access Control (RBAC)**: Fine-grained permissions check gates (Owner, Admin, Operator, Viewer).
- **Audit Logs**: dot-notation event taxonomy recording user/system activity under strict tenant boundaries.
- **Billing Foundation**: Entitlements and quota checking (e.g. max drivers, max vehicles) based on subscription tiers (Free, Pro, Enterprise).
- **Fleet Operations**: Realtime tracking, vehicles, and trip management.
- **Vehicle Maintenance**: Schedules, suppliers directory, actual records, and automated alerts for upcoming inspections.
- **Fuel Management**: Log transactions, fuel type validations, efficiency analysis, and anomaly alerts.
- **Driver Documents Center**: File attachment upload/downloads, expiration monitoring, and compliance badges (Compliant, Expiring Soon, Non-Compliant).
- **Driver Detail Page (Phase 1)**: Unified read-only operational dashboard displaying driver details, active documents, and timeline logs.

---

## Architecture & Layers

This project strictly adheres to **Clean Architecture** patterns:

```
delivery (HTTP Handlers & DTOs) ──> usecase (Business Logic Only) ──> domain (Entities & Repository Interfaces) <── repository (PostgreSQL SQL Queries)
```

### Folder Map

```
cmd/api/          ── Entry point (main.go), handles dependency injection and startup
internal/
  config/         ── Environment config loader
  domain/         ── Core entities, interfaces, errors, and constants (Imports nothing)
  usecase/        ── Pure business orchestration (Imports domain only)
  repository/
    postgres/     ── SQL database repository implementations (Imports domain only)
  delivery/
    http/
      handler/    ── Decodes request -> calls UseCase -> encodes response envelope
      middleware/ ── JWT Auth, RBAC check, billing limits, audit context, rate limiters
      dto/        ── Request and response structures (Uses camelCase keys)
      response/   ── Standardized API envelope wrappers ({ success, data, error })
  platform/       ── Database setups, logger initialization, JWT utility libraries
migrations/       ── Chronological Goose database migration files (Goose SQL format)
```

---

## Requirements

- **Go**: Version 1.25 or higher
- **PostgreSQL**: Version 14 or higher

---

## Installation & Setup

1. **Clone the repository**:
   ```bash
   git clone <repo-url> Btech.API
   cd Btech.API
   ```

2. **Configure environment variables**:
   Create a `.env` file from the example:
   ```bash
   cp .env.example .env
   ```

3. **Install Go dependencies**:
   ```bash
   go mod download
   go mod verify
   ```

4. **Run database migrations**:
   Ensure PostgreSQL is running, then run migrations via goose (integrated in startup or CLI):
   ```bash
   go run cmd/api/main.go
   ```

5. **Start the API server**:
   ```bash
   go run cmd/api/main.go
   ```

---

## Environment Variables

The application reads configurations from the environment:

| Variable | Description | Example / Default |
|---|---|---|
| `PORT` | API Listening Port | `8080` |
| `ENV` | Running Environment | `development` / `production` |
| `DATABASE_URL` | PostgreSQL Connection URI | `postgres://user:pass@localhost:5432/db` |
| `JWT_SECRET` | Secret key used for signing tokens | `yoursecret` (Min 32 characters) |
| `JWT_EXPIRES_IN` | Access token lifespan duration | `15m` / `2h` |
| `REFRESH_TOKEN_EXPIRES_IN` | Refresh token lifespan duration | `168h` (7 days) |
| `FRONTEND_URL` | Frontend origin for CORS policy | `http://localhost:5173` |
| `TIMEOUT_SECONDS` | HTTP handler context timeout | `15` |
| `RATE_LIMIT_RATE` | Rate limit tokens per second | `10.0` |
| `RATE_LIMIT_BURST` | Rate limit burst capacity | `20` |
| `STORAGE_BASE_DIR` | Directory where document files are saved | `uploads` |
| `STORAGE_BASE_URL` | Base URL used to download attachments | `http://localhost:8080/uploads` |

---

## Running Tests

### Run Unit Tests
```bash
go test ./internal/usecase/... ./internal/platform/...
```

### Run Integration Tests
Integration tests run against a real database (requires database config set via environment):
```bash
go test ./internal/delivery/http/handler/...
```

---

## Security Controls

1. **Multi-Tenant Isolation**: Handlers extract `organization_id` from validated JWT claims. This tenant ID is passed down as the first argument to the use case and repository layers. All database queries append `WHERE organization_id = $1` to prevent data cross-contamination.
2. **Access Token & Refresh Tokens**: Tokens are short-lived. Session renewals are secured via **Refresh Token Rotation (RTR)**. Compromised token reuse immediately invalidates the entire session tree.
3. **Audit Log System**: Sensitive modifications automatically trigger background audit logging. Events include client metadata (IP address, User Agent) mapped to the organization timeline.
4. **Rate Limiting**: Public endpoints `/auth/*` are guarded by token bucket rate limiters to prevent brute-force attacks.

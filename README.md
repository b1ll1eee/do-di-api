# FlowDo API — Hexagonal Architecture

A production-ready Flowdo REST API in **Go 1.22+** following the **Hexagonal Architecture** (Ports & Adapters) pattern.

The persistence layer is backed by [GORM](https://gorm.io); the core business logic depends only on port interfaces, never on the concrete adapter.

---

## Architecture

```
                         ┌──────────────────────────────────────────────┐
                         │              CORE  (framework-free)           │
                         │                                               │
  ┌──────────────┐       │  ┌──────────────┐    ┌─────────────────────┐ │
  │   HTTP/Gin   │       │  │   domain/    │    │     service/        │ │
  │  (Inbound    │──────▶│  │  Flowdo entity │◀───│  FlowdoService impl   │ │
  │   Adapter)   │inbound│  │  User entity │    │  AuthService impl   │ │
  └──────────────┘ port  │  │  state mach. │    └──────────┬──────────┘ │
                         │  └──────────────┘               │outbound    │
                         └─────────────────────────────────┼────────────┘
                                                           │ port
                                          ┌────────────────▼────────────┐
                                          │     SECONDARY ADAPTERS      │
                                          │                             │
                                          │     ┌──────┐                │
                                          │     │ GORM │  Redis        │
                                          │     └──────┘                │
                                          └─────────────────────────────┘

  Dependency rule: all arrows point INWARD (toward the domain).
  core/ has ZERO imports from adapter/ or any framework.
```

---

## Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Domain** | `internal/core/domain/` | Pure Go structs, enums, domain errors, state-machine methods (`MarkDone`, `StartProgress`). Zero framework or library imports. |
| **Inbound ports** | `internal/core/port/inbound/` | Use-case contracts — `FlowdoService`, `AuthService` interfaces + input/output DTOs. |
| **Outbound ports** | `internal/core/port/outbound/` | Repository contracts — `FlowdoRepository`, `UserRepository` interfaces. |
| **Services** | `internal/core/service/` | Business logic. Depends **only** on port interfaces, never on concrete adapters. |
| **HTTP adapter** | `internal/adapter/inbound/http/` | Gin handlers, JWT middleware, router. Calls inbound port interfaces only. |
| **GORM adapter** | `internal/adapter/outbound/gormrepo/` | GORM + `gorm.io/driver/postgres`. Owns its own persistence models; converts to/from domain types at the boundary. |
| **Cache adapter** | `internal/adapter/outbound/cache/` | Redis-backed generic JSON cache. |
| **Main / DI** | `cmd/api/main.go` | The only file aware of all layers. Wires everything together manually — no DI framework. |
| **Shared utilities** | `pkg/` | `config`, `logger` (zerolog), `response` helpers — framework-agnostic. |

---

## Project Structure

```
flowdo-api/
├── cmd/
│   └── api/
│       └── main.go                    ← Entrypoint & manual DI (adapter switch here)
├── docs/                              ← Swagger docs (run `make swagger`)
├── internal/
│   ├── core/
│   │   ├── domain/
│   │   │   ├── flowdo.go                ← Flowdo entity, Status type, state machine
│   │   │   └── user.go                ← User entity, domain errors
│   │   ├── port/
│   │   │   ├── inbound/
│   │   │   │   ├── flowdo_service.go    ← FlowdoService interface + DTOs
│   │   │   │   └── auth_service.go    ← AuthService interface + DTOs
│   │   │   └── outbound/
│   │   │       ├── flowdo_repository.go
│   │   │       └── user_repository.go
│   │   └── service/
│   │       ├── flowdo_service.go        ← Business logic
│   │       ├── flowdo_service_test.go   ← Unit tests (mocked repos, no DB)
│   │       ├── auth_service.go        ← JWT auth logic
│   │       └── auth_service_test.go
│   └── adapter/
│       ├── inbound/
│       │   └── http/
│       │       ├── handler/
│       │       │   ├── flowdo_handler.go
│       │       │   └── auth_handler.go
│       │       ├── middleware/
│       │       │   └── auth_middleware.go
│       │       └── router/
│       │           └── router.go
│       └── outbound/
│           ├── gormrepo/              ← GORM adapter
│           │   ├── db.go              ← gorm.Open + connection pool setup
│           │   ├── flowdo_repo.go
│           │   ├── flowdo_repo_test.go  ← Integration tests (testcontainers)
│           │   ├── user_repo.go
│           │   └── model/
│           │       └── models.go      ← GORM models (adapter-local, never in core)
│           └── cache/
│               └── redis_cache.go
├── migrations/
│   ├── 000001_create_users_table.up.sql
│   ├── 000001_create_users_table.down.sql
│   ├── 000002_create_flowdos_table.up.sql
│   └── 000002_create_flowdos_table.down.sql
├── mocks/                             ← Generated by mockery
│   ├── FlowdoRepository.go
│   └── UserRepository.go
├── pkg/
│   ├── config/config.go
│   ├── logger/logger.go
│   └── response/response.go
├── .env.example
├── .gitignore
├── .mockery.yaml
├── docker-compose.yml
├── Dockerfile
├── go.mod / go.sum
├── Makefile
└── README.md
```

---

## GORM Adapter — Design Decisions

### Why separate persistence models?

Domain models in `core/domain/` are **pure Go structs** with no GORM tags. GORM-specific models (`internal/adapter/outbound/gormrepo/model/`) live exclusively in the adapter layer. This preserves the hexagonal boundary:

```go
// ✅ core/domain/flowdo.go — no framework tags
type Flowdo struct {
    ID     uuid.UUID
    Status Status
    ...
}

// ✅ adapter/outbound/gormrepo/model/models.go — GORM tags here only
type Flowdo struct {
    ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
    Status string    `gorm:"type:flowdo_status;not null;default:'pending'"`
    ...
}
```

Conversion happens at the adapter boundary:

```go
// inbound: domain → GORM persistence model
m := model.FlowdoFromDomain(flowdo)
r.db.Create(m)

// outbound: GORM persistence model → domain
return m.ToDomain(), nil
```

---

## API Endpoints

### Authentication (public)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/register` | Register a new user |
| `POST` | `/api/v1/login` | Authenticate and receive JWT |

### Flowdos (JWT required — `Authorization: Bearer <token>`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/flowdos` | List flowdos (`?status=&limit=&offset=`) |
| `POST` | `/api/v1/flowdos` | Create a flowdo |
| `GET` | `/api/v1/flowdos/:id` | Get a flowdo by UUID |
| `PUT` | `/api/v1/flowdos/:id` | Update title, description, or status |
| `DELETE` | `/api/v1/flowdos/:id` | Soft-delete a flowdo |

### Other

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Health check |
| `GET` | `/swagger/*` | Swagger UI |

### Response envelope

Every endpoint returns the same JSON shape:

```json
{
  "success": true,
  "data": { ... },
  "error": "",
  "meta": { "limit": 20, "offset": 0, "total": 42 }
}
```

---

## Flowdo State Machine

```
  pending ─────────▶ in_progress ─────────▶ done
     │                                        │
     └────────── no direct jump allowed ──────┘
```

| Domain method | Transition | HTTP error if invalid |
|---------------|-----------|----------------------|
| `StartProgress()` | `pending → in_progress` | `422 Unprocessable Entity` |
| `MarkDone()` | `in_progress → done` | `422 Unprocessable Entity` |

The state machine lives in `core/domain/flowdo.go` — the only place that knows about valid transitions.

---

## Setup

### Prerequisites

- Go 1.22+
- Docker + Docker Compose
- [`golang-migrate`](https://github.com/golang-migrate/migrate): `brew install golang-migrate`
- [`swag`](https://github.com/swaggo/swag): `go install github.com/swaggo/swag/cmd/swag@latest`
- [`mockery`](https://github.com/vektra/mockery): `go install github.com/vektra/mockery/v2@latest`

### 1. Clone and configure

```bash
git clone <repo>
cd flowdo-api
cp .env.example .env
# Edit .env — set JWT_SECRET and DATABASE_DSN at minimum
```

### 2. Start infrastructure

```bash
make docker-up      # starts Postgres + Redis via Docker Compose
make migrate-up     # applies SQL migrations (creates users + flowdos tables)
```

### 3. Run the API

```bash
make run
# Server starts at :8080
```

### 4. View Swagger UI

Open [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

```bash
make swagger   # regenerate after editing handler annotations
```

---

## Testing

```bash
# Unit tests — fast, no Docker, all business logic covered
make test-unit

# Integration tests — spins up real Postgres via testcontainers (Docker required)
make test-integration

# All tests
make test
```

### Test structure

| Package | Type | Coverage target |
|---------|------|----------------|
| `internal/core/service/` | Unit (mocked repos) | 100% business logic |
| `internal/adapter/outbound/gormrepo/` | Integration (testcontainers) | GORM adapter |

---

## Code Generation

```bash
make mock     # regenerate testify mocks with mockery (after changing port interfaces)
make swagger  # regenerate Swagger docs (after changing handler annotations)
```

---

## Makefile Reference

| Target | Description |
|--------|-------------|
| `make run` | Run the API server |
| `make build` | Build a static binary to `./bin/` |
| `make test` | Run all tests |
| `make test-unit` | Unit tests only (no Docker needed) |
| `make test-integration` | Integration tests (Docker required) |
| `make mock` | Regenerate mocks with mockery |
| `make swagger` | Regenerate Swagger docs |
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-create name=foo` | Create a new migration pair |
| `make lint` | Run golangci-lint |
| `make vet` | Run go vet |
| `make docker-up` | Start all services with Docker Compose |
| `make docker-down` | Stop and remove containers + volumes |

---

## Tech Stack

| Concern | Library |
|---------|---------|
| HTTP router | [Gin](https://github.com/gin-gonic/gin) v1.10 |
| ORM adapter | [GORM](https://gorm.io) v1.31 + [gorm.io/driver/postgres](https://github.com/go-gorm/postgres) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) v4 (sqlx connection) |
| Caching | [go-redis/v9](https://github.com/redis/go-redis) v9.5 |
| Auth | [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) v5.2 |
| Logging | [zerolog](https://github.com/rs/zerolog) v1.33 |
| Config | [godotenv](https://github.com/joho/godotenv) v1.5 |
| Mocking | [testify/mock](https://github.com/stretchr/testify) + [mockery](https://github.com/vektra/mockery) |
| Integration tests | [testcontainers-go](https://golang.testcontainers.org/) v0.31 |
| Swagger | [swaggo/swag](https://github.com/swaggo/swag) v1.16 |
| Password hashing | [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) |

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run server
go run ./cmd/api

# Seed DB (roles, permissions, admin user, system_configs) — idempotent
go run ./cmd/seed

# Test
go test ./...                           # all tests (unit, no DB needed)
go test ./internal/auth/...             # single package
go test -run TestLogin ./internal/auth/ # single test

# Build
go build ./...
go build -o parkieee-api ./cmd/api

# Vet
go vet ./...
```

Default admin credentials after seeding: `admin` / `admin123`

## .env

All required variables are in `.env` (already present). The app panics at startup if any `mustGetEnv` variable is unset. Key variables:

| Var | Notes |
|---|---|
| `DATABASE_URL` | Full Postgres DSN |
| `JWT_SECRET` | Signs user access tokens (HS256) |
| `GATE_JWT_SECRET` | Signs gate JWT (future — currently stored as column) |
| `LPR_SERVICE_URL` | URL to Python FastAPI LPR service |
| `PORT` | Default `8000` |
| `LOG_FORMAT` | `text` (dev) or `json` (prod) |

## Architecture

### Module pattern (`internal/<module>/`)

Every domain module follows this four-file layout (see `internal/auth/` as the reference):

| File | Role |
|---|---|
| `model.go` | GORM models + `AllModels() []any` — passed to `database.Migrate()` |
| `repository.go` | Concrete DB queries; implements the `Repo` interface |
| `service.go` | Business logic; depends on the `Repo` **interface**, not the concrete type |
| `handler.go` + `routes.go` | Fiber handlers + `RegisterRoutes(router, handler)` |

The `Repo` interface in `service.go` is intentional — it lets unit tests inject a `mockRepo` (testify/mock) without a real DB.

### Shared packages (`pkg/`)

| Package | What it does |
|---|---|
| `pkg/database` | `Connect(dsn)` + `Migrate(db, models...)` — thin GORM wrappers |
| `pkg/logger` | Global `slog.Logger`. Call `logger.Init()` once at startup, `logger.Get()` everywhere. `FromContext()` attaches `request_id`/`user_id`. |
| `pkg/middleware` | `AuthMiddleware` (JWT → Locals), `GateAuthMiddleware` (gate_token DB lookup), `RequirePermission("node")`, `RateLimiter`, `RequestID`, `RequestLogger` |
| `pkg/event` | In-process pub/sub `Bus` (buffered channel). `Publish` never blocks — drops on full. Background workers subscribe via `bus.Subscribe()`. |
| `pkg/ratelimit` | In-memory token-bucket keyed by any string (IP, gate_id, …) |
| `pkg/pubsub` | WebSocket broadcast hub |
| `pkg/lock` | Per-key mutex |
| `pkg/storage` | S3 client with manual AWS Signature V4 (no SDK — NevaObjects/Ceph rejects boto3 extra headers) |
| `pkg/midtrans` | Midtrans payment gateway client |

### Request flow

```
Fiber
 → RequestID → RequestLogger
 → RateLimiter (10 req/min per IP on /api/v1)
 → AuthMiddleware       sets Locals: user_id, role, permissions[], session_id
 → RequirePermission    checks permissions[] slice
 → Handler → Service → Repository → PostgreSQL (GORM)
```

### Auth & RBAC

- **Access token**: short-lived JWT (`JWT_EXP_MINUTES`, default 15 min). Claims: `user_id`, `role`, `permissions[]`, `session_id`.
- **Refresh token**: 7-day, stored as SHA-256 hash. Rotated on every `/auth/refresh` (`ReplacedBy` chain in DB).
- **Gate auth**: `GateAuthMiddleware` looks up `gate_token` column in `gates` table per-request.
- **Permission nodes**: `transaction:write/read`, `report:read`, `user:manage`, `fee:write/read`, `zone:write/read`, `override:write/read`, `audit:read`, `notification:read`.

### API response envelope

All endpoints use this shape (helpers `okResponse`/`errResponse` in `handler.go` — copy for new modules):

```json
{ "success": true,  "data": {...}, "meta": {}, "error": null }
{ "success": false, "data": null,  "meta": {}, "error": {"code": "ERR_CODE", "message": "..."} }
```

### Background workers

Started in `main()`, consume from `event.Bus`:
- `StartAuditWriter` — handles `AuditLogEvent` (stub, DB write pending)
- `StartNotificationWriter` — handles `NotificationEvent` (stub, DB write pending)

Emit events via `bus.Publish(event.Event{Type: "AuditLogEvent", Payload: ...})`.

## Current State

- **Auth module** fully implemented: login, refresh, logout, RBAC middleware.
- **Background workers** are stubs — event consumption is logged but not persisted.
- **Other modules** (vehicle, zone, fee, transaction, payment, OCR, etc.) not yet implemented — schema is in `../api_database_plan.md`.
- **Seeder** (`cmd/seed`) bootstraps: roles (`admin`, `petugas`, `owner`), permissions, default admin user, `system_configs` table.

## Adding a New Module

1. Create `internal/<module>/model.go` — define GORM models, export `AllModels()`.
2. Add models to `database.Migrate(db, ...)` call in `cmd/api/main.go`.
3. Create `repository.go` (implements interface), `service.go` (uses interface), `handler.go` + `routes.go`.
4. Call `<module>.RegisterRoutes(v1, handler)` in `main.go`.
5. Apply `middleware.AuthMiddleware(cfg.JWTSecret)` and `RequirePermission("node")` on protected routes.

<!-- code-review-graph MCP tools -->
## MCP Tools: code-review-graph

**IMPORTANT: This project has a knowledge graph. ALWAYS use the
code-review-graph MCP tools BEFORE using Grep/Glob/Read to explore
the codebase.** The graph is faster, cheaper (fewer tokens), and gives
you structural context (callers, dependents, test coverage) that file
scanning cannot.

### When to use graph tools FIRST

- **Exploring code**: `semantic_search_nodes` or `query_graph` instead of Grep
- **Understanding impact**: `get_impact_radius` instead of manually tracing imports
- **Code review**: `detect_changes` + `get_review_context` instead of reading entire files
- **Finding relationships**: `query_graph` with callers_of/callees_of/imports_of/tests_for
- **Architecture questions**: `get_architecture_overview` + `list_communities`

Fall back to Grep/Glob/Read **only** when the graph doesn't cover what you need.

### Key Tools

| Tool | Use when |
|------|----------|
| `detect_changes` | Reviewing code changes — gives risk-scored analysis |
| `get_review_context` | Need source snippets for review — token-efficient |
| `get_impact_radius` | Understanding blast radius of a change |
| `get_affected_flows` | Finding which execution paths are impacted |
| `query_graph` | Tracing callers, callees, imports, tests, dependencies |
| `semantic_search_nodes` | Finding functions/classes by name or keyword |
| `get_architecture_overview` | Understanding high-level codebase structure |
| `refactor_tool` | Planning renames, finding dead code |

### Workflow

1. The graph auto-updates on file changes (via hooks).
2. Use `detect_changes` for code review.
3. Use `get_affected_flows` to understand impact.
4. Use `query_graph` pattern="tests_for" to check coverage.

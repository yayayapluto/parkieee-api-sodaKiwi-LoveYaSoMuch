# parkieee-api

Backend service for Parkieee parking management.

## Tech Stack

- Go 1.24+
- Fiber v2
- GORM + PostgreSQL
- JWT auth + RBAC permissions
- Midtrans integration
- S3-compatible object storage
- SSE for real-time events

## Current Scope

Implemented modules:

- Auth
- User
- Vehicle + RFID
- Zone + Gate pairing
- Fee configuration
- OCR review flow
- Transaction (entry/exit + photo upload)
- Payment (cash, QRIS, refund, Midtrans callback)
- Notification
- Override
- Audit
- Report
- SSE streams
- UI dashboard views
- Background workers (outbox, unclosed checker, view refresher, optional Telegram bot)

## API Base URLs

- Public health: `http://localhost:8000/health`
- API v1: `http://localhost:8000/api/v1`

## Quick Start (Local)

1. Prepare `.env` with required variables.
2. Start infra:

```bash
make infra-up
```

3. Seed base data (idempotent):

```bash
make seed
```

4. Run API (local process):

```bash
make run
```

Or run API inside compose stack:

```bash
make run-dev
```

## Default Seeded Admin

- Username: `admin`
- Password: `admin123`

## Required Environment Variables

Required at startup (panic if missing):

- `DATABASE_URL`
- `JWT_SECRET`
- `GATE_JWT_SECRET`
- `S3_ENDPOINT`
- `S3_BUCKET`
- `S3_ACCESS_KEY`
- `S3_SECRET_KEY`
- `S3_PUBLIC_BASE_URL`
- `MIDTRANS_SERVER_KEY`
- `MIDTRANS_CLIENT_KEY`
- `LPR_SERVICE_URL`

Common optional vars:

- `PORT` (default: `8000`)
- `JWT_EXP_MINUTES` (default: `15`)
- `CORS_ALLOWED_ORIGINS`
- `COOKIE_DOMAIN`, `COOKIE_SECURE`, `COOKIE_SAME_SITE`
- `USE_CSRF` (default: `true`)
- `MIDTRANS_ENV` (default: `sandbox`)
- `LOG_LEVEL` (default: `info`)
- `LOG_FORMAT` (default: `text`)
- `TELEGRAM_BOT_TOKEN`, `TELEGRAM_ADMIN_IDS`

## Docker Compose Infra

`docker-compose.infra.yml` includes:

- `postgres`
- `api`
- `cloudflared`

Useful commands:

```bash
make infra-up
make infra-rebuild
make infra-logs
make infra-down
```

## Route Summary

Auth:

- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `GET /auth/me`

User:

- `GET /users`
- `POST /users`
- `GET /users/:id`
- `PUT /users/:id`
- `PATCH /users/:id/deactivate`
- `PATCH /users/:id/reset-password`

Vehicle + RFID:

- `GET/POST/PUT/DELETE /vehicle-types...`
- `GET/POST/PUT /vehicles...`
- `GET/POST /vehicles/:id/rfid-cards`
- `PATCH /rfid-cards/:id/deactivate`

Zone + Gate:

- `GET/POST/PUT /zones...`
- `GET/POST/PUT /gates...`
- `GET /gates/:id/devices`
- `PATCH /gate-devices/:id/status`
- `POST /gates/pair/request`
- `POST /gates/pair/confirm`
- `GET /gates/pair/verify/:code`

Fee:

- `GET/POST/PUT/DELETE /fee-configs...`
- `GET/POST/PUT/DELETE /fee-configs/:id/tiers...`
- `GET/POST/PUT/DELETE /holiday-rates...`
- `GET /system-configs`
- `PUT /system-configs/:key`

OCR:

- `GET /ocr/jobs`
- `POST /ocr/jobs/:id/review`

Transaction:

- `POST /transactions/entry`
- `POST /transactions/exit`
- `POST /uploads/entry-photo`
- `POST /uploads/exit-photo`
- `GET /transactions`
- `GET /transactions/:id`
- `GET /transactions/:id/logs`

Payment:

- `POST /payments/cash`
- `POST /payments/qris`
- `POST /payments/refunds`
- `POST /payments/midtrans/callback`
- `GET /payments/:transaction_id`
- `GET /payments/:transaction_id/refunds`

Other:

- Notification: unread/list/mark read
- Override: `POST /overrides/:id/override`
- Audit: `GET /audit`
- Report: revenue, occupancy, export
- SSE: `/sse/gate/:id`, `/sse/cashier`

UI pages:

- `/ui/dashboard`
- `/ui/transactions`
- `/ui/vehicles`
- `/ui/zones`
- `/ui/fees`
- `/ui/users`

## Quality Commands

```bash
make test
make test-race
make vet
make fmt
make build
make check
```

## API Response Envelope

Success:

```json
{ "success": true, "data": {}, "meta": {}, "error": null }
```

Error:

```json
{
  "success": false,
  "data": null,
  "meta": {},
  "error": { "code": "ERR_CODE", "message": "message" }
}
```

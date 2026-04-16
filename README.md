# parkieee-api

Go backend for parkieee parking management system.

## Stack

- Go + Fiber
- GORM + PostgreSQL
- JWT auth + RBAC middleware

## Quick Start

1. Make sure required environment variables are available in `.env`.
2. Run database seed (idempotent):

```bash
go run ./cmd/seed
```

3. Start API server:

```bash
go run ./cmd/api
```

4. Health check:

```bash
GET http://localhost:8000/health
```

## Useful Commands

```bash
# Run all tests
go test ./...

# Vet
go vet ./...

# Build
go build -o parkieee-api ./cmd/api
```

## Default Seed Account

- Username: `admin`
- Password: `admin123`

## API Base URL

`http://localhost:8000/api/v1`

## Features Implemented

### 1. Auth

- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

### 2. User Management (permission: `user:manage`)

- `GET /users`
- `POST /users`
- `GET /users/:id`
- `PUT /users/:id`
- `PATCH /users/:id/deactivate`
- `PATCH /users/:id/reset-password`

### 3. Vehicle + RFID Management

Read permission: `zone:read`
Write permission: `zone:write`

Vehicle types:
- `GET /vehicle-types`
- `POST /vehicle-types`
- `PUT /vehicle-types/:id`
- `DELETE /vehicle-types/:id`

Vehicles:
- `GET /vehicles`
- `POST /vehicles`
- `GET /vehicles/:id`
- `PUT /vehicles/:id`

RFID:
- `GET /vehicles/:id/rfid-cards`
- `POST /vehicles/:id/rfid-cards`
- `PATCH /rfid-cards/:id/deactivate`

## Not Yet Implemented

- Zone routes (currently stub)
- Fee routes (currently stub)
- Core phase 2 modules: transaction, payment, OCR, SSE flow

## Postman Testing Flow

1. Login admin

Request:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

Save:
- `data.access_token` -> `access_token`
- `data.refresh_token` -> `refresh_token`

2. Set authorization for protected endpoints

- Header: `Authorization: Bearer {{access_token}}`

3. Try user endpoint

```http
GET /api/v1/users?page=1&limit=20
```

4. Create vehicle type

```http
POST /api/v1/vehicle-types
Content-Type: application/json
Authorization: Bearer {{access_token}}

{
  "name": "Mobil",
  "minimum_fee": 5000,
  "description": "Kendaraan roda 4"
}
```

5. Create vehicle

```http
POST /api/v1/vehicles
Content-Type: application/json
Authorization: Bearer {{access_token}}

{
  "plate_number": "B1234XYZ",
  "vehicle_type_id": "<vehicle_type_uuid>",
  "source": "siswa",
  "notes": "testing postman"
}
```

6. Assign RFID

```http
POST /api/v1/vehicles/<vehicle_id>/rfid-cards
Content-Type: application/json
Authorization: Bearer {{access_token}}

{
  "card_uid": "RFID-001"
}
```

## Notes

- API uses a standard response envelope:
  - success: `{ success, data, meta, error }`
  - error: `{ success, data: null, meta, error: { code, message } }`
- Some endpoints enforce role/permission checks and will return forbidden if permissions are missing.

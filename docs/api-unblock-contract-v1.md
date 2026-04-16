# API Unblock Contract v1

Date: 2026-04-16
Status: Active for FE unblock implementation

## 1. Transaction Method Enum

Canonical values for entry/exit payload method:
- rfid
- ticket

Notes:
- `manual` and `lpr` are not accepted by transaction service in v1.
- FE should map UI labels (Manual/LPR) to supported backend flow until API method expansion is shipped.

## 2. Gate Authentication

Canonical gate auth in v1:
- `gate_token` persisted in `gates.gate_token`
- Read by gate middleware from query param `gate_token` or bearer token

Pairing flow returns `gate_token` after confirmation.

## 3. Response Envelope

All endpoints use:
- success: bool
- data: any
- meta: object
- error: object|null

## 4. SSE Contract (Final)

SSE `data:` payload is typed JSON object:

```json
{
	"type": "gate.barrier.open",
	"timestamp": "2026-04-16T10:00:00Z",
	"payload": {
		"gate_id": "...",
		"transaction_id": "...",
		"source": "payment"
	}
}
```

Current event types:
- `gate.barrier.open`
- `cashier.plate_mismatch`

## 5. Newly Exposed Unblock Endpoints

- Zones/Gates/Pairing
- OCR jobs list
- Transaction logs list
- Upload entry/exit photo
- Notification unread count
- Refund create/list

See module routes for exact paths.

## 6. Compatibility Rule

Clients should treat unknown `type` as non-fatal and ignore unsupported payload fields.

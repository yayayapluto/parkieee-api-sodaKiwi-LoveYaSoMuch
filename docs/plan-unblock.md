## Plan: API-FE Unblock Sprint

Deliver minimum backend scope so frontend can wire real endpoints without stubs, while freezing contracts to avoid churn.

**Steps**
1. Freeze contracts first: finalize transaction method enum (`rfid+ticket` or `rfid+lpr+manual`), gate auth model (`gate_token` or gate JWT), and SSE v1 payload shape (`type`, `timestamp`, IDs, payload).
2. Publish API contract pack: OpenAPI update, error code matrix, and sample request/response for each new endpoint. *depends on step 1*
3. Implement missing zone/gate surface in zone module: zones CRUD, gates CRUD, gate-devices status endpoints, and route registration in API bootstrap. *depends on step 1*
4. Implement gate pairing flow: request code, admin confirm, device verify, expiry handling, one-time use semantics. *depends on step 3*
5. Implement media upload flow for kiosk photos: either direct upload endpoint or pre-signed upload issuance; align payload used by transaction entry/exit APIs. *parallel with step 4 after step 1*
6. Implement OCR jobs listing endpoint (`pending` filter, pagination) for admin review queue. *parallel with step 5*
7. Implement transaction logs endpoint for timeline view (`GET /transactions/:id/logs`) with pagination and stable ordering. *parallel with step 6*
8. Implement refunds API surface (create refund, list by transaction, status model), including permission checks and audit emission. *parallel with step 7*
9. Implement unread notification count endpoint or document canonical derived strategy from list meta; pick one and freeze. *parallel with step 8*
10. Add/update tests for all new handlers/services/repositories and permission gates; run full test suite. *depends on steps 3-9*
11. Run frontend staging handshake: FE wires real endpoints, backend resolves any contract mismatches same day, then lock v1 contract tag. *depends on step 10*

**Relevant files**
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/zone/routes.go` — currently stub; add zone/gate routes
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/zone/handler.go` — zone/gate handlers
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/zone/service.go` — pairing + business rules
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/zone/repository.go` — pairing + gate persistence
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/ocr/routes.go` — add OCR list endpoint
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/ocr/handler.go` — list endpoint handler
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/transaction/routes.go` — add logs route
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/transaction/handler.go` — logs handler
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/payment/routes.go` — refund routes
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/payment/handler.go` — refund handlers
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/notification/routes.go` — unread count route if added
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/cmd/api/main.go` — register new route groups

**Verification**
1. Run `go test ./...` and ensure all tests pass.
2. Add integration tests for critical flows: pairing, entry photo upload path, refund state transitions, OCR pending list.
3. Verify RBAC gates by role (`admin`, `petugas`, `owner`) for each new endpoint.
4. Validate FE smoke on staging: admin OCR queue opens, kiosk photo path accepted, transaction timeline loads, refund action works.
5. Confirm SSE payload examples consumed without custom parsing hacks in FE.

**Decisions**
- Scope includes only endpoints/contract needed to unblock FE roadmap phases; deep hardening deferred.
- Keep response envelope uniform (`success`, `data`, `meta`, `error`) for every new endpoint.
- If unread-count endpoint not added, list endpoint must expose consistent meta for FE badge.

**Further Considerations**
1. Prefer upload endpoint vs pre-signed URL: upload endpoint simpler FE; pre-signed scales better. Recommendation: pre-signed if object store policy ready, else endpoint now + migrate later.
2. Refund domain ownership: keep under payment module for now to avoid cross-module churn; revisit after v1.
3. Gate auth migration path: if moving to JWT, keep temporary compatibility window for existing gate_token devices.

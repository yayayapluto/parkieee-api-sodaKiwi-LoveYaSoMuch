# parkieee-web Frontend Implementation Plan

> Generated 2026-04-16. Depends on parkieee-api (Go Fiber) at `../parkieee-api/`.

---

## 0. Context

API `v1` base: `http://<host>:8000/api/v1`. Response envelope: `{success, data, meta, error}`. Cookie-based session (JWT httpOnly + refresh), CSRF via `X-CSRF-Token` + cookie read. SSE on `/sse/gate/:id` (gate JWT) + `/sse/cashier` (user JWT).

Roles to panels:

| Role | Panel |
|---|---|
| `petugas` | Operator (cashier) + Kiosk gate-in/out terminal |
| `admin` | Admin |
| `owner` | Monitoring |

Kiosk = petugas-authenticated dedicated terminal UI (touch-friendly). Same role, different shell.

---

## 1. Architecture Additions

### 1.1 Routing

Add route groups:

```
routes/
  _admin-layout/admin/           (exists)
  _petugas-layout/petugas/       (operator)
  _owner-layout/owner/           (monitoring)
  _kiosk-layout.tsx              NEW — petugas role, fullscreen shell, no nav
  _kiosk-layout/kiosk/
    in.tsx                       Entry kiosk
    out.tsx                      Exit kiosk
    pair.tsx                     Gate pairing
```

`_kiosk-layout` guards: role=`petugas` + has `transaction:write`. Auto-fullscreen, disable F5/right-click, long idle auto-lock.

### 1.2 API Client Additions

Files per module under `apps/web/src/api/`:

- `users.api.ts`, `vehicles.api.ts`, `vehicle-types.api.ts`, `rfid.api.ts`
- `transactions.api.ts`, `payments.api.ts`, `ocr.api.ts`, `overrides.api.ts`
- `reports.api.ts`, `notifications.api.ts`, `audit.api.ts`, `system-configs.api.ts`
- `zones.api.ts` (blocked — see §6)

`lib/api-client.ts` already handles 401 refresh retry + CSRF. Extend:

- Add `unwrap<T>()` helper — throw on `!success`, use `error.code` for i18n.
- Add `ApiError` class carrying `code`+`message`. Replace raw `HTTP_401`.
- Paginate helper: `{page, limit}` to `URLSearchParams`.

### 1.3 Realtime (SSE)

`lib/sse.ts` — `createEventSource(path, onEvent, onError)`. Uses `EventSource` w/ `withCredentials: true`. Reconnect w/ exponential backoff (1s to 30s cap).

Hooks:

- `useCashierSSE()` — subscribes `/sse/cashier`, pushes into TanStack Query cache via `queryClient.setQueryData`.
- `useGateSSE(gateId)` — kiosk only, listens entry/exit confirm + OCR results.

Events schema TBD on API — assume `{type, payload}`. Invalidate `["transactions"]`, `["notifications"]`, `["ocr", "pending"]` on matching types.

### 1.4 State/Data

- TanStack Query: default `staleTime: 30s`, `refetchOnWindowFocus: true` (disable on kiosk).
- Mutations: optimistic on notifs mark-read only. Rest = server-truth.
- Query keys (hierarchy):
  - `["auth","me"]`
  - `["users", params?]`, `["users", id]`
  - `["vehicles", params?]`, `["vehicle", id]`
  - `["transactions", params?]`, `["transaction", id]`
  - `["payments", txId]`
  - `["reports","revenue", params]`, `["reports","occupancy", params]`
  - `["notifications"]`, `["audit-logs", params]`
  - `["system-configs"]`

### 1.5 Forms

All forms: `react-hook-form` + `zod` schema in `types/*.schema.ts` (zod shares types via `z.infer`). Submit to mutation to toast via sonner.

### 1.6 Env

Add to `packages/env/src/web.ts`:

- `VITE_API_BASE_URL` (exists)
- `VITE_SSE_HEARTBEAT_MS`
- `VITE_KIOSK_IDLE_LOCK_MS`
- `VITE_MIDTRANS_CLIENT_KEY` (if using Snap.js client-side)

---

## 2. Kiosk Panel

### 2.1 Entry Kiosk — `/kiosk/in`

Purpose: petugas on duty at gate-in, captures vehicle entry. Large buttons, minimal text.

Flow:

```
Idle screen (zone occupancy + clock)
  -> vehicle approach
  -> petugas picks method: RFID tap OR LPR snapshot OR manual ticket
  -> POST /transactions/entry {gate_id, method: "rfid"|"lpr"|"manual", rfid_card_uid?, photo_path}
  -> show confirm screen (plate, zone, ticket_code) 5s
  -> print ticket (if manual/lpr) via browser print or desktop IPC
  -> return idle
```

Screens/components:

- `KioskIdleScreen` — clock, zone live count from SSE, "Tap RFID or Press Start".
- `EntryMethodPicker` — 3 big buttons (RFID / LPR / Manual).
- `LPRCaptureView` — camera preview via `getUserMedia`, "Capture" button, upload then POST entry w/ `photo_path` returned by upload endpoint (TBD — need `/uploads` or pre-signed S3 URL from API; see §6).
- `RFIDWaitView` — listens for card reader input (HID keyboard emu or WebSerial). 10s timeout then cancel.
- `EntryConfirmScreen` — plate, ticket code, QR/barcode (generated client-side from `ticket_code`), auto-dismiss 5s.
- `EntryErrorScreen` — `DOUBLE_ENTRY` / `RFID_NOT_FOUND` messaging.

Hooks:

- `useEntryMutation()` — `POST /transactions/entry`, invalidates `["transactions"]`.
- `useRfidReader()` — subscribes to HID events (listen for keydown sequences ending `Enter`).
- `useCamera()` — manages stream + capture canvas.

### 2.2 Exit Kiosk — `/kiosk/out`

Flow:

```
Idle -> RFID tap OR scan ticket QR OR manual plate search
  -> POST /transactions/exit {gate_id, method, rfid_card_uid?, ticket_code?, photo_path}
  -> show fee + duration
  -> if paid already (prepaid RFID): open gate event -> confirm screen
  -> else: payment view (cash / QRIS)
     - Cash: POST /payments/cash {transaction_id, amount} -> receipt -> open gate
     - QRIS: POST /payments/qris {transaction_id} -> display QR -> poll GET /payments/:tx_id OR wait SSE "payment.settled" -> open gate
  -> idle
```

Screens:

- `ExitMethodPicker`
- `ExitSummary` — plate, entry time, duration, fee breakdown.
- `PaymentChoice` — cash / QRIS buttons.
- `CashPaymentView` — amount tendered input, change calc (server returns `change_due` — do NOT compute frontend per rule #2).
- `QRISPaymentView` — render QR from `qr_code` field, poll/SSE for settlement, 2min timeout then retry/cancel.
- `ExitConfirmScreen` — "Terima kasih" + plate + gate number.
- Plate-mismatch banner if `plate_mismatch=true` — require petugas supervisor PIN before opening gate.

### 2.3 Gate Pairing — `/kiosk/pair`

API endpoint missing — see §6. Placeholder UI: show 6-char code, poll for admin confirmation. Blocked.

### 2.4 Kiosk Hardening

- Disable right-click, F5, Ctrl-R, browser chrome (Electrobun/fullscreen).
- Service worker cache static assets — tolerate brief API offline. Queue entry requests in IndexedDB, replay on reconnect (with idempotency key).
- Heartbeat: `useOnlineStatus()` polls `/health` every 10s; banner when offline.
- Idle timeout: auto-logout after `VITE_KIOSK_IDLE_LOCK_MS`.

---

## 3. Admin Panel

Role `admin`. Full CRUD. Sidebar nav.

### 3.1 Routes

```
_admin-layout/admin/
  dashboard.tsx              (exists, stub)
  users/
    index.tsx                list + create
    $userId.tsx              detail + edit + deactivate + reset-pw
  vehicles/
    index.tsx
    $vehicleId.tsx           detail + RFID mgmt
  vehicle-types/
    index.tsx                CRUD
  zones/                     BLOCKED §6
  gates/                     BLOCKED §6
  fee-configs/               BLOCKED §6 (module not in routes yet, only system-configs)
  system-configs/
    index.tsx                key-value editor
  overrides/
    $transactionId.tsx       apply override form
  audit-logs/
    index.tsx                filter + pagination + export
  notifications/
    index.tsx                (shared w/ header bell)
  ocr-review/
    index.tsx                pending OCR jobs queue
    $jobId.tsx               review screen
```

### 3.2 Pages Detail

**Users** (`/admin/users`)

- List: pagination, filter by role/active. Columns: username, role, last_login, status.
- Create: username, password, role (admin/petugas/owner), permissions inherited from role.
- Detail: edit role, deactivate (`PATCH /:id/deactivate`), reset password (`PATCH /:id/reset-password` returns temp password in toast — copy button).

**Vehicles** (`/admin/vehicles`)

- List: plate, type, owner, active RFID count.
- Create form: plate, vehicle_type_id, owner_name.
- Detail: RFID card list + assign (`POST /vehicles/:id/rfid-cards`) + deactivate (`PATCH /rfid-cards/:id/deactivate`).

**Vehicle Types** (`/admin/vehicle-types`)

- Simple CRUD table (modal edit).

**System Configs** (`/admin/system-configs`)

- `GET /system-configs` list. Inline edit via `PUT /system-configs/:key`.

**OCR Review** (`/admin/ocr-review`)

- List pending jobs from transactions list filtered by `plate_mismatch=true`. (No dedicated listing endpoint — see §6, may need.)
- Detail: side-by-side entry photo + LPR-detected plate + vehicle plate; approve/reject via `POST /ocr/jobs/:id/review`.

**Overrides** (`/admin/transactions/:id/override`)

- Form: waive_fee | manual_close | adjust_fee. Reason textarea required. POST override.

**Audit Logs** (`/admin/audit-logs`)

- Filter: actor, action, date range. Paginated. No export endpoint exposed yet.

**Notifications** — shared header dropdown + dedicated page. Mark read/read-all.

---

## 4. Operator (Petugas) Panel

Desktop-browser counterpart to kiosk. Cashier workstation — assists multiple gates, handles exceptions, issues refunds, manual closeout.

### 4.1 Routes

```
_petugas-layout/petugas/
  dashboard.tsx              (exists, stub)
  live.tsx                   Live gate feed (SSE cashier channel)
  transactions/
    index.tsx                active + historical
    $transactionId.tsx       detail + actions
  payments/
    $transactionId.tsx       take payment manually
  notifications/
    index.tsx
```

### 4.2 Live Feed (`/petugas/live`)

- Full-height column layout: "Entries", "Exits Pending Payment", "Alerts".
- Consumes `useCashierSSE()`. Cards appear as events arrive.
- Click entry card opens transaction detail in right drawer.
- Alert types: `DOUBLE_ENTRY`, `PLATE_MISMATCH`, `UNCLOSED_FLAG` — highlighted red.

### 4.3 Transactions Detail

- Summary: plate, entry/exit gate, times, duration, fee.
- Log timeline: `transaction_logs[]` (needs backend — see §6).
- Actions: take payment (cash/QRIS), apply override (if perm), print receipt.
- Plate-mismatch banner with OCR review shortcut.

### 4.4 Payment Flow (manual)

Same as kiosk exit payment but initiated from transaction detail. Useful when kiosk unavailable or ticket lost.

---

## 5. Monitoring Panel (Owner)

Read-only. Charts + tables. Permission: `report:read`.

### 5.1 Routes

```
_owner-layout/owner/
  dashboard.tsx              overview KPIs
  revenue.tsx                revenue report + export CSV
  occupancy.tsx              occupancy report
  audit.tsx                  audit log view (if owner has audit:read)
```

### 5.2 Dashboard KPIs

- Today revenue (from `/reports/revenue?start=today&end=now`)
- Today transaction count
- Current occupancy per zone (from `/reports/occupancy`)
- Trend chart: revenue last 7/30 days (API provides aggregated — no frontend reducing).

### 5.3 Revenue (`/owner/revenue`)

- Date range picker, zone filter, vehicle type filter.
- Table + chart from `/reports/revenue`.
- Export button: `GET /reports/revenue/export?...` triggers download via `<a href>` with blob.

### 5.4 Occupancy (`/owner/occupancy`)

- Live occupancy per zone. Historical timeline from `/reports/occupancy` (API returns aggregated bucketed data).
- SSE subscribe optional for live count (if API emits `zone.capacity` events).

---

## 6. Gaps / Blockers

Require backend work before UI wiring:

| # | Gap | Owner |
|---|---|---|
| 1 | Zone module endpoints empty (`zone/routes.go` stub) — no GET/POST zones, gates, gate_devices | API |
| 2 | Fee configs CRUD not exposed (only `/system-configs`) — no `/fee-configs`, `/fee-tiers`, `/holiday-rates` routes | API |
| 3 | Gate pairing flow endpoints missing — `/gates/pair/request`, `/gates/pair/confirm`, `/gates/pair/verify` | API |
| 4 | File upload endpoint for kiosk photos — need `/uploads/entry-photo` returning S3 key path, OR pre-signed URL generator | API |
| 5 | OCR job listing endpoint — `GET /ocr/jobs?status=pending` not exposed | API |
| 6 | Transaction logs timeline — `GET /transactions/:id/logs` not in routes | API |
| 7 | Refunds endpoint — `refunds` table exists, no routes | API |
| 8 | SSE event schema undocumented — need `{type, payload}` contract | API |
| 9 | Notification count endpoint for bell badge — derive from list or add `GET /notifications/unread-count` | API |
| 10 | CSRF token mint endpoint — `csrf.ts` reads cookie but cookie-set endpoint unclear | verify |

Frontend fallback: build w/ stubs + feature flags. Hide nav entries when endpoint 404s.

---

## 7. Cross-Cutting

### 7.1 Layout Shell

Existing `role-layout-shell.tsx` base. Add:

- Sidebar w/ role-scoped nav (items derived from `user.permissions`).
- Header: breadcrumbs, notification bell (badge count), theme toggle, user menu (logout).
- Content area: `<Outlet />`.

Nav config file: `lib/nav.ts` with `{admin: [...], petugas: [...], owner: [...]}` w/ `requires: string[]` permission filter.

### 7.2 Permission Gating

Component: `<PermGuard perms={["transaction:write"]}>` — hides children if user lacks. Route-level: `beforeLoad` check in `_<role>-layout.tsx` already redirects unauth, extend with per-route perm check.

### 7.3 Error Boundary

`<ErrorBoundary>` at layout level with fallback UI w/ "Reload" + toast trigger. API errors: `ApiError.code` mapped via i18n map (`lib/error-messages.ts`).

### 7.4 i18n

Indonesian primary (project is SMK UKK). Lightweight — `lib/i18n.ts` just exports maps. No runtime switching needed MVP. All error codes mapped to id strings.

### 7.5 Printing

`lib/print.ts` — hidden iframe print for tickets/receipts. Template components `PrintableTicket`, `PrintableReceipt`. CSS `@media print` rules.

### 7.6 Testing

- Unit: hooks + api clients (mock `fetch`). Vitest.
- E2E: Playwright for kiosk flows (entry -> confirm, exit -> payment -> open).
- Visual regression not required.

---

## 8. Phased Roadmap

| Phase | Scope | Unblocks |
|---|---|---|
| P0 | Foundation: api-client extensions, SSE lib, nav, layout shells complete, perm guard, error boundary, i18n base | All |
| P1 | Admin users + vehicles + vehicle-types + system-configs | Core mgmt |
| P2 | Operator transactions list + detail + manual payment flow | Cashier ops |
| P3 | Kiosk entry flow (manual + RFID; LPR stub until upload endpoint ready) | Gate-in |
| P4 | Kiosk exit flow + cash payment + QRIS payment | Gate-out |
| P5 | Monitoring owner dashboard + revenue + occupancy + export | Owner |
| P6 | OCR review + overrides + audit-logs + notifications | Admin ops |
| P7 | LPR photo upload wiring + gate pairing UI + refunds | Needs API §6 |
| P8 | Offline kiosk queue + service worker + Electrobun packaging | Prod-ready |

---

## 9. Per-Feature Implementation Pattern

Follow CLAUDE.md "Pola Lengkap":

1. `types/*.types.ts` — interfaces + zod schemas
2. `api/*.api.ts` — fetch, unwrap envelope, throw `ApiError`
3. `hooks/use-*.ts` — `useQuery`/`useMutation`, invalidations
4. `components/<feature>/*.tsx` — pure view, no logic
5. `routes/.../index.tsx` — thin `createFileRoute`

No `.sort/.filter/.reduce` frontend. No `className` override on shadcn. No `any`. Loading=`Skeleton`, error=`text-destructive`. Biome clean before commit.

---

## 10. Deliverables Checklist

- [ ] 12 api client files
- [ ] 20+ hook files
- [ ] ~60 route files (thin)
- [ ] ~80 view components
- [ ] `lib/` additions: `sse.ts`, `api-error.ts`, `nav.ts`, `print.ts`, `i18n.ts`, `perm-guard.tsx`, `online-status.ts`
- [ ] Env vars added
- [ ] Playwright kiosk e2e specs
- [ ] README per panel

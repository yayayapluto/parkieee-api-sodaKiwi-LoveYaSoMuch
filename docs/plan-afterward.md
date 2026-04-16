## Plan: Post-Unblock Production Hardening

After frontend unblocked, focus on reliability, security, observability, and operations so system is production-safe under real load.

**Steps**
1. Build idempotency framework for mutation endpoints (entry, exit, cash payment, qris initiation, refund, callback processing) with replay-safe semantics and duplicate response behavior.
2. Harden payment callback processing: signature verification strictness, duplicate callback dedup, eventual consistency reconciliation job. *depends on step 1*
3. Strengthen transactional integrity: lock strategy review, race-condition tests on concurrent gate/payment requests, deterministic status transitions.
4. Upgrade outbox/worker reliability: retry with exponential backoff, dead-letter storage, replay command, and poison-event visibility.
5. Establish security hardening baseline: permission matrix tests, CSRF/session lifecycle tests, secret rotation runbook, gate credential rotation process.
6. Add observability stack for operations: metrics (latency/error/throughput/worker lag), structured logs with correlation IDs, tracing for critical request paths.
7. Define SLOs and alerting: API error-rate, callback failure spikes, SSE disconnect rates, queue lag thresholds, DB saturation indicators.
8. Run performance and soak testing: kiosk burst traffic, cashier concurrent usage, callback flood scenarios, and long-lived SSE sessions.
9. Prepare operational readiness: backup/restore drill, migration rollback drill, incident playbooks, on-call triage checklist.
10. Final release gate: UAT sign-off, docs sync (OpenAPI/Postman/runbooks), go-live checklist with rollback criteria.

**Relevant files**
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/transaction/service.go` — idempotency + transition integrity
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/payment/service.go` — callback dedup + settlement guarantees
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/payment/handler.go` — callback validation and response behavior
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/worker/unclosed_checker.go` — worker reliability patterns
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/pkg/outbox` — retry/backoff/dead-letter enhancements
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/pkg/middleware/auth.go` — auth/csrf hardening and tests
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/pkg/logger/logger.go` — structured log enrichment
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/internal/sse/handler.go` — event contract + connection metrics
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/README.md` — ops/runbook pointers
- `d:/farras/SMK_Taruna_Bhakti/Projek UKK - Parkir/parkieee-api/postman/parkieee-api.postman_collection.json` — contract sync

**Verification**
1. Run `go test ./...` plus added integration/concurrency tests; all pass.
2. Execute callback replay tests: repeated identical callback must not duplicate payment finalization.
3. Validate idempotency key behavior for each mutation endpoint.
4. Run load/soak scenarios and verify SLO thresholds + alert behavior.
5. Execute backup/restore and migration rollback drills successfully.
6. Confirm on-call runbook can resolve simulated incidents within target response window.

**Decisions**
- Post-unblock scope excludes new product features; focus is stability and operational maturity.
- Reliability changes must be backward-compatible with FE contract frozen in unblock phase.
- Alerting noise budget must be controlled before go-live.

**Further Considerations**
1. Choose metrics backend early (Prometheus stack vs managed service) to avoid rework.
2. Decide tracing depth pragmatically: sample all errors + a percentage of success paths.
3. Plan credential rotation drills quarterly once live.

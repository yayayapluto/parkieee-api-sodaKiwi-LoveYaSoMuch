-- parkieee-api/pkg/database/migrations/001_materialized_views.sql

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_revenue AS
SELECT
    DATE_TRUNC('day', t.exit_at)  AS day,
    t.zone_id,
    v.vehicle_type_id,
    COUNT(*)                       AS transaction_count,
    COALESCE(SUM(p.amount), 0)     AS total_revenue
FROM transactions t
JOIN payments p ON p.transaction_id = t.id AND p.status = 'completed'
JOIN vehicles v ON v.id = t.vehicle_id
WHERE t.status = 'completed'
GROUP BY 1, 2, 3;

CREATE UNIQUE INDEX IF NOT EXISTS mv_daily_revenue_idx
    ON mv_daily_revenue (day, zone_id, vehicle_type_id);

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_zone_occupancy AS
SELECT
    z.id        AS zone_id,
    z.name      AS zone_name,
    z.capacity,
    COUNT(t.id) AS occupied
FROM zones z
LEFT JOIN transactions t ON t.zone_id = z.id AND t.status = 'active'
GROUP BY z.id, z.name, z.capacity;

CREATE UNIQUE INDEX IF NOT EXISTS mv_zone_occupancy_idx ON mv_zone_occupancy (zone_id);

-- FTS index on audit_logs for audit log search (Phase 4 Task 2)
CREATE INDEX IF NOT EXISTS idx_audit_logs_fts
    ON audit_logs
    USING GIN (to_tsvector('english', action || ' ' || COALESCE(entity_type, '')));

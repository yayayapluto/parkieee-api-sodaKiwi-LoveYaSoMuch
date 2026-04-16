package audit

import (
	"time"

	"gorm.io/gorm"
)

type Repo interface {
	Create(log *AuditLog) error
	Search(query string, before time.Time, limit int) ([]AuditLog, error)
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(log *AuditLog) error {
	return r.db.Create(log).Error
}

// Search uses PostgreSQL full-text search on action + entity_type columns.
// Results are ordered newest-first and capped at limit rows.
func (r *Repository) Search(query string, before time.Time, limit int) ([]AuditLog, error) {
	q := r.db.Model(&AuditLog{}).Order("created_at DESC")

	if query != "" {
		q = q.Where(
			"to_tsvector('english', action || ' ' || COALESCE(entity_type, '')) @@ plainto_tsquery('english', ?)",
			query,
		)
	}
	if !before.IsZero() {
		q = q.Where("created_at < ?", before)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q = q.Limit(limit)

	var logs []AuditLog
	return logs, q.Find(&logs).Error
}

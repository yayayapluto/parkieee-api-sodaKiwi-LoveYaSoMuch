package audit

import (
	"context"
	"time"
)

type Service struct{ repo Repo }

func NewService(repo Repo) *Service { return &Service{repo: repo} }

// Search delegates to repo with no additional business rules.
// _ ctx reserved for future tracing / cancellation propagation.
func (s *Service) Search(_ context.Context, query string, before time.Time, limit int) ([]AuditLog, error) {
	return s.repo.Search(query, before, limit)
}

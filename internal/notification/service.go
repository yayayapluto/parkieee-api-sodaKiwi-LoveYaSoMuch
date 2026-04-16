package notification

import (
	"context"

	"github.com/google/uuid"
)

type Service struct{ repo Repo }

func NewService(repo Repo) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID *uuid.UUID, page, limit int) ([]Notification, int64, error) {
	_ = ctx
	return s.repo.FindByUserID(userID, page, limit)
}

func (s *Service) MarkRead(ctx context.Context, id uuid.UUID) error {
	_ = ctx
	return s.repo.MarkRead(id)
}

func (s *Service) UnreadCount(ctx context.Context, userID *uuid.UUID) (int64, error) {
	_ = ctx
	return s.repo.CountUnread(userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID *uuid.UUID) error {
	_ = ctx
	return s.repo.MarkAllRead(userID)
}

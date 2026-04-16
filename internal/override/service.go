package override

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/internal/sse"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
	"gorm.io/gorm"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidOverrideType = errors.New("override_type must be fee_adjust or barrier_release")
	ErrDailyLimitExceeded  = errors.New("daily override limit exceeded")
)

type ApplyInput struct {
	TransactionID uuid.UUID
	OperatorID    uuid.UUID
	OverrideType  string // "fee_adjust" | "barrier_release"
	Reason        string
	AdjustedFee   int
}

type Service struct {
	repo  Repo
	db    *gorm.DB
	hub   *pubsub.Hub
	now   func() time.Time
	runTx func(fn func(*gorm.DB) error) error
}

func NewService(repo Repo, db *gorm.DB, hub *pubsub.Hub) *Service {
	s := &Service{repo: repo, db: db, hub: hub, now: time.Now}
	s.runTx = s.defaultRunTx
	return s
}

func (s *Service) Apply(ctx context.Context, in ApplyInput) (*OperatorOverride, error) {
	if in.OverrideType != "fee_adjust" && in.OverrideType != "barrier_release" {
		return nil, ErrInvalidOverrideType
	}

	tx, err := s.repo.FindTransaction(in.TransactionID)
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, ErrTransactionNotFound
	}

	cfg, err := s.repo.FindActiveConfig()
	if err != nil {
		return nil, err
	}

	if cfg != nil && cfg.MaxOverridesPerDay > 0 {
		count, err := s.repo.CountTodayByOperator(in.OperatorID)
		if err != nil {
			return nil, err
		}
		if count >= int64(cfg.MaxOverridesPerDay) {
			if s.db != nil {
				// Escalation notification is best-effort; failure must not block the limit-exceeded error path.
				_ = outbox.Publish(s.db, "notify.override_escalation", map[string]any{
					"operator_id":    in.OperatorID,
					"transaction_id": in.TransactionID,
					"count":          count,
					"limit":          cfg.MaxOverridesPerDay,
				})
			}
			return nil, ErrDailyLimitExceeded
		}
	}

	rec := &OperatorOverride{
		ID:            uuid.New(),
		TransactionID: in.TransactionID,
		OperatorID:    in.OperatorID,
		OverrideType:  in.OverrideType,
		OriginalFee:   tx.CalculatedFee,
		AdjustedFee:   in.AdjustedFee,
		Reason:        in.Reason,
		CreatedAt:     s.now(),
	}
	if in.OverrideType == "barrier_release" {
		rec.AdjustedFee = tx.CalculatedFee // no fee change on barrier release
	}

	err = s.runTx(func(dbTx *gorm.DB) error {
		if err := s.repo.Create(dbTx, rec); err != nil {
			return err
		}
		if in.OverrideType == "fee_adjust" && in.AdjustedFee != tx.CalculatedFee {
			return s.repo.UpdateTransactionFee(dbTx, in.TransactionID, in.AdjustedFee)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if in.OverrideType == "barrier_release" && s.hub != nil && tx.ExitGateID != nil {
		s.hub.Publish("gate:"+tx.ExitGateID.String(), sse.NewEventMessage("gate.barrier.open", map[string]any{
			"gate_id":        *tx.ExitGateID,
			"transaction_id": tx.ID,
			"source":         "override",
		}))
	}

	// TODO: thread ctx through to repo and outbox calls once context propagation is wired
	_ = ctx
	return rec, nil
}

func (s *Service) defaultRunTx(fn func(*gorm.DB) error) error {
	if s.db == nil {
		return fn(nil)
	}
	return s.db.Transaction(fn)
}

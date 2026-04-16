package payment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/internal/sse"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"github.com/yyypluto/parkieee-api/pkg/midtrans"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
	"gorm.io/gorm"
)

type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Message }

var (
	ErrTransactionNotFound    = &AppError{Code: "TRANSACTION_NOT_FOUND", Message: "transaction not found"}
	ErrInvalidStatus          = &AppError{Code: "INVALID_TRANSACTION_STATUS", Message: "transaction must be pending payment"}
	ErrPaymentNotFound        = &AppError{Code: "PAYMENT_NOT_FOUND", Message: "payment not found"}
	ErrInsufficientCash       = &AppError{Code: "INSUFFICIENT_CASH", Message: "cash tendered is below required fee"}
	ErrPaymentGateway         = &AppError{Code: "PAYMENT_GATEWAY_ERROR", Message: "payment gateway request failed"}
	ErrPaymentExists          = &AppError{Code: "PAYMENT_EXISTS", Message: "payment already exists"}
	ErrInvalidCallbackPayload = &AppError{Code: "INVALID_CALLBACK_PAYLOAD", Message: "invalid callback payload"}
)

type Repo interface {
	FindTransactionByID(id uuid.UUID) (*transaction.Transaction, error)
	FindPaymentByTransactionID(transactionID uuid.UUID) (*Payment, error)
	FindPaymentByOrderID(orderID string) (*Payment, error)
	CreatePayment(tx *gorm.DB, p *Payment) error
	UpdatePayment(tx *gorm.DB, p *Payment) error
	CreateCallback(tx *gorm.DB, c *MidtransCallback) error
	UpdateCallback(tx *gorm.DB, c *MidtransCallback) error
	UpdateTransaction(tx *gorm.DB, t *transaction.Transaction) error
	CreateTransactionLog(tx *gorm.DB, l *transaction.TransactionLog) error
	CreateRefund(tx *gorm.DB, refund *Refund) error
	ListRefundsByTransactionID(transactionID uuid.UUID) ([]Refund, error)
}

type MidtransGateway interface {
	CreateQRIS(ctx context.Context, orderID string, amount int) (*midtrans.QRISResult, error)
	ValidateCallback(orderID, statusCode, grossAmount, signature string) bool
}

type Service struct {
	repo      Repo
	gateway   MidtransGateway
	hub       *pubsub.Hub
	db        *gorm.DB
	now       func() time.Time
	runTx     func(fn func(tx *gorm.DB) error) error
	channelFn func(gateID uuid.UUID) string
}

func NewService(repo Repo, gateway MidtransGateway, hub *pubsub.Hub, db *gorm.DB) *Service {
	s := &Service{
		repo:    repo,
		gateway: gateway,
		hub:     hub,
		db:      db,
		now:     time.Now,
		channelFn: func(gateID uuid.UUID) string {
			return "gate:" + gateID.String()
		},
	}
	s.runTx = s.defaultRunTx
	return s
}

type CashInput struct {
	TransactionID uuid.UUID
	CashTendered  int
	ActorID       uuid.UUID
}

type QRISInput struct {
	TransactionID uuid.UUID
	ActorID       uuid.UUID
}

type RefundInput struct {
	TransactionID uuid.UUID
	Amount        int
	Reason        string
	ActorID       uuid.UUID
}

func (s *Service) PayCash(ctx context.Context, in CashInput) (*Payment, error) {
	txData, err := s.repo.FindTransactionByID(in.TransactionID)
	if err != nil {
		return nil, err
	}
	if txData == nil {
		return nil, ErrTransactionNotFound
	}
	if txData.Status != transaction.StatusPendingPayment {
		return nil, ErrInvalidStatus
	}

	existing, err := s.repo.FindPaymentByTransactionID(in.TransactionID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status != StatusFailed {
		return nil, ErrPaymentExists
	}

	if in.CashTendered < txData.CalculatedFee {
		return nil, ErrInsufficientCash
	}

	now := s.now()
	change := in.CashTendered - txData.CalculatedFee
	payment := &Payment{
		ID:              uuid.New(),
		TransactionID:   txData.ID,
		Method:          MethodCash,
		Amount:          txData.CalculatedFee,
		Status:          StatusCompleted,
		CashTendered:    in.CashTendered,
		CashChange:      change,
		TenantID:        txData.TenantID,
		PaidAt:          &now,
		CreatedAt:       now,
		UpdatedAt:       now,
		HandledByUserID: nil,
	}
	if in.ActorID != uuid.Nil {
		payment.HandledByUserID = &in.ActorID
	}

	err = s.runTx(func(tx *gorm.DB) error {
		if err := s.repo.CreatePayment(tx, payment); err != nil {
			return err
		}
		if tx != nil {
			if err := outbox.Publish(tx, "audit.payment.completed", payment); err != nil {
				return err
			}
		}

		txData.Status = transaction.StatusCompleted
		txData.UpdatedAt = now
		if err := s.repo.UpdateTransaction(tx, txData); err != nil {
			return err
		}

		log := &transaction.TransactionLog{
			ID:            uuid.New(),
			TransactionID: txData.ID,
			FromStatus:    transaction.StatusPendingPayment,
			ToStatus:      transaction.StatusCompleted,
			Event:         "payment_completed",
			TriggeredBy:   "user",
			CreatedAt:     now,
		}
		if in.ActorID != uuid.Nil {
			log.TriggeredByUserID = &in.ActorID
		}
		return s.repo.CreateTransactionLog(tx, log)
	})
	if err != nil {
		return nil, err
	}

	s.publishBarrier(txData)
	_ = ctx
	return payment, nil
}

func (s *Service) CreateQRIS(ctx context.Context, in QRISInput) (*Payment, error) {
	txData, err := s.repo.FindTransactionByID(in.TransactionID)
	if err != nil {
		return nil, err
	}
	if txData == nil {
		return nil, ErrTransactionNotFound
	}
	if txData.Status != transaction.StatusPendingPayment {
		return nil, ErrInvalidStatus
	}

	existing, err := s.repo.FindPaymentByTransactionID(in.TransactionID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status != StatusFailed {
		return nil, ErrPaymentExists
	}

	orderID := "PKR-" + txData.TransactionCode
	result, err := s.gateway.CreateQRIS(ctx, orderID, txData.CalculatedFee)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentGateway, err)
	}

	now := s.now()
	payment := &Payment{
		ID:              uuid.New(),
		TransactionID:   txData.ID,
		Method:          MethodQRIS,
		Amount:          txData.CalculatedFee,
		Status:          StatusPending,
		MidtransOrderID: orderID,
		QRISString:      result.QRISString,
		QRISImageURL:    result.QRISImageURL,
		QRISExpiresAt:   &result.ExpiresAt,
		TenantID:        txData.TenantID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if in.ActorID != uuid.Nil {
		payment.HandledByUserID = &in.ActorID
	}

	if err := s.repo.CreatePayment(nil, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *Service) HandleMidtransCallback(ctx context.Context, payload *midtrans.CallbackPayload, rawPayload []byte) error {
	if payload == nil {
		return ErrInvalidCallbackPayload
	}

	now := s.now()
	cb := &MidtransCallback{
		ID:              uuid.New(),
		MidtransOrderID: payload.OrderID,
		RawPayload:      rawPayload,
		SignatureValid:  false,
		Processed:       false,
		ReceivedAt:      now,
	}

	payment, err := s.repo.FindPaymentByOrderID(payload.OrderID)
	if err != nil {
		return err
	}
	if payment != nil {
		cb.PaymentID = &payment.ID
	}
	if err := s.repo.CreateCallback(nil, cb); err != nil {
		return err
	}

	isValid := s.gateway.ValidateCallback(payload.OrderID, payload.StatusCode, payload.GrossAmount, payload.SignatureKey)
	if !isValid {
		cb.SignatureValid = false
		cb.Processed = true
		cb.ProcessedAt = &now
		cb.ErrorMessage = "INVALID_SIGNATURE"
		return s.repo.UpdateCallback(nil, cb)
	}

	cb.SignatureValid = true
	if payment == nil {
		cb.Processed = true
		cb.ProcessedAt = &now
		cb.ErrorMessage = "ORDER_NOT_FOUND"
		return s.repo.UpdateCallback(nil, cb)
	}

	err = s.runTx(func(tx *gorm.DB) error {
		status := strings.ToLower(payload.TransactionStatus)
		if status == "settlement" {
			if payment.Status != StatusCompleted {
				payment.Status = StatusCompleted
				payment.PaidAt = &now
				payment.MidtransTransactionID = payload.TransactionID
				payment.MidtransStatus = payload.TransactionStatus
				payment.UpdatedAt = now
				if err := s.repo.UpdatePayment(tx, payment); err != nil {
					return err
				}
				if tx != nil {
					if err := outbox.Publish(tx, "audit.payment.completed", payment); err != nil {
						return err
					}
				}

				txData, err := s.repo.FindTransactionByID(payment.TransactionID)
				if err != nil {
					return err
				}
				if txData != nil {
					txData.Status = transaction.StatusCompleted
					txData.UpdatedAt = now
					if err := s.repo.UpdateTransaction(tx, txData); err != nil {
						return err
					}
					log := &transaction.TransactionLog{
						ID:            uuid.New(),
						TransactionID: txData.ID,
						FromStatus:    transaction.StatusPendingPayment,
						ToStatus:      transaction.StatusCompleted,
						Event:         "payment_completed",
						TriggeredBy:   "system",
						CreatedAt:     now,
					}
					if err := s.repo.CreateTransactionLog(tx, log); err != nil {
						return err
					}
					s.publishBarrier(txData)
				}
			}
		} else if status == "expire" || status == "cancel" {
			payment.Status = StatusFailed
			payment.MidtransStatus = payload.TransactionStatus
			payment.UpdatedAt = now
			if err := s.repo.UpdatePayment(tx, payment); err != nil {
				return err
			}
		}

		cb.Processed = true
		cb.ProcessedAt = &now
		cb.ErrorMessage = ""
		return s.repo.UpdateCallback(tx, cb)
	})
	if err != nil {
		return err
	}

	_ = ctx
	return nil
}

func (s *Service) GetByTransactionID(ctx context.Context, transactionID uuid.UUID) (*Payment, error) {
	_ = ctx
	return s.repo.FindPaymentByTransactionID(transactionID)
}

func (s *Service) CreateRefund(ctx context.Context, in RefundInput) (*Refund, error) {
	_ = ctx

	txData, err := s.repo.FindTransactionByID(in.TransactionID)
	if err != nil {
		return nil, err
	}
	if txData == nil {
		return nil, ErrTransactionNotFound
	}

	payment, err := s.repo.FindPaymentByTransactionID(in.TransactionID)
	if err != nil {
		return nil, err
	}
	if payment == nil || payment.Status != StatusCompleted {
		return nil, ErrPaymentNotFound
	}

	now := s.now()
	refund := &Refund{
		ID:            uuid.New(),
		TransactionID: in.TransactionID,
		PaymentID:     payment.ID,
		Amount:        in.Amount,
		Reason:        strings.TrimSpace(in.Reason),
		Status:        RefundStatusApproved,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if in.ActorID != uuid.Nil {
		refund.RequestedBy = &in.ActorID
	}

	if err := s.repo.CreateRefund(nil, refund); err != nil {
		return nil, err
	}
	return refund, nil
}

func (s *Service) ListRefunds(ctx context.Context, transactionID uuid.UUID) ([]Refund, error) {
	_ = ctx
	return s.repo.ListRefundsByTransactionID(transactionID)
}

func (s *Service) publishBarrier(txData *transaction.Transaction) {
	if s.hub == nil || txData == nil || txData.ExitGateID == nil {
		return
	}
	s.hub.Publish(s.channelFn(*txData.ExitGateID), sse.NewEventMessage("gate.barrier.open", map[string]any{
		"gate_id":        *txData.ExitGateID,
		"transaction_id": txData.ID,
		"source":         "payment",
	}))
}

func (s *Service) defaultRunTx(fn func(tx *gorm.DB) error) error {
	if s.db == nil {
		return fn(nil)
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) || errors.Is(err, ErrInsufficientCash) {
			return err
		}
		return err
	}
	return nil
}

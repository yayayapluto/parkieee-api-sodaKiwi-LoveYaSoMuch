package payment

import (
	"errors"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindTransactionByID(id uuid.UUID) (*transaction.Transaction, error) {
	var t transaction.Transaction
	err := r.db.Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *Repository) FindPaymentByTransactionID(transactionID uuid.UUID) (*Payment, error) {
	var p Payment
	err := r.db.Where("transaction_id = ?", transactionID).Order("created_at DESC").First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *Repository) FindPaymentByOrderID(orderID string) (*Payment, error) {
	var p Payment
	err := r.db.Where("midtrans_order_id = ?", orderID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *Repository) CreatePayment(tx *gorm.DB, p *Payment) error {
	return r.session(tx).Create(p).Error
}

func (r *Repository) UpdatePayment(tx *gorm.DB, p *Payment) error {
	return r.session(tx).Save(p).Error
}

func (r *Repository) CreateCallback(tx *gorm.DB, c *MidtransCallback) error {
	return r.session(tx).Create(c).Error
}

func (r *Repository) UpdateCallback(tx *gorm.DB, c *MidtransCallback) error {
	return r.session(tx).Save(c).Error
}

func (r *Repository) UpdateTransaction(tx *gorm.DB, t *transaction.Transaction) error {
	return r.session(tx).Save(t).Error
}

func (r *Repository) CreateTransactionLog(tx *gorm.DB, l *transaction.TransactionLog) error {
	return r.session(tx).Create(l).Error
}

func (r *Repository) CreateRefund(tx *gorm.DB, refund *Refund) error {
	return r.session(tx).Create(refund).Error
}

func (r *Repository) ListRefundsByTransactionID(transactionID uuid.UUID) ([]Refund, error) {
	var rows []Refund
	err := r.db.Where("transaction_id = ?", transactionID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *Repository) session(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

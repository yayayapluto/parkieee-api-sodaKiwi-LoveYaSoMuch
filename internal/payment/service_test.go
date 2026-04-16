package payment

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"github.com/yyypluto/parkieee-api/pkg/midtrans"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
	"gorm.io/gorm"
)

type fakeGateway struct {
	qrisResult    *midtrans.QRISResult
	qrisErr       error
	validateOK    bool
	createInvoked int
}

func (f *fakeGateway) CreateQRIS(_ context.Context, _ string, _ int) (*midtrans.QRISResult, error) {
	f.createInvoked++
	if f.qrisErr != nil {
		return nil, f.qrisErr
	}
	return f.qrisResult, nil
}

func (f *fakeGateway) ValidateCallback(_, _, _, _ string) bool {
	return f.validateOK
}

type fakeRepo struct {
	transactionByID      *transaction.Transaction
	paymentByTransaction *Payment
	paymentByOrder       *Payment
	createdPayments      []*Payment
	updatedPayments      []*Payment
	createdCallbacks     []*MidtransCallback
	updatedCallbacks     []*MidtransCallback
	updatedTransactions  []*transaction.Transaction
	createdLogs          []*transaction.TransactionLog
	createdRefunds       []*Refund
	refundsByTransaction []Refund
}

func (f *fakeRepo) FindTransactionByID(uuid.UUID) (*transaction.Transaction, error) {
	return f.transactionByID, nil
}

func (f *fakeRepo) FindPaymentByTransactionID(uuid.UUID) (*Payment, error) {
	return f.paymentByTransaction, nil
}

func (f *fakeRepo) FindPaymentByOrderID(string) (*Payment, error) {
	return f.paymentByOrder, nil
}

func (f *fakeRepo) CreatePayment(_ *gorm.DB, p *Payment) error {
	f.createdPayments = append(f.createdPayments, p)
	return nil
}

func (f *fakeRepo) UpdatePayment(_ *gorm.DB, p *Payment) error {
	f.updatedPayments = append(f.updatedPayments, p)
	return nil
}

func (f *fakeRepo) CreateCallback(_ *gorm.DB, c *MidtransCallback) error {
	f.createdCallbacks = append(f.createdCallbacks, c)
	return nil
}

func (f *fakeRepo) UpdateCallback(_ *gorm.DB, c *MidtransCallback) error {
	f.updatedCallbacks = append(f.updatedCallbacks, c)
	return nil
}

func (f *fakeRepo) UpdateTransaction(_ *gorm.DB, t *transaction.Transaction) error {
	f.updatedTransactions = append(f.updatedTransactions, t)
	return nil
}

func (f *fakeRepo) CreateTransactionLog(_ *gorm.DB, l *transaction.TransactionLog) error {
	f.createdLogs = append(f.createdLogs, l)
	return nil
}

func (f *fakeRepo) CreateRefund(_ *gorm.DB, r *Refund) error {
	f.createdRefunds = append(f.createdRefunds, r)
	return nil
}

func (f *fakeRepo) ListRefundsByTransactionID(uuid.UUID) ([]Refund, error) {
	return f.refundsByTransaction, nil
}

func TestServicePayCash_SuccessPublishesBarrier(t *testing.T) {
	exitGateID := uuid.New()
	repo := &fakeRepo{
		transactionByID: &transaction.Transaction{
			ID:            uuid.New(),
			Status:        transaction.StatusPendingPayment,
			CalculatedFee: 10000,
			TenantID:      uuid.New(),
			ExitGateID:    &exitGateID,
		},
	}

	hub := pubsub.NewHub()
	ch := hub.Subscribe("gate:" + exitGateID.String())
	defer hub.Unsubscribe("gate:"+exitGateID.String(), ch)

	svc := NewService(repo, &fakeGateway{}, hub, nil)
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	payment, err := svc.PayCash(context.Background(), CashInput{
		TransactionID: repo.transactionByID.ID,
		CashTendered:  12000,
		ActorID:       uuid.New(),
	})
	require.NoError(t, err)
	require.NotNil(t, payment)
	assert.Equal(t, StatusCompleted, payment.Status)
	assert.Equal(t, 2000, payment.CashChange)
	assert.Len(t, repo.updatedTransactions, 1)
	assert.Equal(t, transaction.StatusCompleted, repo.updatedTransactions[0].Status)

	select {
	case msg := <-ch:
		var event struct {
			Type      string         `json:"type"`
			Timestamp string         `json:"timestamp"`
			Payload   map[string]any `json:"payload"`
		}
		require.NoError(t, json.Unmarshal([]byte(msg), &event))
		assert.Equal(t, "gate.barrier.open", event.Type)
		assert.NotEmpty(t, event.Timestamp)
		assert.NotNil(t, event.Payload)
	case <-time.After(1 * time.Second):
		t.Fatal("expected barrier event")
	}
}

func TestServiceCreateQRIS_GatewayError(t *testing.T) {
	repo := &fakeRepo{
		transactionByID: &transaction.Transaction{
			ID:              uuid.New(),
			Status:          transaction.StatusPendingPayment,
			CalculatedFee:   5000,
			TransactionCode: "TXN-ABCD1234",
		},
	}

	gateway := &fakeGateway{qrisErr: errors.New("gateway down")}
	svc := NewService(repo, gateway, nil, nil)

	out, err := svc.CreateQRIS(context.Background(), QRISInput{TransactionID: repo.transactionByID.ID})
	assert.Nil(t, out)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPaymentGateway)
	assert.Equal(t, 1, gateway.createInvoked)
}

func TestServiceHandleCallback_UnknownOrderStoredAndProcessed(t *testing.T) {
	repo := &fakeRepo{}
	gateway := &fakeGateway{validateOK: true}
	svc := NewService(repo, gateway, nil, nil)

	payload := &midtrans.CallbackPayload{
		OrderID:       "PKR-NOT-FOUND",
		StatusCode:    "200",
		GrossAmount:   "10000",
		SignatureKey:  "sig",
		TransactionID: "trx-1",
	}
	err := svc.HandleMidtransCallback(context.Background(), payload, []byte(`{"order_id":"PKR-NOT-FOUND"}`))
	require.NoError(t, err)
	require.Len(t, repo.createdCallbacks, 1)
	require.Len(t, repo.updatedCallbacks, 1)
	assert.True(t, repo.updatedCallbacks[0].Processed)
	assert.Equal(t, "ORDER_NOT_FOUND", repo.updatedCallbacks[0].ErrorMessage)
	assert.True(t, repo.updatedCallbacks[0].SignatureValid)
}

func TestServiceCreateRefund_Success(t *testing.T) {
	txID := uuid.New()
	repo := &fakeRepo{
		transactionByID: &transaction.Transaction{ID: txID, Status: transaction.StatusCompleted},
		paymentByTransaction: &Payment{ID: uuid.New(), TransactionID: txID, Status: StatusCompleted},
	}
	svc := NewService(repo, &fakeGateway{}, nil, nil)

	refund, err := svc.CreateRefund(context.Background(), RefundInput{TransactionID: txID, Amount: 5000, Reason: "manual adjustment", ActorID: uuid.New()})
	require.NoError(t, err)
	require.NotNil(t, refund)
	assert.Equal(t, RefundStatusApproved, refund.Status)
	require.Len(t, repo.createdRefunds, 1)
}

func TestServiceCreateRefund_NoCompletedPayment(t *testing.T) {
	txID := uuid.New()
	repo := &fakeRepo{transactionByID: &transaction.Transaction{ID: txID, Status: transaction.StatusCompleted}}
	svc := NewService(repo, &fakeGateway{}, nil, nil)

	refund, err := svc.CreateRefund(context.Background(), RefundInput{TransactionID: txID, Amount: 1000})
	assert.Nil(t, refund)
	assert.ErrorIs(t, err, ErrPaymentNotFound)
}

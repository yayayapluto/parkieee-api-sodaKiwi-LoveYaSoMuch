package transaction

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"gorm.io/gorm"
)

type fakeOCR struct {
	enqueued int
	lastType string
}

type fakeUploader struct {
	url string
	err error
	key string
}

func (f *fakeUploader) PutObject(_ context.Context, key string, _ []byte, _ string) (string, error) {
	f.key = key
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

func (f *fakeOCR) EnqueueJob(_ *gorm.DB, _ uuid.UUID, _ string, photoType string) {
	f.enqueued++
	f.lastType = photoType
}

type fakeRepo struct {
	gate        *Gate
	card        *RFIDCard
	activeByVeh *Transaction
	activeByTkt *Transaction
	byID        *Transaction
	feeConfig   *fee.FeeConfig
	tiers       []fee.FeeTier
	vehicle     *Vehicle
	acquireOk   bool
	createdTx   *Transaction
	updatedTx   *Transaction
	createdLogs []*TransactionLog
	listRows    []Transaction
	listTotal   int64
	logRows     []TransactionLog
	logTotal    int64
	entryPhoto  map[uuid.UUID]string
	exitPhoto   map[uuid.UUID]string
}

func (f *fakeRepo) CreateTransaction(_ *gorm.DB, t *Transaction) error {
	f.createdTx = t
	return nil
}
func (f *fakeRepo) FindActiveByVehicle(uuid.UUID) (*Transaction, error) { return f.activeByVeh, nil }
func (f *fakeRepo) FindActiveByTicket(string) (*Transaction, error)     { return f.activeByTkt, nil }
func (f *fakeRepo) FindByID(uuid.UUID) (*Transaction, error)            { return f.byID, nil }
func (f *fakeRepo) UpdateTransaction(_ *gorm.DB, t *Transaction) error {
	f.updatedTx = t
	return nil
}
func (f *fakeRepo) CreateLog(_ *gorm.DB, l *TransactionLog) error {
	f.createdLogs = append(f.createdLogs, l)
	return nil
}
func (f *fakeRepo) FindRFIDCard(string) (*RFIDCard, error) { return f.card, nil }
func (f *fakeRepo) FindGate(uuid.UUID) (*Gate, error)      { return f.gate, nil }
func (f *fakeRepo) FindVehicleByID(uuid.UUID) (*Vehicle, error) {
	return f.vehicle, nil
}
func (f *fakeRepo) FindActiveFeeConfig(uuid.UUID, uuid.UUID) (*fee.FeeConfig, error) {
	return f.feeConfig, nil
}
func (f *fakeRepo) FindTiersByConfig(uuid.UUID) ([]fee.FeeTier, error) { return f.tiers, nil }
func (f *fakeRepo) ListTransactions(ListFilter) ([]Transaction, int64, error) {
	return f.listRows, f.listTotal, nil
}
func (f *fakeRepo) ListLogs(uuid.UUID, int, int) ([]TransactionLog, int64, error) {
	return f.logRows, f.logTotal, nil
}
func (f *fakeRepo) AcquireAdvisoryLock(_ *gorm.DB, _ string) (bool, error) {
	return f.acquireOk, nil
}
func (f *fakeRepo) UpdateEntryPhotoURL(id uuid.UUID, url string) error {
	if f.entryPhoto == nil {
		f.entryPhoto = map[uuid.UUID]string{}
	}
	f.entryPhoto[id] = url
	return nil
}
func (f *fakeRepo) UpdateExitPhotoURL(id uuid.UUID, url string) error {
	if f.exitPhoto == nil {
		f.exitPhoto = map[uuid.UUID]string{}
	}
	f.exitPhoto[id] = url
	return nil
}

func TestServiceEntry_DuplicateRFIDReturnsError(t *testing.T) {
	uid := "RFID-1"
	repo := &fakeRepo{
		gate:        &Gate{ID: uuid.New(), ZoneID: uuid.New(), GateType: "entry", IsActive: true},
		card:        &RFIDCard{ID: uuid.New(), VehicleID: uuid.New(), IsActive: true},
		activeByVeh: &Transaction{ID: uuid.New(), Status: StatusActive},
		acquireOk:   true,
	}

	svc := NewService(repo, nil, nil, nil)
	_, err := svc.Entry(context.Background(), EntryInput{GateID: repo.gate.ID, Method: MethodRFID, RFIDCardUID: &uid})
	assert.ErrorIs(t, err, ErrDoubleEntry)
	assert.Nil(t, repo.createdTx)
}

func TestServiceExit_SetsPendingPaymentAndCalculatedFee(t *testing.T) {
	uid := "RFID-2"
	vehicleTypeID := uuid.New()
	active := &Transaction{
		ID:      uuid.New(),
		ZoneID:  uuid.New(),
		Status:  StatusActive,
		EntryAt: time.Date(2026, 4, 15, 8, 0, 0, 0, time.UTC),
	}

	repo := &fakeRepo{
		gate:        &Gate{ID: uuid.New(), ZoneID: active.ZoneID, GateType: "exit", IsActive: true},
		card:        &RFIDCard{ID: uuid.New(), VehicleID: uuid.New(), VehicleTypeID: &vehicleTypeID, IsActive: true},
		activeByVeh: active,
		feeConfig:   &fee.FeeConfig{ID: uuid.New(), BaseFee: 1000, GracePeriodMinutes: 0},
		tiers: []fee.FeeTier{
			{ID: uuid.New(), TierOrder: 1, DurationMinutes: 60, FeeAmount: 2000, IsLastTier: true},
		},
		acquireOk: true,
	}

	ocrSvc := &fakeOCR{}
	svc := NewService(repo, nil, ocrSvc, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	}

	out, err := svc.Exit(context.Background(), ExitInput{GateID: repo.gate.ID, Method: MethodRFID, RFIDCardUID: &uid, PhotoPath: "/tmp/exit.jpg"})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, StatusPendingPayment, out.Status)
	assert.Equal(t, 5000, out.CalculatedFee)
	assert.NotNil(t, repo.updatedTx)
	assert.Equal(t, 1, ocrSvc.enqueued)
	assert.Equal(t, "exit", ocrSvc.lastType)
	assert.Len(t, repo.createdLogs, 1)
}

func TestServiceDetail_NotFound(t *testing.T) {
	repo := &fakeRepo{byID: nil}
	svc := NewService(repo, nil, nil, nil)

	out, err := svc.Detail(context.Background(), uuid.New())
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrTransactionNotFound)
}

func TestServiceEntry_TicketGeneratesQRCodeBase64(t *testing.T) {
	repo := &fakeRepo{
		gate:      &Gate{ID: uuid.New(), ZoneID: uuid.New(), GateType: "entry", IsActive: true},
		acquireOk: true,
	}

	svc := NewService(repo, nil, nil, nil)
	out, err := svc.Entry(context.Background(), EntryInput{GateID: repo.gate.ID, Method: MethodTicket, PhotoPath: "/tmp/a.jpg"})
	require.NoError(t, err)
	require.NotNil(t, out.TicketCode)
	require.NotEmpty(t, out.TicketCodeImage)

	decoded, decodeErr := base64.StdEncoding.DecodeString(out.TicketCodeImage)
	require.NoError(t, decodeErr)
	assert.Greater(t, len(decoded), 10)
}

func TestServiceEntry_UpdatesEntryPhotoURLForHTTPPath(t *testing.T) {
	repo := &fakeRepo{
		gate:      &Gate{ID: uuid.New(), ZoneID: uuid.New(), GateType: "entry", IsActive: true},
		acquireOk: true,
	}

	svc := NewService(repo, nil, nil, nil)
	svc.runAsync = func(fn func()) { fn() }

	out, err := svc.Entry(context.Background(), EntryInput{GateID: repo.gate.ID, Method: MethodTicket, PhotoPath: "https://cdn.local/entry.jpg"})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, repo.entryPhoto)
	assert.Equal(t, "https://cdn.local/entry.jpg", repo.entryPhoto[out.ID])
}

func TestServiceListLogs(t *testing.T) {
	repo := &fakeRepo{
		logRows: []TransactionLog{{ID: uuid.New(), Event: "entry"}},
		logTotal: 1,
	}
	svc := NewService(repo, nil, nil, nil)

	rows, total, err := svc.ListLogs(context.Background(), uuid.New(), 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, "entry", rows[0].Event)
}

func TestServiceUploadPhoto(t *testing.T) {
	uploader := &fakeUploader{url: "https://cdn.local/uploads/entry/img.jpg"}
	svc := NewService(&fakeRepo{}, nil, nil, uploader)

	url, err := svc.UploadPhoto(context.Background(), "entry", []byte("img"), "image/jpeg")
	require.NoError(t, err)
	assert.Equal(t, uploader.url, url)
	assert.Contains(t, uploader.key, "uploads/entry/")
}

func TestServiceUploadPhoto_InvalidType(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil, nil, &fakeUploader{url: "ok"})

	url, err := svc.UploadPhoto(context.Background(), "other", []byte("img"), "image/jpeg")
	assert.Empty(t, url)
	assert.ErrorIs(t, err, ErrInvalidPhotoType)
}

package transaction

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
	"gorm.io/gorm"
)

type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

var (
	ErrGateNotFound        = &AppError{Code: "GATE_NOT_FOUND", Message: "gate not found"}
	ErrInvalidGateType     = &AppError{Code: "INVALID_GATE_TYPE", Message: "invalid gate type"}
	ErrRFIDNotFound        = &AppError{Code: "RFID_NOT_FOUND", Message: "rfid card not found"}
	ErrDoubleEntry         = &AppError{Code: "DOUBLE_ENTRY", Message: "double entry detected"}
	ErrTransactionNotFound = &AppError{Code: "TRANSACTION_NOT_FOUND", Message: "transaction not found"}
	ErrNoFeeConfig         = &AppError{Code: "NO_FEE_CONFIG", Message: "no active fee config"}
	ErrInvalidMethod       = &AppError{Code: "INVALID_ENTRY_METHOD", Message: "invalid entry/exit method"}
	ErrInvalidPhotoType    = &AppError{Code: "INVALID_PHOTO_TYPE", Message: "invalid photo type"}
)

type Repo interface {
	CreateTransaction(tx *gorm.DB, t *Transaction) error
	FindActiveByVehicle(vehicleID uuid.UUID) (*Transaction, error)
	FindActiveByTicket(ticketCode string) (*Transaction, error)
	FindByID(id uuid.UUID) (*Transaction, error)
	UpdateTransaction(tx *gorm.DB, t *Transaction) error
	CreateLog(tx *gorm.DB, l *TransactionLog) error
	FindRFIDCard(cardUID string) (*RFIDCard, error)
	FindGate(id uuid.UUID) (*Gate, error)
	FindVehicleByID(id uuid.UUID) (*Vehicle, error)
	FindActiveFeeConfig(zoneID, vehicleTypeID uuid.UUID) (*fee.FeeConfig, error)
	FindTiersByConfig(configID uuid.UUID) ([]fee.FeeTier, error)
	ListTransactions(filter ListFilter) ([]Transaction, int64, error)
	ListLogs(transactionID uuid.UUID, page, limit int) ([]TransactionLog, int64, error)
	AcquireAdvisoryLock(tx *gorm.DB, key string) (bool, error)
	UpdateEntryPhotoURL(id uuid.UUID, url string) error
	UpdateExitPhotoURL(id uuid.UUID, url string) error
}

type OCRService interface {
	EnqueueJob(db *gorm.DB, txID uuid.UUID, photoPath, photoType string)
}

type Service struct {
	repo     Repo
	db       *gorm.DB
	ocr      OCRService
	uploader Uploader
	baseCtx  context.Context
	now      func() time.Time
	runAsync func(fn func())
}

type Uploader interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) (string, error)
}

func NewService(repo Repo, db *gorm.DB, ocr OCRService, uploader Uploader) *Service {
	return &Service{
		repo:     repo,
		db:       db,
		ocr:      ocr,
		uploader: uploader,
		baseCtx:  context.Background(),
		now:      time.Now,
		runAsync: func(fn func()) { go fn() },
	}
}

func (s *Service) SetBaseContext(ctx context.Context) {
	if ctx == nil {
		s.baseCtx = context.Background()
		return
	}
	s.baseCtx = ctx
}

type EntryInput struct {
	GateID      uuid.UUID
	Method      string
	RFIDCardUID *string
	PhotoPath   string
	ActorID     uuid.UUID
	TenantID    uuid.UUID
}

type ExitInput struct {
	GateID      uuid.UUID
	Method      string
	RFIDCardUID *string
	TicketCode  *string
	PhotoPath   string
	ActorID     uuid.UUID
}

func (s *Service) Entry(ctx context.Context, in EntryInput) (*Transaction, error) {
	gate, err := s.repo.FindGate(in.GateID)
	if err != nil {
		return nil, err
	}
	if gate == nil {
		return nil, ErrGateNotFound
	}
	if gate.GateType != "entry" {
		return nil, ErrInvalidGateType
	}

	now := s.now()
	transactionCode := "TXN-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:8]

	txData := &Transaction{
		ID:              uuid.New(),
		TransactionCode: transactionCode,
		EntryGateID:     in.GateID,
		EntryMethod:     in.Method,
		EntryAt:         now,
		EntryPhotoPath:  in.PhotoPath,
		Status:          StatusActive,
		ZoneID:          gate.ZoneID,
		TenantID:        in.TenantID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	var lockKey string
	method := strings.ToLower(strings.TrimSpace(in.Method))
	if method == MethodRFID {
		if in.RFIDCardUID == nil || strings.TrimSpace(*in.RFIDCardUID) == "" {
			return nil, ErrRFIDNotFound
		}

		card, err := s.repo.FindRFIDCard(strings.TrimSpace(*in.RFIDCardUID))
		if err != nil {
			return nil, err
		}
		if card == nil {
			return nil, ErrRFIDNotFound
		}

		active, err := s.repo.FindActiveByVehicle(card.VehicleID)
		if err != nil {
			return nil, err
		}
		if active != nil {
			return nil, ErrDoubleEntry
		}

		lockKey = card.VehicleID.String()
		txData.VehicleID = &card.VehicleID
		txData.RFIDCardID = &card.ID
	} else if method == MethodTicket {
		ticket := uuid.NewString()
		png, err := qrcode.Encode(ticket, qrcode.Medium, 256)
		if err != nil {
			return nil, err
		}
		ticketImage := base64.StdEncoding.EncodeToString(png)
		lockKey = ticket
		txData.TicketCode = &ticket
		txData.TicketCodeImage = ticketImage
	} else {
		return nil, ErrInvalidMethod
	}

	err = s.withTransaction(func(tx *gorm.DB) error {
		acquired, err := s.repo.AcquireAdvisoryLock(tx, lockKey)
		if err != nil {
			return err
		}
		if !acquired {
			return ErrDoubleEntry
		}

		if err := s.repo.CreateTransaction(tx, txData); err != nil {
			return err
		}

		if tx != nil {
			if err := outbox.Publish(tx, "audit.transaction.entry", txData); err != nil {
				return err
			}
		}

		log := &TransactionLog{
			ID:            uuid.New(),
			TransactionID: txData.ID,
			ToStatus:      StatusActive,
			Event:         "entry",
			TriggeredBy:   "system",
			CreatedAt:     now,
		}
		if in.ActorID != uuid.Nil {
			log.TriggeredByUserID = &in.ActorID
		}
		return s.repo.CreateLog(tx, log)
	})
	if err != nil {
		return nil, err
	}

	if s.ocr != nil {
		s.ocr.EnqueueJob(s.db, txData.ID, in.PhotoPath, "entry")
	}
	s.runAsync(func() {
		s.uploadPhotoAndUpdate(txData.ID, in.PhotoPath, "entry")
	})

	return txData, nil
}

func (s *Service) Exit(ctx context.Context, in ExitInput) (*Transaction, error) {
	gate, err := s.repo.FindGate(in.GateID)
	if err != nil {
		return nil, err
	}
	if gate == nil {
		return nil, ErrGateNotFound
	}
	if gate.GateType != "exit" {
		return nil, ErrInvalidGateType
	}

	method := strings.ToLower(strings.TrimSpace(in.Method))
	var active *Transaction
	var vehicleTypeID *uuid.UUID

	if method == MethodRFID {
		if in.RFIDCardUID == nil || strings.TrimSpace(*in.RFIDCardUID) == "" {
			return nil, ErrRFIDNotFound
		}
		card, err := s.repo.FindRFIDCard(strings.TrimSpace(*in.RFIDCardUID))
		if err != nil {
			return nil, err
		}
		if card == nil {
			return nil, ErrRFIDNotFound
		}
		active, err = s.repo.FindActiveByVehicle(card.VehicleID)
		if err != nil {
			return nil, err
		}
		vehicleTypeID = card.VehicleTypeID
	} else if method == MethodTicket {
		if in.TicketCode == nil || strings.TrimSpace(*in.TicketCode) == "" {
			return nil, ErrTransactionNotFound
		}
		active, err = s.repo.FindActiveByTicket(strings.TrimSpace(*in.TicketCode))
		if err != nil {
			return nil, err
		}
		if active != nil && active.VehicleID != nil {
			vehicle, err := s.repo.FindVehicleByID(*active.VehicleID)
			if err != nil {
				return nil, err
			}
			if vehicle != nil {
				vehicleTypeID = vehicle.VehicleTypeID
			}
		}
	} else {
		return nil, ErrInvalidMethod
	}

	if active == nil {
		return nil, ErrTransactionNotFound
	}
	if vehicleTypeID == nil {
		return nil, ErrNoFeeConfig
	}

	cfg, err := s.repo.FindActiveFeeConfig(active.ZoneID, *vehicleTypeID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, ErrNoFeeConfig
	}

	tiers, err := s.repo.FindTiersByConfig(cfg.ID)
	if err != nil {
		return nil, err
	}

	now := s.now()
	durationMinutes := int(now.Sub(active.EntryAt).Minutes())
	if durationMinutes < 0 {
		durationMinutes = 0
	}

	amount, err := fee.Calculate(*cfg, tiers, durationMinutes)
	if err != nil {
		return nil, fmt.Errorf("calculate fee: %w", err)
	}

	err = s.withTransaction(func(tx *gorm.DB) error {
		active.ExitGateID = &in.GateID
		active.ExitMethod = &method
		active.ExitAt = &now
		active.ExitPhotoPath = in.PhotoPath
		active.CalculatedFee = amount
		active.FeeConfigID = &cfg.ID
		active.Status = StatusPendingPayment
		active.UpdatedAt = now

		if err := s.repo.UpdateTransaction(tx, active); err != nil {
			return err
		}

		if tx != nil {
			if err := outbox.Publish(tx, "audit.transaction.exit", active); err != nil {
				return err
			}
		}

		log := &TransactionLog{
			ID:            uuid.New(),
			TransactionID: active.ID,
			FromStatus:    StatusActive,
			ToStatus:      StatusPendingPayment,
			Event:         "exit",
			TriggeredBy:   "system",
			CreatedAt:     now,
		}
		if in.ActorID != uuid.Nil {
			log.TriggeredByUserID = &in.ActorID
		}
		return s.repo.CreateLog(tx, log)
	})
	if err != nil {
		return nil, err
	}

	if s.ocr != nil {
		s.ocr.EnqueueJob(s.db, active.ID, in.PhotoPath, "exit")
	}
	s.runAsync(func() {
		s.uploadPhotoAndUpdate(active.ID, in.PhotoPath, "exit")
	})

	return active, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Transaction, int64, error) {
	_ = ctx
	return s.repo.ListTransactions(filter)
}

func (s *Service) Detail(ctx context.Context, id uuid.UUID) (*Transaction, error) {
	_ = ctx
	out, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, ErrTransactionNotFound
	}
	return out, nil
}

func (s *Service) ListLogs(ctx context.Context, id uuid.UUID, page, limit int) ([]TransactionLog, int64, error) {
	_ = ctx
	return s.repo.ListLogs(id, page, limit)
}

func (s *Service) UploadPhoto(ctx context.Context, photoType string, data []byte, contentType string) (string, error) {
	if photoType != "entry" && photoType != "exit" {
		return "", ErrInvalidPhotoType
	}
	if s.uploader == nil {
		return "", fmt.Errorf("uploader is nil")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty file")
	}

	ext := ".jpg"
	if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
		ext = exts[0]
	}
	key := fmt.Sprintf("uploads/%s/%d_%s%s", photoType, time.Now().UnixNano(), strings.ReplaceAll(uuid.NewString(), "-", ""), ext)

	url, err := s.uploader.PutObject(ctx, key, data, contentType)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (s *Service) withTransaction(fn func(tx *gorm.DB) error) error {
	if s.db == nil {
		return fn(nil)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

func (s *Service) uploadPhotoAndUpdate(transactionID uuid.UUID, photoPath, photoType string) {
	photoPath = strings.TrimSpace(photoPath)
	if transactionID == uuid.Nil || photoPath == "" {
		return
	}

	if strings.HasPrefix(photoPath, "http://") || strings.HasPrefix(photoPath, "https://") {
		if photoType == "entry" {
			_ = s.repo.UpdateEntryPhotoURL(transactionID, photoPath)
		} else {
			_ = s.repo.UpdateExitPhotoURL(transactionID, photoPath)
		}
		return
	}

	if s.uploader == nil {
		return
	}

	body, err := os.ReadFile(photoPath)
	if err != nil {
		return
	}

	ext := strings.ToLower(filepath.Ext(photoPath))
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "image/jpeg"
		ext = ".jpg"
	}

	key := fmt.Sprintf("transactions/%s/%s_%d%s", transactionID.String(), photoType, time.Now().UnixNano(), ext)
	parentCtx := s.baseCtx
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	uploadCtx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()

	url, err := s.uploader.PutObject(uploadCtx, key, body, contentType)
	if err != nil {
		return
	}

	if photoType == "entry" {
		_ = s.repo.UpdateEntryPhotoURL(transactionID, url)
		return
	}
	_ = s.repo.UpdateExitPhotoURL(transactionID, url)
}

package ocr

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyypluto/parkieee-api/internal/sse"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
	"gorm.io/gorm"
)

type Repo interface {
	CreateJob(tx *gorm.DB, job *OCRJob) error
	FindJobByID(id uuid.UUID) (*OCRJob, error)
	FindResultByJobID(jobID uuid.UUID) (*OCRResult, error)
	CreateReviewLog(log *OCRReviewLog) error
	UpdateResult(result *OCRResult) error
	UpdateTransactionMismatch(transactionID uuid.UUID, mismatch bool) error
	UpdateJob(job *OCRJob) error
	CreateResult(result *OCRResult) error
	FindTransactionContext(transactionID uuid.UUID) (*TransactionContext, error)
	FindActiveThreshold() (float64, error)
	ListJobs(status string, page, limit int) ([]OCRJob, int64, error)
}

type Detector interface {
	Detect(ctx context.Context, imagePath string) (*LPRResponse, error)
}

type Service struct {
	repo      Repo
	detector  Detector
	hub       *pubsub.Hub
	db        *gorm.DB
	baseCtx   context.Context
	runAsync  func(fn func())
	now       func() time.Time
	operation func(ctx context.Context, imagePath string) (*LPRResponse, error)
}

var (
	ErrJobNotFound    = errors.New("ocr job not found")
	ErrResultNotFound = errors.New("ocr result not found")
)

func NewService(repo Repo, detector Detector, hub *pubsub.Hub, db *gorm.DB) *Service {
	s := &Service{
		repo:     repo,
		detector: detector,
		hub:      hub,
		db:       db,
		baseCtx:  context.Background(),
		runAsync: func(fn func()) {
			go fn()
		},
		now: time.Now,
	}
	if detector != nil {
		s.operation = detector.Detect
	} else {
		s.operation = func(context.Context, string) (*LPRResponse, error) {
			return nil, nil
		}
	}
	return s
}

func (s *Service) SetBaseContext(ctx context.Context) {
	if ctx == nil {
		s.baseCtx = context.Background()
		return
	}
	s.baseCtx = ctx
}

// EnqueueJob stores a queued OCR job after transaction commit and runs detection asynchronously.
func (s *Service) EnqueueJob(db *gorm.DB, txID uuid.UUID, photoPath, photoType string) {
	if s.repo == nil {
		slog.Error("ocr.enqueue failed", "error", "repo is nil")
		return
	}
	if txID == uuid.Nil {
		slog.Warn("ocr.enqueue skipped", "reason", "empty transaction id")
		return
	}
	photoPath = strings.TrimSpace(photoPath)
	if photoPath == "" {
		slog.Warn("ocr.enqueue skipped", "reason", "empty photo path", "tx_id", txID)
		return
	}
	if photoType != PhotoTypeEntry && photoType != PhotoTypeExit {
		slog.Warn("ocr.enqueue skipped", "reason", "invalid photo type", "tx_id", txID, "photo_type", photoType)
		return
	}

	job := &OCRJob{
		ID:            uuid.New(),
		TransactionID: txID,
		ImageURL:      photoPath,
		PhotoType:     photoType,
		Status:        JobStatusQueued,
		RetryCount:    0,
		QueuedAt:      s.now(),
	}

	if err := s.repo.CreateJob(db, job); err != nil {
		slog.Error("ocr.enqueue failed", "error", err, "tx_id", txID)
		return
	}

	s.runAsync(func() {
		s.runJob(job)
	})
}

func (s *Service) runJob(job *OCRJob) {
	if job == nil {
		return
	}

	now := s.now()
	job.Status = JobStatusProcessing
	job.StartedAt = &now
	if err := s.repo.UpdateJob(job); err != nil {
		slog.Error("ocr.job update failed", "job_id", job.ID, "error", err)
		return
	}

	if s.detector == nil {
		s.finishFailedJob(job, "detector unavailable")
		return
	}

	parentCtx := s.baseCtx
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	resp, err := s.operation(ctx, job.ImageURL)
	if err != nil {
		s.finishFailedJob(job, err.Error())
		return
	}
	if resp == nil {
		s.finishFailedJob(job, "empty detector response")
		return
	}

	txCtx, err := s.repo.FindTransactionContext(job.TransactionID)
	if err != nil {
		s.finishFailedJob(job, err.Error())
		return
	}
	if txCtx == nil {
		s.finishFailedJob(job, "transaction context not found")
		return
	}

	threshold, err := s.repo.FindActiveThreshold()
	if err != nil {
		s.finishFailedJob(job, err.Error())
		return
	}

	plateDetected := strings.TrimSpace(resp.PlateText)
	actualPlate := strings.TrimSpace(txCtx.ActualPlate)
	isMatch := actualPlate != "" && strings.EqualFold(plateDetected, actualPlate)
	autoAccept := threshold > 0 && resp.Confidence >= threshold && isMatch

	raw, _ := json.Marshal(map[string]any{
		"plate_text":  resp.PlateText,
		"confidence":  resp.Confidence,
		"image_url":   resp.ImageURL,
		"threshold":   threshold,
		"auto_accept": autoAccept,
	})

	result := &OCRResult{
		ID:            uuid.New(),
		OCRJobID:      job.ID,
		PlateDetected: plateDetected,
		ActualPlate:   actualPlate,
		IsMatch:       isMatch,
		Confidence:    resp.Confidence,
		RawOutput:     raw,
		VehicleID:     txCtx.VehicleID,
		IsVerified:    autoAccept,
		CreatedAt:     s.now(),
	}
	if err := s.repo.CreateResult(result); err != nil {
		s.finishFailedJob(job, err.Error())
		return
	}

	completedAt := s.now()
	job.Status = JobStatusCompleted
	job.CompletedAt = &completedAt
	job.ErrorMessage = ""
	job.OutputImageURL = resp.ImageURL
	if err := s.repo.UpdateJob(job); err != nil {
		slog.Error("ocr.job completion update failed", "job_id", job.ID, "error", err)
	}

	if !autoAccept {
		if err := s.repo.UpdateTransactionMismatch(job.TransactionID, true); err != nil {
			slog.Error("ocr.job mismatch update failed", "job_id", job.ID, "error", err)
		}
		if s.hub != nil {
			s.hub.Publish("cashier:events", sse.NewEventMessage("cashier.plate_mismatch", map[string]any{
				"transaction_id": job.TransactionID,
				"job_id":         job.ID,
				"detected_plate": plateDetected,
				"actual_plate":   actualPlate,
				"confidence":     resp.Confidence,
				"auto_accept":    autoAccept,
			}))
		}
		if s.db != nil {
			_ = outbox.Publish(s.db, "notify.plate_mismatch", map[string]any{
				"transaction_id": job.TransactionID,
				"job_id":         job.ID,
				"detected":       plateDetected,
				"actual":         actualPlate,
				"confidence":     resp.Confidence,
			})
		}
		slog.Warn("plate mismatch", "tx_id", job.TransactionID, "detected", plateDetected, "actual", actualPlate, "confidence", resp.Confidence)
	}

	slog.Info("ocr job complete", "tx_id", job.TransactionID, "match", isMatch, "confidence", resp.Confidence)
}

type ReviewInput struct {
	JobID       uuid.UUID
	ManualPlate string
	Match       bool
	ReviewNote  string
	ReviewerID  uuid.UUID
}

func (s *Service) ReviewJob(ctx context.Context, in ReviewInput) error {
	job, err := s.repo.FindJobByID(in.JobID)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrJobNotFound
	}

	result, err := s.repo.FindResultByJobID(in.JobID)
	if err != nil {
		return err
	}
	if result == nil {
		return ErrResultNotFound
	}

	now := s.now()
	log := &OCRReviewLog{
		ID:              uuid.New(),
		OCRResultID:     result.ID,
		OCRPlate:        result.PlateDetected,
		OCRConfidence:   result.Confidence,
		ManualPlate:     strings.TrimSpace(in.ManualPlate),
		ReviewedBy:      in.ReviewerID,
		Match:           in.Match,
		AutoMatchResult: result.IsMatch,
		ReviewNote:      strings.TrimSpace(in.ReviewNote),
		ReviewedAt:      now,
		CreatedAt:       now,
	}
	if err := s.repo.CreateReviewLog(log); err != nil {
		return err
	}

	result.IsVerified = true
	if in.ReviewerID != uuid.Nil {
		result.VerifiedBy = &in.ReviewerID
	}
	result.VerifiedAt = &now
	if err := s.repo.UpdateResult(result); err != nil {
		return err
	}

	if in.Match {
		if err := s.repo.UpdateTransactionMismatch(job.TransactionID, false); err != nil {
			return err
		}
	}

	slog.InfoContext(ctx, "ocr review submitted", "job_id", in.JobID, "match", in.Match, "reviewer", in.ReviewerID)
	return nil
}

func (s *Service) finishFailedJob(job *OCRJob, message string) {
	completedAt := s.now()
	job.Status = JobStatusFailed
	job.CompletedAt = &completedAt
	job.ErrorMessage = message
	if err := s.repo.UpdateJob(job); err != nil {
		slog.Error("ocr.job failure update failed", "job_id", job.ID, "error", err)
	}
	slog.Error("ocr job failed", "job_id", job.ID, "error", message)
}

func (s *Service) ListJobs(ctx context.Context, status string, page, limit int) ([]OCRJob, int64, error) {
	_ = ctx
	return s.repo.ListJobs(strings.TrimSpace(strings.ToLower(status)), page, limit)
}

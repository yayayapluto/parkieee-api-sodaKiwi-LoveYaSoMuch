package ocr

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeRepo struct {
	createdJobs    []*OCRJob
	updatedJobs    []*OCRJob
	createdResults []*OCRResult
	transactionCtx *TransactionContext
	threshold      float64
	jobByID        *OCRJob
	resultByJobID  *OCRResult
	reviewLogs     []*OCRReviewLog
	updatedResult  *OCRResult
	clearedMatchID *uuid.UUID
	jobs           []OCRJob
}

func (f *fakeRepo) CreateJob(_ *gorm.DB, job *OCRJob) error {
	f.createdJobs = append(f.createdJobs, job)
	return nil
}

func (f *fakeRepo) UpdateJob(job *OCRJob) error {
	f.updatedJobs = append(f.updatedJobs, job)
	return nil
}

func (f *fakeRepo) CreateResult(result *OCRResult) error {
	f.createdResults = append(f.createdResults, result)
	return nil
}

func (f *fakeRepo) FindTransactionContext(uuid.UUID) (*TransactionContext, error) {
	return f.transactionCtx, nil
}

func (f *fakeRepo) FindActiveThreshold() (float64, error) {
	return f.threshold, nil
}

func (f *fakeRepo) FindJobByID(uuid.UUID) (*OCRJob, error) {
	return f.jobByID, nil
}

func (f *fakeRepo) FindResultByJobID(uuid.UUID) (*OCRResult, error) {
	return f.resultByJobID, nil
}

func (f *fakeRepo) CreateReviewLog(log *OCRReviewLog) error {
	f.reviewLogs = append(f.reviewLogs, log)
	return nil
}

func (f *fakeRepo) UpdateResult(result *OCRResult) error {
	f.updatedResult = result
	return nil
}

func (f *fakeRepo) UpdateTransactionMismatch(transactionID uuid.UUID, mismatch bool) error {
	if !mismatch {
		f.clearedMatchID = &transactionID
	}
	return nil
}

func (f *fakeRepo) ListJobs(status string, _, _ int) ([]OCRJob, int64, error) {
	if status == "" {
		return f.jobs, int64(len(f.jobs)), nil
	}
	out := make([]OCRJob, 0, len(f.jobs))
	for _, row := range f.jobs {
		if row.Status == status {
			out = append(out, row)
		}
	}
	return out, int64(len(out)), nil
}

type fakeDetector struct {
	called int
	path   string
}

func (f *fakeDetector) Detect(_ context.Context, imagePath string) (*LPRResponse, error) {
	f.called++
	f.path = imagePath
	return &LPRResponse{PlateText: "B1234CD", Confidence: 0.93}, nil
}

func TestService_EnqueueJob_CreatesQueuedJobAndRunsDetector(t *testing.T) {
	repo := &fakeRepo{
		transactionCtx: &TransactionContext{TransactionID: uuid.New(), ActualPlate: "B1234CD"},
		threshold:      0.8,
	}
	detector := &fakeDetector{}
	svc := NewService(repo, detector, nil, nil)

	svc.now = func() time.Time {
		return time.Date(2026, 4, 15, 9, 0, 0, 0, time.UTC)
	}
	svc.runAsync = func(fn func()) { fn() }

	txID := uuid.New()
	repo.transactionCtx.TransactionID = txID
	svc.EnqueueJob(nil, txID, "/tmp/entry.jpg", PhotoTypeEntry)

	require.Len(t, repo.createdJobs, 1)
	job := repo.createdJobs[0]
	assert.Equal(t, txID, job.TransactionID)
	assert.Equal(t, JobStatusCompleted, job.Status)
	assert.Equal(t, "/tmp/entry.jpg", job.ImageURL)
	assert.Equal(t, PhotoTypeEntry, job.PhotoType)
	assert.Equal(t, 1, detector.called)
	assert.Equal(t, "/tmp/entry.jpg", detector.path)
	assert.GreaterOrEqual(t, len(repo.updatedJobs), 2)
	require.Len(t, repo.createdResults, 1)
}

func TestService_EnqueueJob_InvalidInputSkipsCreate(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, nil, nil, nil)
	svc.runAsync = func(fn func()) { fn() }

	svc.EnqueueJob(nil, uuid.Nil, "/tmp/a.jpg", PhotoTypeEntry)
	svc.EnqueueJob(nil, uuid.New(), "", PhotoTypeEntry)
	svc.EnqueueJob(nil, uuid.New(), "/tmp/a.jpg", "other")

	assert.Len(t, repo.createdJobs, 0)
}

func TestService_ReviewJob_VerifiesResultAndClearsMismatch(t *testing.T) {
	txID := uuid.New()
	reviewerID := uuid.New()
	repo := &fakeRepo{
		transactionCtx: &TransactionContext{TransactionID: txID},
		jobByID:        &OCRJob{ID: uuid.New(), TransactionID: txID},
		resultByJobID: &OCRResult{
			ID:            uuid.New(),
			PlateDetected: "B1234CD",
			Confidence:    0.73,
			IsMatch:       false,
		},
	}

	svc := NewService(repo, nil, nil, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 4, 15, 9, 30, 0, 0, time.UTC)
	}

	err := svc.ReviewJob(context.Background(), ReviewInput{
		JobID:       repo.jobByID.ID,
		ManualPlate: "B1234CD",
		Match:       true,
		ReviewNote:  "verified by cashier",
		ReviewerID:  reviewerID,
	})
	require.NoError(t, err)
	require.Len(t, repo.reviewLogs, 1)
	require.NotNil(t, repo.updatedResult)
	assert.True(t, repo.updatedResult.IsVerified)
	assert.NotNil(t, repo.updatedResult.VerifiedAt)
	require.NotNil(t, repo.updatedResult.VerifiedBy)
	assert.Equal(t, reviewerID, *repo.updatedResult.VerifiedBy)
	require.NotNil(t, repo.clearedMatchID)
	assert.Equal(t, txID, *repo.clearedMatchID)
}

func TestService_ListJobs_FilterStatus(t *testing.T) {
	repo := &fakeRepo{jobs: []OCRJob{{ID: uuid.New(), Status: JobStatusQueued}, {ID: uuid.New(), Status: JobStatusCompleted}}}
	svc := NewService(repo, nil, nil, nil)

	rows, total, err := svc.ListJobs(context.Background(), JobStatusQueued, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, JobStatusQueued, rows[0].Status)
}

func TestService_EnqueueJob_UsesBaseContextCancellation(t *testing.T) {
	repo := &fakeRepo{
		transactionCtx: &TransactionContext{TransactionID: uuid.New(), ActualPlate: "B1234CD"},
		threshold:      0.8,
	}
	svc := NewService(repo, nil, nil, nil)
	svc.runAsync = func(fn func()) { fn() }

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	svc.SetBaseContext(canceledCtx)
	svc.operation = func(ctx context.Context, _ string) (*LPRResponse, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	txID := uuid.New()
	repo.transactionCtx.TransactionID = txID
	svc.EnqueueJob(nil, txID, "/tmp/entry.jpg", PhotoTypeEntry)

	require.Len(t, repo.createdJobs, 1)
	assert.Equal(t, JobStatusFailed, repo.createdJobs[0].Status)
	assert.Len(t, repo.createdResults, 0)
}

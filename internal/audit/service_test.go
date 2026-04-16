package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/internal/audit"
	auditmocks "github.com/yyypluto/parkieee-api/internal/audit/mocks"
)

func TestSearch_ReturnsResults(t *testing.T) {
	repo := auditmocks.NewRepo(t)
	svc := audit.NewService(repo)

	before := time.Now()
	expected := []audit.AuditLog{{Action: "transaction.entry", EntityType: "transaction"}}
	repo.On("Search", "transaction", before, 10).Return(expected, nil)

	got, err := svc.Search(context.Background(), "transaction", before, 10)

	assert.NoError(t, err)
	assert.Equal(t, expected, got)
	repo.AssertExpectations(t)
}

func TestSearch_EmptyQuery_ReturnsAll(t *testing.T) {
	repo := auditmocks.NewRepo(t)
	svc := audit.NewService(repo)

	zero := time.Time{}
	repo.On("Search", "", zero, 50).Return([]audit.AuditLog{}, nil)

	got, err := svc.Search(context.Background(), "", zero, 50)

	assert.NoError(t, err)
	assert.Empty(t, got)
	repo.AssertExpectations(t)
}

func TestSearch_RepoError_Propagates(t *testing.T) {
	repo := auditmocks.NewRepo(t)
	svc := audit.NewService(repo)

	zero := time.Time{}
	repo.On("Search", "fail", zero, 50).Return(nil, errors.New("db error"))

	_, err := svc.Search(context.Background(), "fail", zero, 50)

	assert.EqualError(t, err, "db error")
	repo.AssertExpectations(t)
}

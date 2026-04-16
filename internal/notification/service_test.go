package notification_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/yyypluto/parkieee-api/internal/notification"
	notificationmocks "github.com/yyypluto/parkieee-api/internal/notification/mocks"
)

func TestList_ReturnsPaginatedRows(t *testing.T) {
	mockRepo := notificationmocks.NewRepo(t)
	userID := uuid.New()
	expected := []notification.Notification{{ID: uuid.New(), Type: "plate_mismatch"}}
	mockRepo.On("FindByUserID", &userID, 1, 20).Return(expected, int64(1), nil)

	svc := notification.NewService(mockRepo)
	rows, total, err := svc.List(context.Background(), &userID, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, expected, rows)
}

func TestMarkRead_DelegatesToRepo(t *testing.T) {
	mockRepo := notificationmocks.NewRepo(t)
	id := uuid.New()
	mockRepo.On("MarkRead", id).Return(nil)

	svc := notification.NewService(mockRepo)
	err := svc.MarkRead(context.Background(), id)

	assert.NoError(t, err)
}

func TestMarkAllRead_DelegatesToRepo(t *testing.T) {
	mockRepo := notificationmocks.NewRepo(t)
	userID := uuid.New()
	mockRepo.On("MarkAllRead", &userID).Return(nil)

	svc := notification.NewService(mockRepo)
	err := svc.MarkAllRead(context.Background(), &userID)

	assert.NoError(t, err)
}

func TestList_RepoError_Propagates(t *testing.T) {
	mockRepo := notificationmocks.NewRepo(t)
	userID := uuid.New()
	mockRepo.On("FindByUserID", &userID, 1, 20).Return(nil, int64(0), errors.New("db error"))

	svc := notification.NewService(mockRepo)
	_, _, err := svc.List(context.Background(), &userID, 1, 20)

	assert.EqualError(t, err, "db error")
}

func TestUnreadCount_DelegatesToRepo(t *testing.T) {
	mockRepo := notificationmocks.NewRepo(t)
	userID := uuid.New()
	mockRepo.On("CountUnread", &userID).Return(int64(5), nil)

	svc := notification.NewService(mockRepo)
	total, err := svc.UnreadCount(context.Background(), &userID)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
}

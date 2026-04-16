package report_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yyypluto/parkieee-api/internal/report"
	reportmocks "github.com/yyypluto/parkieee-api/internal/report/mocks"
)

func TestGetRevenue_ReturnsList(t *testing.T) {
	repo := reportmocks.NewRepo(t)
	svc := report.NewService(repo)

	zoneID := uuid.New()
	expected := []report.DailyRevenue{{ZoneID: zoneID, TransactionCount: 5, TotalRevenue: 25000}}
	repo.On("DailyRevenue", time.Time{}, time.Time{}, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return(expected, nil)

	rows, err := svc.GetRevenue(context.Background(), report.RevenueFilter{})
	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, 25000, rows[0].TotalRevenue)
	repo.AssertExpectations(t)
}

func TestGetOccupancy_ReturnsList(t *testing.T) {
	repo := reportmocks.NewRepo(t)
	svc := report.NewService(repo)

	expected := []report.ZoneOccupancy{{ZoneName: "Zone A", Capacity: 50, Occupied: 12}}
	repo.On("ZoneOccupancy").Return(expected, nil)

	rows, err := svc.GetOccupancy(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, "Zone A", rows[0].ZoneName)
	repo.AssertExpectations(t)
}

func TestGetRevenue_RepoError_Propagates(t *testing.T) {
	repo := reportmocks.NewRepo(t)
	svc := report.NewService(repo)

	repo.On("DailyRevenue", time.Time{}, time.Time{}, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return(nil, errors.New("db error"))

	_, err := svc.GetRevenue(context.Background(), report.RevenueFilter{})
	assert.EqualError(t, err, "db error")
	repo.AssertExpectations(t)
}

func TestExportRevenueCSV_ContainsHeader(t *testing.T) {
	repo := reportmocks.NewRepo(t)
	svc := report.NewService(repo)

	repo.On("DailyRevenue", time.Time{}, time.Time{}, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return([]report.DailyRevenue{}, nil)

	data, err := svc.ExportRevenueCSV(context.Background(), report.RevenueFilter{})
	assert.NoError(t, err)
	assert.Contains(t, string(data), "day,zone_id,vehicle_type_id,transaction_count,total_revenue")
}

package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Service struct{ repo Repo }

func NewService(repo Repo) *Service { return &Service{repo: repo} }

func (s *Service) GetRevenue(ctx context.Context, f RevenueFilter) ([]DailyRevenue, error) {
	_ = ctx
	return s.repo.DailyRevenue(f.From, f.To, f.ZoneID, f.VehicleTypeID)
}

func (s *Service) GetOccupancy(ctx context.Context) ([]ZoneOccupancy, error) {
	_ = ctx
	return s.repo.ZoneOccupancy()
}

func (s *Service) ExportRevenueCSV(ctx context.Context, f RevenueFilter) ([]byte, error) {
	rows, err := s.GetRevenue(ctx, f)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	_ = w.Write([]string{"day", "zone_id", "vehicle_type_id", "transaction_count", "total_revenue"})
	for _, r := range rows {
		_ = w.Write([]string{
			r.Day.Format("2006-01-02"),
			r.ZoneID.String(),
			r.VehicleTypeID.String(),
			strconv.Itoa(r.TransactionCount),
			strconv.Itoa(r.TotalRevenue),
		})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// revenueFilterFromStrings parses URL query params into a RevenueFilter.
func revenueFilterFromStrings(fromStr, toStr, zoneStr, vtStr string) RevenueFilter {
	f := RevenueFilter{}
	if fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			f.From = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			f.To = t
		}
	}
	if zoneStr != "" {
		if id, err := uuid.Parse(zoneStr); err == nil {
			f.ZoneID = &id
		}
	}
	if vtStr != "" {
		if id, err := uuid.Parse(vtStr); err == nil {
			f.VehicleTypeID = &id
		}
	}
	return f
}

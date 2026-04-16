package ui

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/yyypluto/parkieee-api/internal/auth"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"github.com/yyypluto/parkieee-api/internal/payment"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"github.com/yyypluto/parkieee-api/internal/user"
	"github.com/yyypluto/parkieee-api/internal/vehicle"
	"github.com/yyypluto/parkieee-api/internal/zone"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RenderDashboard(c *fiber.Ctx) error {
	var userCount, vehicleCount, txCount, pendingTxCount int64
	var revenue int64

	h.db.Model(&auth.User{}).Count(&userCount)
	h.db.Model(&vehicle.Vehicle{}).Count(&vehicleCount)
	h.db.Model(&transaction.Transaction{}).Count(&txCount)
	h.db.Model(&transaction.Transaction{}).Where("status = ?", "pending_payment").Count(&pendingTxCount)
	
	h.db.Model(&payment.Payment{}).
		Where("status = ?", "completed").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&revenue)

	return c.Render("dashboard", fiber.Map{
		"UserCount":       userCount,
		"VehicleCount":    vehicleCount,
		"TotalTx":         txCount,
		"PendingTx":       pendingTxCount,
		"TotalRevenue":    revenue,
	}, "layouts/main")
}

func (h *Handler) RenderUsers(c *fiber.Ctx) error {
	var users []user.User
	// Assuming User belongs to Role via RoleID
	h.db.Order("created_at desc").Find(&users)
	return c.Render("users", fiber.Map{"Users": users}, "layouts/main")
}

func (h *Handler) RenderVehicles(c *fiber.Ctx) error {
	var vehicles []vehicle.Vehicle
	h.db.Preload("VehicleType").Preload("RFIDCard").Order("created_at desc").Limit(300).Find(&vehicles)
	return c.Render("vehicles", fiber.Map{"Vehicles": vehicles}, "layouts/main")
}

func (h *Handler) RenderZones(c *fiber.Ctx) error {
	var zones []zone.Zone
	h.db.Preload("Gates", func(db *gorm.DB) *gorm.DB {
		return db.Order("type asc")
	}).Find(&zones)
	return c.Render("zones", fiber.Map{"Zones": zones}, "layouts/main")
}

func (h *Handler) RenderFees(c *fiber.Ctx) error {
	var typeList []vehicle.VehicleType
	h.db.Find(&typeList)
	
	var configs []fee.FeeConfig
	h.db.Preload("Tiers").Find(&configs)
	return c.Render("fees", fiber.Map{
		"Configs": configs,
		"VehicleTypes": typeList,
	}, "layouts/main")
}

func (h *Handler) RenderTransactions(c *fiber.Ctx) error {
	var txs []transaction.Transaction
	h.db.Order("created_at desc").Limit(300).Find(&txs)
	
	type txView struct {
		ID        string
		Code      string
		Method    string
		EntryTime string
		ExitTime  string
		Status    string
		Fee       int
	}
	var txList []txView
	for _, t := range txs {
		exitTime := "-"
		if t.ExitAt != nil {
			exitTime = t.ExitAt.Format("02 Jan 15:04:05")
		}
		
		method := t.EntryMethod
		if t.TicketCode != nil {
			method = "Ticket"
		} else if t.RFIDCardID != nil {
			method = "RFID"
		}

		txList = append(txList, txView{
			ID:        t.ID.String()[:8],
			Code:      t.TransactionCode,
			Method:    method,
			EntryTime: t.EntryAt.Format("02 Jan 15:04:05"),
			ExitTime:  exitTime,
			Status:    t.Status,
			Fee:       t.CalculatedFee,
		})
	}
	
	return c.Render("transactions", fiber.Map{"Transactions": txList}, "layouts/main")
}

package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/yyypluto/parkieee-api/config"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"github.com/yyypluto/parkieee-api/internal/user"
	"github.com/yyypluto/parkieee-api/internal/vehicle"
	"github.com/yyypluto/parkieee-api/internal/zone"
	"gopkg.in/telebot.v3"
	"gorm.io/gorm"
)

type TelegramBot struct {
	bot    *telebot.Bot
	db     *gorm.DB
	log    *slog.Logger
	cfg    *config.Config
	admins map[int64]bool
}

func NewTelegramBot(db *gorm.DB, log *slog.Logger, cfg *config.Config) (*TelegramBot, error) {
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is empty")
	}

	pref := telebot.Settings{
		Token:  cfg.TelegramBotToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return nil, err
	}

	admins := make(map[int64]bool)
	if cfg.TelegramAdminIDs != "" {
		ids := strings.Split(cfg.TelegramAdminIDs, ",")
		for _, idStr := range ids {
			id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
			if err == nil {
				admins[id] = true
			}
		}
	}

	tb := &TelegramBot{
		bot:    b,
		db:     db,
		log:    log,
		cfg:    cfg,
		admins: admins,
	}

	tb.setupRoutes()
	return tb, nil
}

func (tb *TelegramBot) Start(ctx context.Context) {
	tb.log.Info("Starting Telegram Bot poller")
	go tb.bot.Start()

	<-ctx.Done()
	tb.log.Info("Stopping Telegram Bot poller")
	tb.bot.Stop()
}

func (tb *TelegramBot) authMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		userID := c.Sender().ID
		if len(tb.admins) > 0 && !tb.admins[userID] {
			tb.log.Warn("Unauthorized Telegram access attempt", "user_id", userID)
			return c.Send("❌ Unauthorized. Your ID: " + fmt.Sprintf("%d", userID))
		}
		return next(c)
	}
}

func (tb *TelegramBot) setupRoutes() {
	tb.bot.Use(tb.authMiddleware)

	tb.bot.Handle("/start", tb.handleStart)
	tb.bot.Handle("/stats", tb.handleStats)
	tb.bot.Handle("/txs", tb.handleTxs)
	tb.bot.Handle("/zones", tb.handleZones)
	tb.bot.Handle("/users", tb.handleUsers)
}

func (tb *TelegramBot) handleStart(c telebot.Context) error {
	msg := "🚗 *Parkieee API Bot*\n\n" +
		"Available commands:\n" +
		"/stats - Global Platform Metrics\n" +
		"/txs - Last 5 Transactions\n" +
		"/zones - Zones and Gate Status\n" +
		"/users - Admin Users List"
	return c.Send(msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (tb *TelegramBot) handleStats(c telebot.Context) error {
	var userCount, vehicleCount, totalTx, pendingTx int64
	tb.db.Model(&user.User{}).Count(&userCount)
	tb.db.Model(&vehicle.Vehicle{}).Count(&vehicleCount)
	tb.db.Model(&transaction.Transaction{}).Count(&totalTx)
	tb.db.Model(&transaction.Transaction{}).Where("status = ?", "pending").Count(&pendingTx)

	type Result struct {
		Total int
	}
	var res Result
	tb.db.Model(&transaction.Transaction{}).Where("status = ?", "completed").Select("COALESCE(SUM(calculated_fee), 0) as total").Scan(&res)

	msg := fmt.Sprintf("📊 *Platform Metrics*\n\n👥 Users: %d\n🚗 Vehicles: %d\n\n💳 *Revenue*: Rp %d\n📈 Total TXs: %d\n⏳ Pending TXs: %d",
		userCount, vehicleCount, res.Total, totalTx, pendingTx)

	return c.Send(msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (tb *TelegramBot) handleTxs(c telebot.Context) error {
	var txs []transaction.Transaction
	tb.db.Order("created_at desc").Limit(5).Find(&txs)

	if len(txs) == 0 {
		return c.Send("No transactions found.")
	}

	msg := "📝 *Last 5 Transactions*\n\n"
	for _, t := range txs {
		fee := t.CalculatedFee
		status := "🟢"
		if t.Status == "pending" {
			status = "🟡"
		}

		method := t.EntryMethod
		if t.TicketCode != nil {
			method = "Ticket"
		} else if t.RFIDCardID != nil {
			method = "RFID"
		}

		msg += fmt.Sprintf("%s `%s` | *%s*\n↳ Fee: Rp%d | %s\n\n",
			status,
			t.TransactionCode,
			method,
			fee,
			t.EntryAt.Format("15:04:05"),
		)
	}

	return c.Send(msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (tb *TelegramBot) handleZones(c telebot.Context) error {
	var zones []zone.Zone
	tb.db.Find(&zones)

	if len(zones) == 0 {
		return c.Send("No zones configured.")
	}

	msg := "🗺️ *Zones & Gates*\n\n"
	for _, z := range zones {
		msg += fmt.Sprintf("📍 *%s* (Cap: %d)\n", z.Name, z.Capacity)

		var gates []zone.Gate
		tb.db.Where("zone_id = ?", z.ID).Find(&gates)
		for _, g := range gates {
			status := "✅"
			if !g.IsActive {
				status = "❌"
			}
			msg += fmt.Sprintf("  ↳ %s %s [%s]\n", status, g.Name, g.GateType)
		}
		msg += "\n"
	}

	return c.Send(msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (tb *TelegramBot) handleUsers(c telebot.Context) error {
	var users []user.User
	tb.db.Limit(10).Find(&users)

	msg := "👥 *Users List*\n\n"
	for _, u := range users {
		active := "✅"
		if !u.IsActive {
			active = "❌"
		}
		msg += fmt.Sprintf("%s *%s* (`%s`)\n", active, u.Name, u.Email)
	}

	return c.Send(msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

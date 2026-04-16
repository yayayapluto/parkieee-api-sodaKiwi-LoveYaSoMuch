// cmd/api/main.go
package main

import (
	"context"
	"log"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/gofiber/template/handlebars/v2"

	"github.com/yyypluto/parkieee-api/config"
	"github.com/yyypluto/parkieee-api/internal/audit"
	"github.com/yyypluto/parkieee-api/internal/auth"
	"github.com/yyypluto/parkieee-api/internal/bot"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"github.com/yyypluto/parkieee-api/internal/notification"
	"github.com/yyypluto/parkieee-api/internal/ocr"
	"github.com/yyypluto/parkieee-api/internal/override"
	"github.com/yyypluto/parkieee-api/internal/payment"
	"github.com/yyypluto/parkieee-api/internal/report"
	"github.com/yyypluto/parkieee-api/internal/sse"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"github.com/yyypluto/parkieee-api/internal/ui"
	"github.com/yyypluto/parkieee-api/internal/user"
	"github.com/yyypluto/parkieee-api/internal/vehicle"
	"github.com/yyypluto/parkieee-api/internal/worker"
	"github.com/yyypluto/parkieee-api/internal/zone"
	"github.com/yyypluto/parkieee-api/pkg/applog"
	"github.com/yyypluto/parkieee-api/pkg/database"
	"github.com/yyypluto/parkieee-api/pkg/logger"
	"github.com/yyypluto/parkieee-api/pkg/middleware"
	"github.com/yyypluto/parkieee-api/pkg/midtrans"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
	"github.com/yyypluto/parkieee-api/pkg/pubsub"
	"github.com/yyypluto/parkieee-api/pkg/ratelimit"
	"github.com/yyypluto/parkieee-api/pkg/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger.Init(cfg.LogLevel, cfg.LogFormat)
	slog := logger.Get()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		log.Fatal(err)
	}

	// AutoMigrate all models
	models := append([]any{}, auth.AllModels()...)
	models = append(models, applog.AllModels()...)
	models = append(models, audit.AllModels()...)
	models = append(models, user.AllModels()...)
	models = append(models, vehicle.AllModels()...)
	models = append(models, zone.AllModels()...)
	models = append(models, fee.AllModels()...)
	models = append(models, ocr.AllModels()...)
	models = append(models, transaction.AllModels()...)
	models = append(models, payment.AllModels()...)
	models = append(models, notification.AllModels()...)
	models = append(models, override.AllModels()...)
	models = append(models, outbox.AllModels()...)
	if err := database.Migrate(db, models...); err != nil {
		slog.Error("database migration failed", "error", err)
		log.Fatal(err)
	}

	// Run SQL migration files (materialized views, FTS indexes)
	migrationsDir := filepath.Join("pkg", "database", "migrations")
	if err := database.RunMigrationFiles(db, migrationsDir); err != nil {
		slog.Error("sql migrations failed", "error", err)
		log.Fatal(err)
	}

	logger.AddDBHandler(db)
	slog = logger.Get()

	// Infrastructure
	hub := pubsub.NewHub()

	// Start background workers
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	auditRepo := audit.NewRepository(db)
	notifRepo := notification.NewRepository(db)
	outboxDone := outbox.Start(ctx, db, map[string]outbox.HandlerFunc{
		"audit.transaction.entry":    audit.HandleEventFn(auditRepo, "transaction.entry", "transaction"),
		"audit.transaction.exit":     audit.HandleEventFn(auditRepo, "transaction.exit", "transaction"),
		"audit.payment.completed":    audit.HandleEventFn(auditRepo, "payment.completed", "payment"),
		"notify.plate_mismatch":      notification.HandleEventFn(notifRepo, "plate_mismatch", "Plate Mismatch Detected"),
		"notify.unclosed_flag":       notification.HandleEventFn(notifRepo, "unclosed_flag", "Unclosed Transaction Flagged"),
		"notify.override_escalation": notification.HandleEventFn(notifRepo, "override_escalation", "Override Limit Exceeded"),
	})

	// Read unclosed check interval from system_configs
	var sysCfg struct{ Value string }
	intervalMinutes := 60
	if err := db.Table("system_configs").Where("key = ?", "unclosed_check_interval_minutes").First(&sysCfg).Error; err == nil {
		if n, err := strconv.Atoi(sysCfg.Value); err == nil && n > 0 {
			intervalMinutes = n
		}
	}
	unclosedDone := worker.StartUnclosedChecker(ctx, db, intervalMinutes)
	viewRefresherDone := worker.StartViewRefresher(ctx, db)

	// Fiber app
	engine := handlebars.New("./views", ".hbs")

	app := fiber.New(fiber.Config{
		Views: engine,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			errCode := "INTERNAL_ERROR"
			message := err.Error()

			if fe, ok := err.(*fiber.Error); ok {
				status = fe.Code
				message = fe.Message
				switch fe.Code {
				case fiber.StatusBadRequest:
					errCode = "BAD_REQUEST"
				case fiber.StatusUnauthorized:
					errCode = "UNAUTHORIZED"
				case fiber.StatusForbidden:
					errCode = "FORBIDDEN"
				case fiber.StatusNotFound:
					errCode = "NOT_FOUND"
				default:
					errCode = "HTTP_ERROR"
				}
			}

			return c.Status(status).JSON(fiber.Map{
				"success": false, "data": nil, "meta": fiber.Map{},
				"error": fiber.Map{"code": errCode, "message": message},
			})
		},
	})
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-CSRF-Token",
	}))
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLogger())

	// Auth rate limiter: 10 req/min per IP
	authLimiter := ratelimit.NewLimiter(10, 10)

	// Wire auth module
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, cfg.JWTSecret, cfg.JWTExpMinutes)
	authHandler := auth.NewHandler(authSvc, auth.CookieConfig{
		Domain:   cfg.CookieDomain,
		Secure:   cfg.CookieSecure,
		SameSite: cfg.CookieSameSite,
	})
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userSvc.SetPasswordHasher(authSvc.HashPassword)
	userHandler := user.NewHandler(userSvc)
	vehicleRepo := vehicle.NewRepository(db)
	vehicleSvc := vehicle.NewService(vehicleRepo)
	vehicleHandler := vehicle.NewHandler(vehicleSvc)
	zoneRepo := zone.NewRepository(db)
	zoneSvc := zone.NewService(zoneRepo)
	zoneHandler := zone.NewHandler(zoneSvc)
	feeRepo := fee.NewRepository(db)
	feeSvc := fee.NewService(feeRepo)
	feeHandler := fee.NewHandler(feeSvc)
	ocrRepo := ocr.NewRepository(db)
	ocrClient := ocr.NewLPRClient(cfg.LPRServiceURL)
	ocrSvc := ocr.NewService(ocrRepo, ocrClient, hub, db)
	ocrSvc.SetBaseContext(ctx)
	ocrHandler := ocr.NewHandler(ocrSvc)
	txRepo := transaction.NewRepository(db)
	storageClient := storage.NewClient(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region, cfg.S3PublicBaseURL)
	txSvc := transaction.NewService(txRepo, db, ocrSvc, storageClient)
	txSvc.SetBaseContext(ctx)
	txHandler := transaction.NewHandler(txSvc)
	payRepo := payment.NewRepository(db)
	midtransClient := midtrans.NewClient(cfg.MidtransServerKey, cfg.MidtransClientKey, cfg.MidtransEnv)
	paySvc := payment.NewService(payRepo, midtransClient, hub, db)
	payHandler := payment.NewHandler(paySvc)
	sseHandler := sse.NewHandler(hub)

	v1 := app.Group("/api/v1")
	v1.Use(middleware.RateLimiter(authLimiter, func(c *fiber.Ctx) string { return c.IP() }))
	if cfg.UseCSRF {
		v1.Use(middleware.CSRFMiddleware())
	} else {
		slog.Warn("csrf middleware disabled via USE_CSRF")
	}

	auth.RegisterRoutes(v1, authHandler, cfg.JWTSecret)
	user.RegisterRoutes(v1, userHandler, cfg.JWTSecret)
	vehicle.RegisterRoutes(v1, vehicleHandler, cfg.JWTSecret)
	zone.RegisterRoutes(v1, zoneHandler, cfg.JWTSecret)
	fee.RegisterRoutes(v1, feeHandler, cfg.JWTSecret)
	ocr.RegisterRoutes(v1, ocrHandler, cfg.JWTSecret)
	transaction.RegisterRoutes(v1, txHandler, cfg.JWTSecret)
	payment.RegisterRoutes(v1, payHandler, cfg.JWTSecret)
	sse.RegisterRoutes(v1, sseHandler, db, cfg.JWTSecret)
	notifSvc := notification.NewService(notifRepo)
	notifHandler := notification.NewHandler(notifSvc)
	notification.RegisterRoutes(v1, notifHandler, cfg.JWTSecret)
	overrideRepo := override.NewRepository(db)
	overrideSvc := override.NewService(overrideRepo, db, hub)
	overrideHandler := override.NewHandler(overrideSvc)
	override.RegisterRoutes(v1, overrideHandler, cfg.JWTSecret)
	auditSvc := audit.NewService(auditRepo)
	auditHandler := audit.NewHandler(auditSvc)
	audit.RegisterRoutes(v1, auditHandler, cfg.JWTSecret)
	reportRepo := report.NewRepository(db)
	reportSvc := report.NewService(reportRepo)
	reportHandler := report.NewHandler(reportSvc)
	report.RegisterRoutes(v1, reportHandler, cfg.JWTSecret)

	uiHandler := ui.NewHandler(db)
	app.Get("/ui", func(c *fiber.Ctx) error { return c.Redirect("/ui/dashboard") })
	app.Get("/ui/dashboard", uiHandler.RenderDashboard)
	app.Get("/ui/transactions", uiHandler.RenderTransactions)
	app.Get("/ui/vehicles", uiHandler.RenderVehicles)
	app.Get("/ui/zones", uiHandler.RenderZones)
	app.Get("/ui/fees", uiHandler.RenderFees)
	app.Get("/ui/users", uiHandler.RenderUsers)

	// Setup Telegram Bot
	var telegramDone <-chan struct{}
	if cfg.TelegramBotToken != "" {
		tgBot, err := bot.NewTelegramBot(db, slog, cfg)
		if err != nil {
			slog.Error("Failed to initialize Telegram Bot", "error", err)
		} else {
			done := make(chan struct{})
			telegramDone = done
			go func() {
				defer close(done)
				tgBot.Start(ctx)
			}()
		}
	}

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	var shutdownRequested atomic.Bool
	go func() {
		<-ctx.Done()
		shutdownRequested.Store(true)
		slog.Info("shutting down server gracefully")
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			slog.Error("server shutdown failed", "error", err)
		}
	}()

	waitForStop := func(name string, done <-chan struct{}) {
		if done == nil {
			return
		}

		select {
		case <-done:
			slog.Info("worker stopped", "name", name)
		case <-time.After(5 * time.Second):
			slog.Warn("worker stop timeout", "name", name)
		}
	}

	slog.Info("server starting", "port", cfg.Port)
	listenErr := app.Listen(":" + cfg.Port)
	if listenErr != nil && !shutdownRequested.Load() {
		slog.Error("server error", "error", listenErr)
		stop()
	}

	waitForStop("outbox", outboxDone)
	waitForStop("unclosed_checker", unclosedDone)
	waitForStop("view_refresher", viewRefresherDone)
	waitForStop("telegram_bot", telegramDone)

	sqlDB, sqlErr := db.DB()
	if sqlErr != nil {
		slog.Error("database sql handle unavailable", "error", sqlErr)
	} else if err := sqlDB.Close(); err != nil {
		slog.Error("database close failed", "error", err)
	}

	if listenErr != nil && !shutdownRequested.Load() {
		log.Fatal(listenErr)
	}
	if shutdownRequested.Load() {
		slog.Info("server stopped")
	}
}

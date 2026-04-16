// cmd/seed/main.go
package main

import (
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/yyypluto/parkieee-api/config"
	"github.com/yyypluto/parkieee-api/internal/audit"
	"github.com/yyypluto/parkieee-api/internal/auth"
	"github.com/yyypluto/parkieee-api/internal/fee"
	"github.com/yyypluto/parkieee-api/internal/notification"
	"github.com/yyypluto/parkieee-api/internal/ocr"
	"github.com/yyypluto/parkieee-api/internal/override"
	"github.com/yyypluto/parkieee-api/internal/payment"
	"github.com/yyypluto/parkieee-api/internal/transaction"
	"github.com/yyypluto/parkieee-api/internal/user"
	"github.com/yyypluto/parkieee-api/internal/vehicle"
	"github.com/yyypluto/parkieee-api/internal/zone"
	"github.com/yyypluto/parkieee-api/pkg/applog"
	"github.com/yyypluto/parkieee-api/pkg/database"
	"github.com/yyypluto/parkieee-api/pkg/outbox"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Migrate all currently registered module models
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
		log.Fatal(err)
	}

	// Seed roles
	roles := []auth.Role{
		{ID: mustUUID("00000001-0000-0000-0000-000000000001"), Name: "admin", Description: "Full system access"},
		{ID: mustUUID("00000001-0000-0000-0000-000000000002"), Name: "petugas", Description: "Gate officer"},
		{ID: mustUUID("00000001-0000-0000-0000-000000000003"), Name: "owner", Description: "Read-only reports"},
	}
	for _, r := range roles {
		var existing auth.Role
		err := db.Where("name = ?", r.Name).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&r).Error; err != nil {
				log.Fatal(err)
			}
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	// Seed permissions
	nodes := []string{
		"transaction:write", "transaction:read",
		"report:read", "user:manage",
		"fee:write", "fee:read",
		"zone:write", "zone:read",
		"override:write", "override:read",
		"audit:read", "notification:read",
	}
	perms := make(map[string]auth.Permission)
	for _, node := range nodes {
		var p auth.Permission
		err := db.Where("node = ?", node).First(&p).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p = auth.Permission{ID: uuid.New(), Node: node, CreatedAt: time.Now()}
			if err := db.Create(&p).Error; err != nil {
				log.Fatal(err)
			}
		} else if err != nil {
			log.Fatal(err)
		}
		perms[node] = p
	}

	// Admin role gets all permissions
	var adminRole auth.Role
	if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		log.Fatal(err)
	}
	for _, p := range perms {
		var existing auth.RolePermission
		err := db.Where("role_id = ? AND permission_id = ?", adminRole.ID, p.ID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rp := auth.RolePermission{
				ID: uuid.New(), RoleID: adminRole.ID,
				PermissionID: p.ID, GrantedAt: time.Now(),
			}
			if err := db.Create(&rp).Error; err != nil {
				log.Fatal(err)
			}
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	// Petugas gets transaction + notification permissions
	var petugasRole auth.Role
	if err := db.Where("name = ?", "petugas").First(&petugasRole).Error; err != nil {
		log.Fatal(err)
	}
	for _, node := range []string{"transaction:write", "transaction:read", "notification:read", "override:write"} {
		if p, ok := perms[node]; ok {
			var existing auth.RolePermission
			err := db.Where("role_id = ? AND permission_id = ?", petugasRole.ID, p.ID).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				rp := auth.RolePermission{ID: uuid.New(), RoleID: petugasRole.ID, PermissionID: p.ID, GrantedAt: time.Now()}
				if err := db.Create(&rp).Error; err != nil {
					log.Fatal(err)
				}
				continue
			}
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Owner gets report + notification permissions
	var ownerRole auth.Role
	if err := db.Where("name = ?", "owner").First(&ownerRole).Error; err != nil {
		log.Fatal(err)
	}
	for _, node := range []string{"report:read", "notification:read"} {
		if p, ok := perms[node]; ok {
			var existing auth.RolePermission
			err := db.Where("role_id = ? AND permission_id = ?", ownerRole.ID, p.ID).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				rp := auth.RolePermission{ID: uuid.New(), RoleID: ownerRole.ID, PermissionID: p.ID, GrantedAt: time.Now()}
				if err := db.Create(&rp).Error; err != nil {
					log.Fatal(err)
				}
				continue
			}
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Seed admin user
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	tenantID := mustUUID(cfg.TenantID)
	adminUser := auth.User{
		ID:           uuid.New(),
		Name:         "Administrator",
		Username:     "admin",
		Email:        "admin@parkieee.local",
		PasswordHash: string(hash),
		RoleID:       adminRole.ID,
		IsActive:     true,
		TenantID:     tenantID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	var existingAdmin auth.User
	err = db.Where("username = ?", "admin").First(&existingAdmin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&adminUser).Error; err != nil {
			log.Fatal(err)
		}
	} else if err != nil {
		log.Fatal(err)
	}

	// Seed system_configs
	configs := []fee.SystemConfig{
		{Key: "ocr_auto_accept_threshold", Value: "0.80", UpdatedAt: time.Now()},
		{Key: "unclosed_check_interval_minutes", Value: "60", UpdatedAt: time.Now()},
		{Key: "payment_timeout_minutes", Value: "15", UpdatedAt: time.Now()},
	}
	for _, c := range configs {
		if err := db.Where("key = ?", c.Key).FirstOrCreate(&c).Error; err != nil {
			log.Fatal(err)
		}
	}

	log.Println("seed complete")
}

func mustUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		log.Fatalf("invalid UUID %q: %v", s, err)
	}
	return id
}

// pkg/database/db.go
package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes a connection to the PostgreSQL database.
func Connect(dsn string) (*gorm.DB, error) {
	gormLogLevel := logger.Warn
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		gormLogLevel = logger.Info
	case "error":
		gormLogLevel = logger.Error
	}

	gormLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormLogLevel,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("database: connect: %w", err)
	}
	return db, nil
}

// Migrate runs AutoMigrate for all models.
// Pass all model pointers in dependency order (referenced tables first).
func Migrate(db *gorm.DB, models ...any) error {
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("database: migrate: %w", err)
	}
	log.Println("database: migration complete")
	return nil
}

// RunMigrationFiles executes *.sql files in sqlDir in lexicographic order.
// All SQL statements must be idempotent (use IF NOT EXISTS).
func RunMigrationFiles(db *gorm.DB, sqlDir string) error {
	pattern := filepath.Join(sqlDir, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("database: glob migrations: %w", err)
	}
	sort.Strings(files)
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("database: read migration %s: %w", f, err)
		}
		if err := db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("database: exec migration %s: %w", f, err)
		}
		log.Printf("database: applied %s", filepath.Base(f))
	}
	return nil
}

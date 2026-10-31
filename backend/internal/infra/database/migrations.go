package database

import (
	"embed"
	"fmt"

	"gorm.io/gorm"
)

//go:embed migrations/*.up.sql
var migrationFiles embed.FS

const latestOrderMigration = "000003_product_catalog.up.sql"

// ApplyMigrations is only called by the explicit migration command or isolated tests.
func ApplyMigrations(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(2144051901)").Error; err != nil {
			return err
		}
		if err := tx.Exec(`CREATE TABLE IF NOT EXISTS techsnacks_schema_migrations (
			version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`).Error; err != nil {
			return err
		}
		files, err := migrationFiles.ReadDir("migrations")
		if err != nil {
			return err
		}
		for _, file := range files {
			var count int64
			if err := tx.Table("techsnacks_schema_migrations").Where("version = ?", file.Name()).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				continue
			}
			sql, err := migrationFiles.ReadFile("migrations/" + file.Name())
			if err != nil {
				return err
			}
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", file.Name(), err)
			}
			if err := tx.Exec("INSERT INTO techsnacks_schema_migrations (version) VALUES (?)", file.Name()).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func CheckOrderSchema(db *gorm.DB) error {
	var count int64
	if err := db.Table("techsnacks_schema_migrations").Where("version = ?", latestOrderMigration).Count(&count).Error; err != nil || count != 1 {
		return fmt.Errorf("order schema is not ready; run go run ./cmd/migrate before starting the API")
	}
	return nil
}

package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"time"

	"PedroFaria14/clean-pos-engine/internal/infra/config"
	"PedroFaria14/clean-pos-engine/internal/infra/database"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("Database connectivity check failed", "reason", err.Error())
		os.Exit(1)
	}
	slog.Info("Database connection verified", "readOnly", true, "persistentChanges", false)
}

func run() error {
	if err := config.LoadEnvironment(); err != nil {
		return err
	}
	settings, err := database.LoadConfig()
	if err != nil {
		return err
	}
	db, err := database.Connect()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return connectivityError{}
	}
	defer tx.Rollback()
	var result int
	if err := tx.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil || result != 1 {
		return connectivityError{}
	}
	provider := "postgresql"
	if strings.HasSuffix(strings.ToLower(settings.Host), ".neon.tech") {
		provider = "neon"
	}
	slog.Info("Read-only query completed", "provider", provider, "verifiedTLS", settings.SSLMode == "verify-full")
	return nil
}

type connectivityError struct{}

func (connectivityError) Error() string {
	return "could not complete a read-only database query; verify connectivity, TLS and credentials"
}

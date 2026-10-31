//go:build integration

package repository

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/infra/database"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func isolatedPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TECHSNACKS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TECHSNACKS_TEST_DATABASE_URL is required; use a disposable local PostgreSQL database")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(config.Database, "techsnacks_test_") ||
		(config.Host != "localhost" && config.Host != "127.0.0.1" && config.Host != "::1") {
		t.Fatal("integration tests require a loopback server and a database named techsnacks_test_*")
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { _ = admin.Close() })
	schema := "test_" + strings.ToLower(rand.Text())
	if _, err := admin.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); err != nil {
			t.Error(err)
		}
	})
	if config.RuntimeParams == nil {
		config.RuntimeParams = make(map[string]string)
	}
	config.RuntimeParams["search_path"] = schema
	connection := stdlib.OpenDB(*config)
	connection.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = connection.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: connection}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func migratedPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	db := isolatedPostgres(t)
	if err := database.ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPostgresMigrationPreservesLegacyOrdersAndRejectsOutOfOrderRollback(t *testing.T) {
	db := isolatedPostgres(t)
	if err := db.Exec(`CREATE TABLE order_models (id text PRIMARY KEY, total_cents bigint);
		INSERT INTO order_models (id, total_cents) VALUES ('legacy-order', 1300)`).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := database.ApplyMigrations(db); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.CheckOrderSchema(db); err != nil {
		t.Fatal(err)
	}
	var legacy OrderModel
	if err := db.First(&legacy, "id = ?", "legacy-order").Error; err != nil {
		t.Fatal(err)
	}
	if legacy.Status != "AWAITING_PAYMENT" || legacy.TotalCents != 1300 {
		t.Fatal("migration changed the legacy sale")
	}
	down, err := os.ReadFile("../database/migrations/000001_order_status_and_checkout_idempotency.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(string(down)).Error }); err == nil {
		t.Fatal("dependent authentication migration must block an out-of-order rollback")
	}
	var count int64
	if err := db.Table("order_models").Where("id = ?", "legacy-order").Count(&count).Error; err != nil || count != 1 {
		t.Fatal("rollback removed the legacy order")
	}
	if err := database.ApplyMigrations(db); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresCheckoutConcurrencyAndConflictAreAtomic(t *testing.T) {
	db := migratedPostgres(t)
	r := NewGormOrderRepository(db)
	results := make(chan port.CheckoutRecord, 24)
	var wg sync.WaitGroup
	for i := range 24 {
		o := testOrder(t, fmt.Sprintf("candidate-%d", i))
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := r.SaveCheckout(context.Background(), o, testRecord(o, "same-key"))
			if err != nil {
				t.Errorf("concurrent save: %v", err)
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(results)
	var first port.CheckoutRecord
	count := 0
	for result := range results {
		if count == 0 {
			first = result
		}
		if result != first {
			t.Fatal("concurrent retries produced different results")
		}
		count++
	}
	if count != 24 {
		t.Fatalf("got %d responses, want 24", count)
	}
	for _, table := range []string{"order_models", "order_item_models", "checkout_idempotency_records"} {
		var rows int64
		if err := db.Table(table).Count(&rows).Error; err != nil || rows != 1 {
			t.Fatalf("%s contains %d rows: %v", table, rows, err)
		}
	}
	o := testOrder(t, "conflicting-candidate")
	record := testRecord(o, "same-key")
	record.RequestHash = strings.Repeat("b", 64)
	if _, err := r.SaveCheckout(context.Background(), o, record); !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("expected content conflict: %v", err)
	}
	if _, err := r.FindByID(o.ID()); !errors.Is(err, port.ErrOrderNotFound) {
		t.Fatal("conflicting candidate was not rolled back")
	}
	// Explicit client order IDs must also replay correctly when the primary key races.
	original, err := r.FindByID(first.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.SaveCheckout(context.Background(), original, testRecord(original, "same-key")); err != nil {
		t.Fatalf("same order/key did not replay: %v", err)
	}
}

func TestPostgresCheckoutRollbackDoesNotReserveKey(t *testing.T) {
	db := migratedPostgres(t)
	r := NewGormOrderRepository(db)
	if err := db.Exec(`ALTER TABLE checkout_idempotency_records ADD CONSTRAINT test_reject_key CHECK (key <> 'fail-key')`).Error; err != nil {
		t.Fatal(err)
	}
	o := testOrder(t, "rollback-order")
	if _, err := r.SaveCheckout(context.Background(), o, testRecord(o, "fail-key")); err == nil {
		t.Fatal("expected database failure")
	}
	for _, table := range []string{"order_models", "order_item_models", "checkout_idempotency_records"} {
		var rows int64
		if err := db.Table(table).Count(&rows).Error; err != nil || rows != 0 {
			t.Fatalf("partial write in %s", table)
		}
	}
	if err := db.Exec(`ALTER TABLE checkout_idempotency_records DROP CONSTRAINT test_reject_key`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := r.SaveCheckout(context.Background(), o, testRecord(o, "fail-key")); err != nil {
		t.Fatalf("retry after rollback failed: %v", err)
	}
}

func TestPostgresRoundTripPreservesEveryStatus(t *testing.T) {
	r := NewGormOrderRepository(migratedPostgres(t))
	for _, status := range []string{"OPEN", "AWAITING_PAYMENT", "COMPLETED"} {
		model := orderModel(testOrder(t, status))
		model.Status = status
		o, err := restoreOrderModel(model)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Save(o); err != nil {
			t.Fatal(err)
		}
		restored, err := r.FindByID(o.ID())
		if err != nil || restored.Status() != order.Status(status) || restored.CalculateTotal().Cents() != 1300 {
			t.Fatalf("round trip %s failed: %v", status, err)
		}
	}
}

func TestPostgresHTTPReplaySurvivesRestartAndCatalogDeletion(t *testing.T) {
	db := migratedPostgres(t)
	if err := db.Create(&UserModel{ID: "http-user", Username: "http-user", PasswordHash: "unused-test-fixture", Role: "admin", Active: true}).Error; err != nil {
		t.Fatal(err)
	}
	products := NewGormProductRepository(db)
	items := testOrder(t, "source").Cart().Items()
	if err := products.Save(items[0].Product()); err != nil {
		t.Fatal(err)
	}
	newRouter := func() *gin.Engine {
		orders := NewGormOrderRepository(db)
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(api.AuthenticatedSessionKey, usecase.AuthSession{User: identity.Principal{ID: "http-user", Username: "http-user", Role: identity.Admin}})
			c.Next()
		})
		router.POST("/checkout", api.NewCheckoutHandler(usecase.NewCheckoutUseCase(orders, products)).Handle)
		router.GET("/orders/:id", api.NewOrderHandler(usecase.NewGetOrderUseCase(orders)).GetByID)
		return router
	}
	checkout := func(router *gin.Engine, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", "restart-key")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, r)
		return response
	}
	body := `{"order_id":"stable-order","items":[{"product_id":"p","quantity":2}]}`
	first := checkout(newRouter(), body)
	if first.Code != 200 {
		t.Fatalf("initial checkout failed: %s", first.Body.String())
	}
	if err := products.Delete("p"); err != nil {
		t.Fatal(err)
	}
	// A later state change must not rewrite the stored original checkout response.
	if err := db.Model(&OrderModel{}).Where("id = ?", "stable-order").Update("status", "COMPLETED").Error; err != nil {
		t.Fatal(err)
	}
	restarted := newRouter()
	replay := checkout(restarted, body)
	if replay.Code != 200 || replay.Body.String() != first.Body.String() {
		t.Fatalf("restarted API did not replay its persisted result: %d %s", replay.Code, replay.Body.String())
	}
	receipt := httptest.NewRecorder()
	restarted.ServeHTTP(receipt, httptest.NewRequest(http.MethodGet, "/orders/stable-order", nil))
	var result usecase.GetOrderOutputDTO
	if err := json.Unmarshal(receipt.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if receipt.Code != 200 || result.Status != "COMPLETED" || result.TotalCents != 1300 || len(result.Items) != 1 {
		t.Fatalf("current receipt is incorrect: %s", receipt.Body.String())
	}
	conflict := checkout(restarted, strings.Replace(body, ":2", ":3", 1))
	if conflict.Code != 409 {
		t.Fatalf("reused key with another payload returned %d", conflict.Code)
	}
	var count int64
	if err := db.Model(&OrderModel{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("HTTP retries duplicated the order")
	}
}

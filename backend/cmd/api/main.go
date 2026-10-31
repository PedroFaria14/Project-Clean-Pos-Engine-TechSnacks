package main

import (
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/infra/database"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
	"PedroFaria14/clean-pos-engine/internal/infra/repository"
	"PedroFaria14/clean-pos-engine/internal/infra/security"
	"PedroFaria14/clean-pos-engine/internal/infra/server"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("API stopped", "reason", err.Error())
		os.Exit(1)
	}
}

func run() error {
	config, err := api.AuthConfigFromEnv()
	if err != nil {
		return err
	}
	if config.SecureCookies {
		gin.SetMode(gin.ReleaseMode)
	}
	db, err := database.Connect()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return fmt.Errorf("database pool unavailable")
	}
	defer pool.Close()
	if err := database.CheckOrderSchema(db); err != nil {
		return err
	}

	productRepo := repository.NewGormProductRepository(db)
	orderRepo := repository.NewGormOrderRepository(db)

	checkoutUC := usecase.NewCheckoutUseCase(orderRepo, productRepo)
	productUC := usecase.NewProductUseCase(productRepo)
	getOrderUC := usecase.NewGetOrderUseCase(orderRepo)

	checkoutHandler := api.NewCheckoutHandler(checkoutUC)
	productHandler := api.NewProductHandler(productUC)
	orderHandler := api.NewOrderHandler(getOrderUC)

	authService, err := usecase.NewAuthService(repository.NewGormAuthRepository(db), security.Argon2id{})
	if err != nil {
		return fmt.Errorf("authentication initialization failed")
	}
	r, err := api.NewRouter(api.NewAuthHandler(authService, config), checkoutHandler, productHandler, orderHandler)
	if err != nil {
		return err
	}
	api.RegisterHealth(r, pool.PingContext)
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("cannot listen on HTTP_ADDR")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	slog.Info("API listening", "address", listener.Addr().String())
	return server.Run(ctx, server.New(address, r), listener, 15*time.Second)
}

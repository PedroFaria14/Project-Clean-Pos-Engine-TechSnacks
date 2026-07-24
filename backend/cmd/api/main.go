package main

import (
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/infra/database"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
	"PedroFaria14/clean-pos-engine/internal/infra/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	db := database.Connect()

	productRepo := repository.NewGormProductRepository(db)
	orderRepo := repository.NewGormOrderRepository(db)

	checkoutUC := usecase.NewCheckoutUseCase(orderRepo, productRepo)
	productUC := usecase.NewProductUseCase(productRepo)
	getOrderUC := usecase.NewGetOrderUseCase(orderRepo)

	checkoutHandler := api.NewCheckoutHandler(checkoutUC)
	productHandler := api.NewProductHandler(productUC)
	orderHandler := api.NewOrderHandler(getOrderUC)

	r := gin.Default()

	// Middleware Manual de CORS (Resolve o problema do OPTIONS)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		// Se for a requisição fantasma do navegador, devolve status 204 (No Content) e libera
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Rotas
	r.POST("/products", productHandler.Create)
	r.GET("/products", productHandler.List)
	r.DELETE("/products/:id", productHandler.Delete)

	r.POST("/checkout", checkoutHandler.Handle)
	r.GET("/orders/:id", orderHandler.GetByID)

	r.Run(":8080")
}

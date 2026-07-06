package main

import (
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
	"PedroFaria14/clean-pos-engine/internal/infra/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	productRepo := repository.NewMemoryProductRepository()
	orderRepo := repository.NewMemoryOrderRepository()

	checkoutUC := usecase.NewCheckoutUseCase(orderRepo, productRepo)
	productUC := usecase.NewProductUseCase(productRepo)
	getOrderUC := usecase.NewGetOrderUseCase(orderRepo) 

	checkoutHandler := api.NewCheckoutHandler(checkoutUC)
	productHandler := api.NewProductHandler(productUC)
	orderHandler := api.NewOrderHandler(getOrderUC) 

	r := gin.Default()

	// Endpoints de Produtos
	r.POST("/products", productHandler.Create)
	r.GET("/products", productHandler.List)

	// Endpoints de Pedidos
	r.POST("/checkout", checkoutHandler.Handle)
	r.GET("/orders/:id", orderHandler.GetByID)

	r.Run(":8080")
}
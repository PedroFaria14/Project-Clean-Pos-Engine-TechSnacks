package api

import (
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"net/http"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	getOrderUC *usecase.GetOrderUseCase
}

func NewOrderHandler(uc *usecase.GetOrderUseCase) *OrderHandler {
	return &OrderHandler{getOrderUC: uc}
}

func (h *OrderHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	output, err := h.getOrderUC.Execute(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, output)
}
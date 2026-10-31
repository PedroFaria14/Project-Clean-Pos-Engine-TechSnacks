package api

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"errors"
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
	actor, authenticated := Principal(c)
	if !authenticated {
		authError(c, usecase.ErrUnauthenticated)
		return
	}
	id := c.Param("id")

	output, err := h.getOrderUC.ExecuteForUser(id, actor)
	if err != nil {
		if !errors.Is(err, port.ErrOrderNotFound) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "order temporarily unavailable"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, output)
}

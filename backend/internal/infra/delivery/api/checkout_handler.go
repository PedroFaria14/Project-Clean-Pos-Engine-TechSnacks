package api

import (
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CheckoutHandler struct {
	usecase *usecase.CheckoutUseCase
}

func NewCheckoutHandler(uc *usecase.CheckoutUseCase) *CheckoutHandler {
	return &CheckoutHandler{usecase: uc}
}

func (h *CheckoutHandler) Handle(c *gin.Context) {
	var input usecase.CheckoutInputDTO

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	output, err := h.usecase.Execute(input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}
package api

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"errors"
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
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024)
	actor, authenticated := Principal(c)
	if !authenticated {
		authError(c, usecase.ErrUnauthenticated)
		return
	}
	keys := c.Request.Header.Values("Idempotency-Key")
	if len(keys) > 1 || len(keys) == 1 && !usecase.ValidIdempotencyKey(keys[0]) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid idempotency key", "details": []gin.H{{"field": "Idempotency-Key", "message": "provide exactly one key with 1 to 128 ASCII letters, digits, underscores or hyphens"}}})
		return
	}
	var key string
	if len(keys) == 1 {
		key = keys[0]
	}
	var input usecase.CheckoutInputDTO

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": []gin.H{{"field": "body", "message": "provide valid checkout JSON within the request size limit"}}})
		return
	}

	output, err := h.usecase.ExecuteForUser(c.Request.Context(), input, key, actor)
	if err != nil {
		if errors.Is(err, port.ErrIdempotencyConflict) || errors.Is(err, port.ErrOrderConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, usecase.ErrCheckoutUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "checkout temporarily unavailable"})
			return
		}
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, output)
}

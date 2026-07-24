package api

import (
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	usecase *usecase.ProductUseCase
}

func NewProductHandler(uc *usecase.ProductUseCase) *ProductHandler {
	return &ProductHandler{usecase: uc}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var input usecase.CreateProductInputDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	output, err := h.usecase.Create(input)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, output)
}

func (h *ProductHandler) List(c *gin.Context) {
	output, err := h.usecase.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, output)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.usecase.Delete(id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.Status(204)
}

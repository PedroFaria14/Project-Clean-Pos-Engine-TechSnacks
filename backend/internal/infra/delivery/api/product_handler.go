package api

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *ProductHandler) Update(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil || raw == nil {
		productBodyError(c)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		productBodyError(c)
		return
	}
	for key, value := range raw {
		if key != "name" && key != "price" || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			productBodyError(c)
			return
		}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		productBodyError(c)
		return
	}
	var input usecase.UpdateProductInputDTO
	if err := json.Unmarshal(encoded, &input); err != nil {
		productBodyError(c)
		return
	}
	output, err := h.usecase.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		var validation *usecase.ProductValidationError
		switch {
		case errors.As(err, &validation):
			c.JSON(400, gin.H{"error": validation.Error(), "details": validation.Details})
		case errors.Is(err, port.ErrProductNotFound):
			c.JSON(404, gin.H{"error": "product not found"})
		default:
			c.JSON(503, gin.H{"error": "product temporarily unavailable"})
		}
		return
	}
	c.JSON(200, output)
}

func productBodyError(c *gin.Context) {
	c.JSON(400, gin.H{"error": "invalid product input", "details": []gin.H{{"field": "body", "message": "check required fields and types; unknown fields and null are not allowed"}}})
}

type ProductHandler struct {
	usecase *usecase.ProductUseCase
}

func NewProductHandler(uc *usecase.ProductUseCase) *ProductHandler {
	return &ProductHandler{usecase: uc}
}

func (h *ProductHandler) Create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	var input struct {
		ID    string  `json:"id"`
		Name  *string `json:"name"`
		Price *int64  `json:"price"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.Name == nil || input.Price == nil {
		productBodyError(c)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		productBodyError(c)
		return
	}
	output, err := h.usecase.Create(usecase.CreateProductInputDTO{ID: input.ID, Name: *input.Name, Price: *input.Price})
	if err != nil {
		var validation *usecase.ProductValidationError
		switch {
		case errors.As(err, &validation):
			c.JSON(400, gin.H{"error": validation.Error(), "details": validation.Details})
		case errors.Is(err, port.ErrProductConflict):
			c.JSON(409, gin.H{"error": "product id already exists"})
		default:
			c.JSON(503, gin.H{"error": "product temporarily unavailable"})
		}
		return
	}
	c.JSON(http.StatusCreated, output)
}

func (h *ProductHandler) List(c *gin.Context) {
	output, err := h.usecase.ListAll()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "catalog temporarily unavailable"})
		return
	}
	c.JSON(http.StatusOK, output)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.usecase.Delete(id)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "product temporarily unavailable"})
		return
	}

	c.Status(204)
}

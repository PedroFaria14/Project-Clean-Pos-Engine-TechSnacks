package order

import (
	"errors"
	"fmt"

	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

type CartItem struct {
	product  *product.Product
	quantity int
	subtotal shared.Money
}

func NewCartItem(product *product.Product, quantity int) (*CartItem, error) {
	if product == nil {
		return nil, errors.New("product cannot be nil")
	}
	if quantity <= 0 {
		return nil, errors.New("quantity must be greater than zero")
	}
	subtotal, err := product.Price().Multiply(int64(quantity))
	if err != nil {
		return nil, fmt.Errorf("calculate item subtotal: %w", err)
	}
	return &CartItem{
		product:  product,
		quantity: quantity,
		subtotal: subtotal,
	}, nil
}

func (c *CartItem) Product() *product.Product { return c.product }
func (c *CartItem) Quantity() int             { return c.quantity }

func (c *CartItem) SubTotal() shared.Money {
	return c.subtotal
}

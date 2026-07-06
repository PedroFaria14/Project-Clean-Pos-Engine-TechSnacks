package order

import (
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"errors"
)


type CartItem struct {
	product  *product.Product
	quantity int
}

func NewCartItem(product *product.Product, quantity int) (*CartItem, error) {
	if product == nil {
		return nil, errors.New("product cannot be nil")
	}
	if quantity <= 0 {
		return nil, errors.New("quantity must be greater than zero")
	}
	return &CartItem{
		product:  product,
		quantity: quantity,
	}, nil
}

func (c *CartItem) Product() *product.Product { return c.product }
func (c *CartItem) Quantity() int { return c.quantity }

func (c *CartItem) SubTotal() shared.Money {
    totalCents := c.product.Price().Cents() * int64(c.quantity)
    
    money, _ := shared.NewMoney(totalCents)
    
    return money
}
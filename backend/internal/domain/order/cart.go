package order

import (
	"fmt"
	"slices"

	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

type Cart struct {
	items []CartItem
	total shared.Money
}

func NewCart() *Cart {
	return &Cart{
		items: make([]CartItem, 0),
	}
}

// AddItem adds a valid item only when the resulting cart total fits in int64.
func (c *Cart) AddItem(item CartItem) error {
	validated, err := NewCartItem(item.Product(), item.Quantity())
	if err != nil {
		return err
	}
	total, err := c.total.Add(validated.SubTotal())
	if err != nil {
		return fmt.Errorf("calculate cart total: %w", err)
	}
	c.items = append(c.items, *validated)
	c.total = total
	return nil
}

// Items returns a copy so callers cannot invalidate the checked cart total.
func (c *Cart) Items() []CartItem {
	return slices.Clone(c.items)
}

func (c *Cart) Total() shared.Money {
	return c.total
}

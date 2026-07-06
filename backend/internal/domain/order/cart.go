package order

import (
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)



type Cart struct {
	items []CartItem
}

func NewCart() *Cart {
	return &Cart{
		items: make([]CartItem, 0),
	}
}

func (c *Cart) AddItem(item CartItem) {
	c.items = append(c.items, item)
}

func (c *Cart) Items() []CartItem {
	return c.items
}

func (c *Cart) Total() shared.Money {
	var total int64 = 0
	for _, item := range c.items {
		total += item.SubTotal().Cents()
	}

	money, _ := shared.NewMoney(total)
	return money
}
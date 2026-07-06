package repository

import (
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"errors"
)

type MemoryOrderRepository struct {
	orders map[string]*order.Order
}

func NewMemoryOrderRepository() *MemoryOrderRepository {
	return &MemoryOrderRepository{
		orders: make(map[string]*order.Order),
	}
}

func (r *MemoryOrderRepository) Save(o *order.Order) error {
	r.orders[o.ID()] = o
	return nil
}

func (r *MemoryOrderRepository) FindByID(id string) (*order.Order, error) {
	if o, exists := r.orders[id]; exists {
		return o, nil
	}
	return nil, errors.New("order not found")
}
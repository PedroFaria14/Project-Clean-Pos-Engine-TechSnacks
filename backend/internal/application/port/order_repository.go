package port

import "PedroFaria14/clean-pos-engine/internal/domain/order"

type OrderRepository interface {
	Save(o *order.Order) error
	FindByID(id string) (*order.Order, error)
}
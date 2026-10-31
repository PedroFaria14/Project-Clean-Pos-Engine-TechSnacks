package port

import (
	"context"
	"errors"

	"PedroFaria14/clean-pos-engine/internal/domain/order"
)

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderConflict       = errors.New("order id already exists")
	ErrCheckoutNotFound    = errors.New("checkout key not found")
	ErrIdempotencyConflict = errors.New("idempotency key already used with different content")
)

// CheckoutRecord is the immutable original response for POST /checkout.
type CheckoutRecord struct {
	Key         string
	RequestHash string
	OrderID     string
	TotalCents  int64
	Status      string
}

type OrderRepository interface {
	Save(o *order.Order) error
	FindByID(id string) (*order.Order, error)
	FindCheckout(ctx context.Context, key string) (CheckoutRecord, error)
	SaveCheckout(ctx context.Context, o *order.Order, record CheckoutRecord) (CheckoutRecord, error)
}

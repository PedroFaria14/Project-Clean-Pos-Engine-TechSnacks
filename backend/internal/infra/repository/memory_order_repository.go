package repository

import (
	"context"
	"sync"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
)

type MemoryOrderRepository struct {
	mu        sync.RWMutex
	orders    map[string]*order.Order
	checkouts map[string]port.CheckoutRecord
}

func NewMemoryOrderRepository() *MemoryOrderRepository {
	return &MemoryOrderRepository{
		orders:    make(map[string]*order.Order),
		checkouts: make(map[string]port.CheckoutRecord),
	}
}

func (r *MemoryOrderRepository) Save(o *order.Order) error {
	snapshot, err := order.RestoreOrderWithCreator(o.ID(), o.Status(), o.Cart().Items(), o.CalculateTotal(), o.CreatorID())
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.orders[o.ID()]; exists {
		return port.ErrOrderConflict
	}
	r.orders[o.ID()] = snapshot
	return nil
}

func (r *MemoryOrderRepository) FindByID(id string) (*order.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if o, exists := r.orders[id]; exists {
		return order.RestoreOrderWithCreator(o.ID(), o.Status(), o.Cart().Items(), o.CalculateTotal(), o.CreatorID())
	}
	return nil, port.ErrOrderNotFound
}

func (r *MemoryOrderRepository) FindCheckout(ctx context.Context, key string) (port.CheckoutRecord, error) {
	if err := ctx.Err(); err != nil {
		return port.CheckoutRecord{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if record, exists := r.checkouts[key]; exists {
		return record, nil
	}
	return port.CheckoutRecord{}, port.ErrCheckoutNotFound
}

func (r *MemoryOrderRepository) SaveCheckout(ctx context.Context, o *order.Order, record port.CheckoutRecord) (port.CheckoutRecord, error) {
	if err := validateCheckoutSnapshot(o, record); err != nil {
		return port.CheckoutRecord{}, err
	}
	snapshot, err := order.RestoreOrderWithCreator(o.ID(), o.Status(), o.Cart().Items(), o.CalculateTotal(), o.CreatorID())
	if err != nil {
		return port.CheckoutRecord{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return port.CheckoutRecord{}, err
	}
	if existing, exists := r.checkouts[record.Key]; exists {
		if existing.RequestHash != record.RequestHash {
			return port.CheckoutRecord{}, port.ErrIdempotencyConflict
		}
		return existing, nil
	}
	if _, exists := r.orders[o.ID()]; exists {
		return port.CheckoutRecord{}, port.ErrOrderConflict
	}
	r.orders[o.ID()] = snapshot
	r.checkouts[record.Key] = record
	return record, nil
}

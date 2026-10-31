package repository

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"context"
	"sort"
	"sync"
)

type MemoryProductRepository struct {
	mu       sync.RWMutex
	products map[string]*product.Product
}

func NewMemoryProductRepository() *MemoryProductRepository {
	return &MemoryProductRepository{
		products: make(map[string]*product.Product),
	}
}

func (r *MemoryProductRepository) Save(p *product.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.products[p.ID()]; exists {
		return port.ErrProductConflict
	}
	r.products[p.ID()] = p
	return nil
}

func (r *MemoryProductRepository) FindByID(id string) (*product.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, exists := r.products[id]; exists {
		return p, nil
	}
	return nil, port.ErrProductNotFound
}

func (r *MemoryProductRepository) FindAll() ([]*product.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*product.Product
	for _, p := range r.products {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Name() == list[j].Name() {
			return list[i].ID() < list[j].ID()
		}
		return list[i].Name() < list[j].Name()
	})
	return list, nil
}

func (r *MemoryProductRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.products, id)
	return nil
}

func (r *MemoryProductRepository) Update(ctx context.Context, id string, name *string, price *int64) (*product.Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, exists := r.products[id]
	if !exists {
		return nil, port.ErrProductNotFound
	}
	newName, cents := p.Name(), p.Price().Cents()
	if name != nil {
		newName = *name
	}
	if price != nil {
		cents = *price
	}
	money, err := shared.NewMoney(cents)
	if err != nil {
		return nil, err
	}
	updated, err := product.NewProduct(id, newName, money)
	if err != nil {
		return nil, err
	}
	r.products[id] = updated
	return updated, nil
}

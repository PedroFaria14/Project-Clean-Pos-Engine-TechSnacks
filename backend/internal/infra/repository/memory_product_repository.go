package repository

import (
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"errors"
)

type MemoryProductRepository struct {
	products map[string]*product.Product
}

func NewMemoryProductRepository() *MemoryProductRepository {
	return &MemoryProductRepository{
		products: make(map[string]*product.Product),
	}
}

func (r *MemoryProductRepository) Save(p *product.Product) error {
	r.products[p.ID()] = p
	return nil
}

func (r *MemoryProductRepository) FindByID(id string) (*product.Product, error) {
	if p, exists := r.products[id]; exists {
		return p, nil
	}
	return nil, errors.New("product not found")
}

func (r *MemoryProductRepository) FindAll() ([]*product.Product, error) {
	var list []*product.Product
	for _, p := range r.products {
		list = append(list, p)
	}
	return list, nil
}
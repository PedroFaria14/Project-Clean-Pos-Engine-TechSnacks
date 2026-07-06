package port

import "PedroFaria14/clean-pos-engine/internal/domain/product"

type ProductRepository interface {
	Save(p *product.Product) error
	FindByID(id string) (*product.Product, error)
	FindAll() ([]*product.Product, error)
}
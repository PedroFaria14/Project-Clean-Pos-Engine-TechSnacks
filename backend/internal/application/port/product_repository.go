package port

import (
	"context"
	"errors"

	"PedroFaria14/clean-pos-engine/internal/domain/product"
)

var ErrProductNotFound = errors.New("product not found")
var ErrProductConflict = errors.New("product id already exists")

type ProductRepository interface {
	Save(p *product.Product) error
	FindByID(id string) (*product.Product, error)
	FindAll() ([]*product.Product, error)
	Delete(id string) error
	Update(ctx context.Context, id string, name *string, price *int64) (*product.Product, error)
}

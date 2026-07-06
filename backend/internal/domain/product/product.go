package product

import (
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"errors"
)

type Product struct {
	id    string
	name  string
	price shared.Money
}

func NewProduct(id, name string, price shared.Money) (*Product, error) {
	if id == "" || name == "" {
		return nil, errors.New("id and name are required")
	}
	return &Product{
		id:    id,
		name:  name,
		price: price,
	}, nil
}

func (p *Product) ID() string { return p.id }
func (p *Product) Name() string { return p.name }
func (p *Product) Price() shared.Money { return p.price }
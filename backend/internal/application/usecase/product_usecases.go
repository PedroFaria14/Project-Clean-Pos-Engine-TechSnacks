package usecase

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

// --- DTOs ---
type CreateProductInputDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"` // em centavos
}

type ProductOutputDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

// --- Use Case ---
type ProductUseCase struct {
	repo port.ProductRepository
}

func NewProductUseCase(r port.ProductRepository) *ProductUseCase {
	return &ProductUseCase{repo: r}
}

func (uc *ProductUseCase) Create(input CreateProductInputDTO) (ProductOutputDTO, error) {
	price, err := shared.NewMoney(input.Price)
	if err != nil {
		return ProductOutputDTO{}, err
	}

	p, err := product.NewProduct(input.ID, input.Name, price)
	if err != nil {
		return ProductOutputDTO{}, err
	}

	err = uc.repo.Save(p)
	if err != nil {
		return ProductOutputDTO{}, err
	}

	return ProductOutputDTO{ID: p.ID(), Name: p.Name(), Price: p.Price().Cents()}, nil
}

func (uc *ProductUseCase) ListAll() ([]ProductOutputDTO, error) {
	products, err := uc.repo.FindAll()
	if err != nil {
		return nil, err
	}

	var output []ProductOutputDTO
	for _, p := range products {
		output = append(output, ProductOutputDTO{
			ID:    p.ID(),
			Name:  p.Name(),
			Price: p.Price().Cents(),
		})
	}
	return output, nil
}
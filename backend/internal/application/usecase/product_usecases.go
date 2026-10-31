package usecase

import (
	"context"
	"strings"
	"unicode/utf8"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

type ValidationDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ProductValidationError struct{ Details []ValidationDetail }

func (*ProductValidationError) Error() string { return "invalid product input" }

type UpdateProductInputDTO struct {
	Name  *string `json:"name"`
	Price *int64  `json:"price"`
}

// Update validates the supplied fields and atomically changes only those fields.
func (uc *ProductUseCase) Update(ctx context.Context, id string, input UpdateProductInputDTO) (ProductOutputDTO, error) {
	details := []ValidationDetail{}
	if input.Name == nil && input.Price == nil {
		details = append(details, ValidationDetail{Field: "body", Message: "provide name or price"})
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		input.Name = &name
		if name == "" || utf8.RuneCountInString(name) > 120 {
			details = append(details, ValidationDetail{Field: "name", Message: "name must contain 1 to 120 characters"})
		}
	}
	if input.Price != nil && (*input.Price < 0 || *input.Price > 9007199254740991) {
		details = append(details, ValidationDetail{Field: "price", Message: "price must be a nonnegative safe integer in cents"})
	}
	if len(details) != 0 {
		return ProductOutputDTO{}, &ProductValidationError{Details: details}
	}
	p, err := uc.repo.Update(ctx, id, input.Name, input.Price)
	if err != nil {
		return ProductOutputDTO{}, err
	}
	return ProductOutputDTO{ID: p.ID(), Name: p.Name(), Price: p.Price().Cents()}, nil
}

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
	details := []ValidationDetail{}
	input.Name = strings.TrimSpace(input.Name)
	if strings.TrimSpace(input.ID) == "" || utf8.RuneCountInString(input.ID) > 128 {
		details = append(details, ValidationDetail{Field: "id", Message: "id must contain 1 to 128 characters"})
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 120 {
		details = append(details, ValidationDetail{Field: "name", Message: "name must contain 1 to 120 characters"})
	}
	if input.Price < 0 || input.Price > 9007199254740991 {
		details = append(details, ValidationDetail{Field: "price", Message: "price must be a nonnegative safe integer in cents"})
	}
	if len(details) != 0 {
		return ProductOutputDTO{}, &ProductValidationError{Details: details}
	}
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

	output := []ProductOutputDTO{}
	for _, p := range products {
		output = append(output, ProductOutputDTO{
			ID:    p.ID(),
			Name:  p.Name(),
			Price: p.Price().Cents(),
		})
	}
	return output, nil
}

func (uc *ProductUseCase) Delete(id string) error {
	return uc.repo.Delete(id)
}

package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/infra/repository"
)

func productValue[T any](value T) *T { return &value }

func TestProductUpdateValidatesAndChangesOnlySuppliedFields(t *testing.T) {
	repo := repository.NewMemoryProductRepository()
	if err := repo.Save(checkoutProduct(t, 650)); err != nil {
		t.Fatal(err)
	}
	uc := usecase.NewProductUseCase(repo)
	name := "  Coxinha artesanal  "
	updated, err := uc.Update(context.Background(), "product-1", usecase.UpdateProductInputDTO{Name: &name})
	if err != nil || updated.Name != "Coxinha artesanal" || updated.Price != 650 {
		t.Fatalf("partial name update failed: %+v %v", updated, err)
	}
	updated, err = uc.Update(context.Background(), "product-1", usecase.UpdateProductInputDTO{Price: productValue(int64(0))})
	if err != nil || updated.Name != "Coxinha artesanal" || updated.Price != 0 {
		t.Fatal("zero price or unsupplied name was lost")
	}
	for _, input := range []usecase.UpdateProductInputDTO{
		{}, {Name: productValue(" ")}, {Name: productValue(strings.Repeat("x", 121))}, {Price: productValue(int64(-1))}, {Price: productValue(int64(9007199254740992))},
	} {
		_, err := uc.Update(context.Background(), "product-1", input)
		var validation *usecase.ProductValidationError
		if !errors.As(err, &validation) || len(validation.Details) == 0 {
			t.Fatalf("invalid input accepted: %+v %v", input, err)
		}
	}
	if _, err := uc.Update(context.Background(), "missing", usecase.UpdateProductInputDTO{Name: productValue("Valid")}); !errors.Is(err, port.ErrProductNotFound) {
		t.Fatal("missing product not identified")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Update(ctx, "product-1", usecase.UpdateProductInputDTO{Name: productValue("Valid")}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation was ignored")
	}
}

func TestProductCreationValidatesAndDoesNotOverwriteDuplicateIDs(t *testing.T) {
	repo := repository.NewMemoryProductRepository()
	uc := usecase.NewProductUseCase(repo)
	for _, input := range []usecase.CreateProductInputDTO{{ID: "", Name: "Valid", Price: 1}, {ID: "p", Name: " ", Price: 1}, {ID: "p", Name: "Valid", Price: -1}, {ID: "p", Name: "Valid", Price: 9007199254740992}} {
		_, err := uc.Create(input)
		var validation *usecase.ProductValidationError
		if !errors.As(err, &validation) || len(validation.Details) == 0 {
			t.Fatal("invalid create input was accepted")
		}
	}
	if _, err := uc.Create(usecase.CreateProductInputDTO{ID: "p", Name: " Original ", Price: 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(usecase.CreateProductInputDTO{ID: "p", Name: "Replacement", Price: 100}); !errors.Is(err, port.ErrProductConflict) {
		t.Fatal("duplicate create was not rejected")
	}
	p, err := repo.FindByID("p")
	if err != nil || p.Name() != "Original" || p.Price().Cents() != 0 {
		t.Fatal("duplicate create modified existing data")
	}
}

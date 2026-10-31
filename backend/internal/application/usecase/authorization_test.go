package usecase_test

import (
	"context"
	"errors"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
	"PedroFaria14/clean-pos-engine/internal/infra/repository"
)

func TestOrderOwnershipAndIdempotencyAreScopedToPrincipal(t *testing.T) {
	repo := repository.NewMemoryOrderRepository()
	uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650)})
	firstUser := identity.Principal{ID: "operator-1", Role: identity.Operator}
	secondUser := identity.Principal{ID: "operator-2", Role: identity.Operator}
	input := usecase.CheckoutInputDTO{Items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 2}}}
	first, err := uc.ExecuteForUser(context.Background(), input, "shared-raw-key", firstUser)
	if err != nil {
		t.Fatal(err)
	}
	second, err := uc.ExecuteForUser(context.Background(), input, "shared-raw-key", secondUser)
	if err != nil || second.OrderID == first.OrderID {
		t.Fatal("another principal recovered the first principal's order")
	}
	replay, err := uc.ExecuteForUser(context.Background(), input, "shared-raw-key", firstUser)
	if err != nil || replay != first {
		t.Fatal("same principal did not recover its own order")
	}
	reader := usecase.NewGetOrderUseCase(repo)
	if _, err := reader.ExecuteForUser(first.OrderID, firstUser); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ExecuteForUser(first.OrderID, secondUser); !errors.Is(err, port.ErrOrderNotFound) {
		t.Fatal("operator accessed another operator's order")
	}
	if _, err := reader.ExecuteForUser(first.OrderID, identity.Principal{ID: "admin", Role: identity.Admin}); err != nil {
		t.Fatal("admin could not read an operator's order")
	}
	legacy, err := uc.Execute(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ExecuteForUser(legacy.OrderID, firstUser); !errors.Is(err, port.ErrOrderNotFound) {
		t.Fatal("operator accessed an unowned legacy order")
	}
	if _, err := reader.ExecuteForUser(legacy.OrderID, identity.Principal{ID: "admin", Role: identity.Admin}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.ExecuteForUser(context.Background(), input, "key", identity.Principal{}); !errors.Is(err, usecase.ErrUnauthenticated) {
		t.Fatal("anonymous checkout accepted")
	}
}

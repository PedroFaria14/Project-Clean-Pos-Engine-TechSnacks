package order_test

import (
	"math"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

func newProduct(t *testing.T, cents int64) *product.Product {
	t.Helper()
	price, err := shared.NewMoney(cents)
	if err != nil {
		t.Fatal(err)
	}
	p, err := product.NewProduct("product-1", "Coxinha", price)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func newItem(t *testing.T, cents int64, quantity int) order.CartItem {
	t.Helper()
	item, err := order.NewCartItem(newProduct(t, cents), quantity)
	if err != nil {
		t.Fatal(err)
	}
	return *item
}

func TestNewCartItemCalculatesCheckedSubtotal(t *testing.T) {
	for _, tt := range []struct {
		price    int64
		quantity int
		want     int64
	}{
		{price: 650, quantity: 2, want: 1300},
		{price: 0, quantity: 3, want: 0},
		{price: math.MaxInt64, quantity: 1, want: math.MaxInt64},
	} {
		item := newItem(t, tt.price, tt.quantity)
		if item.SubTotal().Cents() != tt.want || item.Quantity() != tt.quantity {
			t.Fatalf("unexpected cart item: %+v", item)
		}
	}
}

func TestNewCartItemRejectsInvalidValues(t *testing.T) {
	for _, quantity := range []int{-1, 0, 2, 3} {
		if _, err := order.NewCartItem(newProduct(t, math.MaxInt64), quantity); err == nil {
			t.Errorf("quantity %d should be rejected for max-priced product", quantity)
		}
	}
	if _, err := order.NewCartItem(nil, 1); err == nil {
		t.Fatal("nil product must be rejected")
	}
}

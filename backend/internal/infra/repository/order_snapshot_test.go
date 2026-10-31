package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
)

func testOrder(t *testing.T, id string) *order.Order {
	t.Helper()
	o, err := restoreOrderModel(OrderModel{ID: id, Status: "AWAITING_PAYMENT", TotalCents: 1300,
		Items: []OrderItemModel{{ProductID: "p", ProductName: "Coxinha", Quantity: 2, PriceCents: 650}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func testRecord(o *order.Order, key string) port.CheckoutRecord {
	return port.CheckoutRecord{Key: key, RequestHash: strings.Repeat("a", 64), OrderID: o.ID(), TotalCents: o.CalculateTotal().Cents(), Status: string(o.Status())}
}

func TestOrderModelRestoresAllStatesAndHistoricalTotals(t *testing.T) {
	for _, status := range []string{"OPEN", "AWAITING_PAYMENT", "COMPLETED"} {
		model := orderModel(testOrder(t, "restore"))
		model.Status = status
		if status != "OPEN" {
			model.TotalCents = 1100
		}
		o, err := restoreOrderModel(model)
		if err != nil || string(o.Status()) != status || o.CalculateTotal().Cents() != model.TotalCents {
			t.Fatalf("restore %s failed: %v", status, err)
		}
	}
	for _, change := range []func(*OrderModel){
		func(m *OrderModel) { m.Status = "UNKNOWN" },
		func(m *OrderModel) { m.TotalCents = -1 },
		func(m *OrderModel) { m.Items[0].PriceCents = -1 },
		func(m *OrderModel) { m.Items[0].ProductName = "" },
		func(m *OrderModel) { m.Items[0].Quantity = 0 },
	} {
		model := orderModel(testOrder(t, "invalid"))
		change(&model)
		if _, err := restoreOrderModel(model); err == nil {
			t.Fatal("invalid persisted data was accepted")
		}
	}
}

func TestMemoryCheckoutDoesNotPersistLosingCandidates(t *testing.T) {
	r := NewMemoryOrderRepository()
	ctx := context.Background()
	first := testOrder(t, "first")
	original, err := r.SaveCheckout(ctx, first, testRecord(first, "key"))
	if err != nil {
		t.Fatal(err)
	}
	second := testOrder(t, "second")
	replay, err := r.SaveCheckout(ctx, second, testRecord(second, "key"))
	if err != nil || replay != original {
		t.Fatal("retry did not return the original result")
	}
	if len(r.orders) != 1 || len(r.checkouts) != 1 {
		t.Fatal("losing candidate was persisted")
	}
	conflict := testRecord(second, "key")
	conflict.RequestHash = strings.Repeat("b", 64)
	if _, err := r.SaveCheckout(ctx, second, conflict); !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatal("payload mismatch accepted")
	}
	if _, err := r.FindByID("second"); !errors.Is(err, port.ErrOrderNotFound) {
		t.Fatal("conflict persisted an order")
	}
	if err := r.Save(first); !errors.Is(err, port.ErrOrderConflict) {
		t.Fatal("duplicate order ID accepted")
	}
}

package order_test

import (
	"testing"

	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

func TestRestoreOrderPreservesStatusAndSnapshotTotal(t *testing.T) {
	for _, status := range []order.Status{order.StatusOpen, order.StatusAwaitingPayment, order.StatusCompleted} {
		t.Run(string(status), func(t *testing.T) {
			totalCents := int64(1300)
			if status != order.StatusOpen {
				totalCents = 1100
			} // A historical discounted total is a snapshot.
			total, err := shared.NewMoney(totalCents)
			if err != nil {
				t.Fatal(err)
			}
			o, err := order.RestoreOrder("restored", status, []order.CartItem{newItem(t, 650, 2)}, total)
			if err != nil {
				t.Fatal(err)
			}
			if o.Status() != status || o.CalculateTotal().Cents() != totalCents {
				t.Fatal("snapshot was not preserved")
			}
			if status != order.StatusOpen {
				if err := o.AddItem(newItem(t, 1, 1)); err == nil {
					t.Fatal("restored closed cart accepted changes")
				}
				if err := o.Checkout(); err == nil {
					t.Fatal("restored checkout executed twice")
				}
			} else {
				if err := o.AddItem(newItem(t, 100, 1)); err != nil {
					t.Fatal(err)
				}
				if o.CalculateTotal().Cents() != 1400 {
					t.Fatal("open order did not recalculate after editing")
				}
			}
		})
	}
}

func TestRestoreOrderRejectsInvalidSnapshots(t *testing.T) {
	zero := shared.Money{}
	for _, tt := range []struct {
		id     string
		status order.Status
		items  []order.CartItem
	}{
		{id: "", status: order.StatusOpen},
		{id: "x", status: "UNKNOWN"},
		{id: "x", status: order.StatusAwaitingPayment},
		{id: "x", status: order.StatusCompleted},
		{id: "x", status: order.StatusOpen, items: []order.CartItem{newItem(t, 650, 1)}},
	} {
		if _, err := order.RestoreOrder(tt.id, tt.status, tt.items, zero); err == nil {
			t.Fatalf("accepted invalid snapshot: %+v", tt)
		}
	}
}

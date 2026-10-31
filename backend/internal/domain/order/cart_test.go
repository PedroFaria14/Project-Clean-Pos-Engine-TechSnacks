package order_test

import (
	"math"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/domain/order"
)

func TestCartAccumulatesCheckedTotal(t *testing.T) {
	cart := order.NewCart()
	if cart.Total().Cents() != 0 || len(cart.Items()) != 0 {
		t.Fatal("new cart must be empty with a zero total")
	}
	for _, item := range []order.CartItem{newItem(t, 650, 2), newItem(t, 350, 1)} {
		if err := cart.AddItem(item); err != nil {
			t.Fatal(err)
		}
	}
	if cart.Total().Cents() != 1650 || len(cart.Items()) != 2 {
		t.Fatal("unexpected accumulated cart total")
	}
}

func TestCartRejectsOverflowWithoutChangingItemsOrTotal(t *testing.T) {
	cart := order.NewCart()
	if err := cart.AddItem(newItem(t, math.MaxInt64-1, 1)); err != nil {
		t.Fatal(err)
	}
	if err := cart.AddItem(newItem(t, 1, 1)); err != nil {
		t.Fatalf("exact int64 limit must be accepted: %v", err)
	}
	if err := cart.AddItem(newItem(t, 1, 1)); err == nil {
		t.Fatal("cart total overflow must be rejected")
	}
	if cart.Total().Cents() != math.MaxInt64 || len(cart.Items()) != 2 {
		t.Fatal("rejected addition changed the cart")
	}
}

func TestCartRejectsUninitializedItem(t *testing.T) {
	cart := order.NewCart()
	if err := cart.AddItem(order.CartItem{}); err == nil {
		t.Fatal("uninitialized item must be rejected")
	}
	if len(cart.Items()) != 0 || cart.Total().Cents() != 0 {
		t.Fatal("invalid item changed the cart")
	}
}

func TestCartItemsCannotModifyInternalTotal(t *testing.T) {
	cart := order.NewCart()
	if err := cart.AddItem(newItem(t, 650, 2)); err != nil {
		t.Fatal(err)
	}
	items := cart.Items()
	items[0] = newItem(t, 1, 1)
	stored := cart.Items()
	if stored[0].SubTotal().Cents() != 1300 || cart.Total().Cents() != 1300 {
		t.Fatal("external mutation changed checked cart contents")
	}
}

func TestOrderPropagatesCartOverflow(t *testing.T) {
	o := order.NewOrder("order-1")
	if err := o.AddItem(newItem(t, math.MaxInt64, 1)); err != nil {
		t.Fatal(err)
	}
	if err := o.AddItem(newItem(t, 1, 1)); err == nil {
		t.Fatal("order must propagate cart validation failures")
	}
	if err := o.Checkout(); err != nil {
		t.Fatal(err)
	}
	if err := o.AddItem(newItem(t, 0, 1)); err == nil {
		t.Fatal("order must reject item changes while awaiting payment")
	}
}

package order

import (
	"PedroFaria14/clean-pos-engine/internal/domain/payment"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"errors"
)

type Order struct {
    id              string
    cart            *Cart
    state           State
    discount        *payment.Discount
    paymentStrategy payment.PaymentStrategy
    observers       []Observer
}

func NewOrder(id string) *Order {
	order := &Order{
		id:        id,
		cart:      NewCart(),
		observers: make([]Observer, 0),
	}
	order.SetState(&OpenState{})
	return order
}

func (o *Order) SetState(state State) {
	o.state = state
}

func (o *Order) Cart() *Cart {
	return o.cart
}

func (o *Order) ApplyDiscount(discount *payment.Discount) error {
	if o.paymentStrategy != nil {
		return errors.New("cannot apply discount after payment strategy is set")
	}
	o.discount = discount
	return nil
}

func (o *Order) SetPaymentStrategy(strategy payment.PaymentStrategy) {
	o.paymentStrategy = strategy
}

func (o *Order) CalculateTotal() shared.Money {
	subtotal := o.cart.Total()

	if o.discount != nil {
		subtotal = o.discount.Apply(subtotal)
	}

	if o.paymentStrategy != nil {
		return o.paymentStrategy.CalculateFinalTotal(subtotal)
	}

	return subtotal
}

func (o *Order) AddItem(item CartItem) error {
	return o.state.AddItem(o, item)
}

func (o *Order) Checkout() error {
	return o.state.Checkout(o)
}

func (o *Order) Pay() error {
	if o.paymentStrategy == nil {
		return errors.New("payment strategy must be set before paying")
	}
	return o.state.Pay(o)
}

func (o *Order) AttachObserver(observer Observer) {
	o.observers = append(o.observers, observer)
}

func (o *Order) NotifyPaid() {
	event := PaidEvent{
		OrderID: o.id,
		Items:   o.cart.Items(),
	}
	for _, obs := range o.observers {
		obs.OnOrderPaid(event)
	}
}
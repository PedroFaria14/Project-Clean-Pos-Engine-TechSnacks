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
	restoredTotal   *shared.Money
	createdBy       string
}

type Status string

const (
	StatusOpen            Status = "OPEN"
	StatusAwaitingPayment Status = "AWAITING_PAYMENT"
	StatusCompleted       Status = "COMPLETED"
)

// RestoreOrder restores a snapshot without executing checkout, payment or observers.
func RestoreOrder(id string, status Status, items []CartItem, total shared.Money) (*Order, error) {
	if id == "" {
		return nil, errors.New("order id is required")
	}
	o := NewOrder(id)
	for _, item := range items {
		if err := o.AddItem(item); err != nil {
			return nil, err
		}
	}
	switch status {
	case StatusOpen:
		if total.Cents() != o.cart.Total().Cents() {
			return nil, errors.New("open order total does not match its items")
		}
	case StatusAwaitingPayment:
		if len(items) == 0 {
			return nil, errors.New("persisted checkout cannot be empty")
		}
		o.SetState(&AwaitingPaymentState{})
	case StatusCompleted:
		if len(items) == 0 {
			return nil, errors.New("persisted checkout cannot be empty")
		}
		o.SetState(&CompletedState{})
	default:
		return nil, errors.New("invalid persisted order status")
	}
	o.restoredTotal = &total
	return o, nil
}

func (o *Order) Status() Status {
	switch o.state.(type) {
	case *OpenState:
		return StatusOpen
	case *AwaitingPaymentState:
		return StatusAwaitingPayment
	case *CompletedState:
		return StatusCompleted
	default:
		return ""
	}
}

// RestoreOrderWithCreator restores ownership without executing a state transition.
func RestoreOrderWithCreator(id string, status Status, items []CartItem, total shared.Money, creator string) (*Order, error) {
	o, err := RestoreOrder(id, status, items, total)
	if err != nil {
		return nil, err
	}
	o.createdBy = creator
	return o, nil
}

func (o *Order) CreatorID() string { return o.createdBy }

func (o *Order) AssignCreator(id string) error {
	if id == "" || o.createdBy != "" || o.Status() != StatusOpen {
		return errors.New("order creator cannot be assigned")
	}
	o.createdBy = id
	return nil
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
	o.restoredTotal = nil
	return nil
}

func (o *Order) SetPaymentStrategy(strategy payment.PaymentStrategy) {
	o.paymentStrategy = strategy
	o.restoredTotal = nil
}
func (o *Order) ID() string {
	return o.id
}

func (o *Order) CalculateTotal() shared.Money {
	if o.restoredTotal != nil {
		return *o.restoredTotal
	}
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
	if err := o.state.AddItem(o, item); err != nil {
		return err
	}
	o.restoredTotal = nil
	return nil
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

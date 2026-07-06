package order

import "errors"

type AwaitingPaymentState struct{}

func (s *AwaitingPaymentState) AddItem(order *Order, item CartItem) error {
	return errors.New("cannot add items while awaiting payment")
}

func (s *AwaitingPaymentState) Checkout(order *Order) error {
	return errors.New("order is already at checkout phase")
}

func (s *AwaitingPaymentState) Pay(order *Order) error {
	order.SetState(&CompletedState{})
	order.NotifyPaid()
	return nil
}
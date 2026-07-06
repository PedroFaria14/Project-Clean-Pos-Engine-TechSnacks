package order

import "errors"

type CompletedState struct{}

func (s *CompletedState) AddItem(order *Order, item CartItem) error {
	return errors.New("cannot add items to a completed order")
}

func (s *CompletedState) Checkout(order *Order) error {
	return errors.New("order is already completed")
}

func (s *CompletedState) Pay(order *Order) error {
	return errors.New("order is already paid")
}
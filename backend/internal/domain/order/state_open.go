package order

import "errors"

type OpenState struct{}

func (s *OpenState) AddItem(order *Order, item CartItem) error {
	order.Cart().AddItem(item)
	return nil
}

func (s *OpenState) Checkout(order *Order) error {
	if len(order.Cart().Items()) == 0 {
		return errors.New("cannot checkout an empty order")
	}
	order.SetState(&AwaitingPaymentState{})
	return nil
}

func (s *OpenState) Pay(order *Order) error {
	return errors.New("cannot pay an open order, proceed to checkout first")
}
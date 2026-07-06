package order

type State interface {
	AddItem(order *Order, item CartItem) error
	Checkout(order *Order) error
	Pay(order *Order) error
}
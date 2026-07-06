package order

type PaidEvent struct {
	OrderID string
	Items   []CartItem
}
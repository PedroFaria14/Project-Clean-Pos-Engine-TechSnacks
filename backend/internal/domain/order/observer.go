package order

type Observer interface {
	OnOrderPaid(event PaidEvent)
}
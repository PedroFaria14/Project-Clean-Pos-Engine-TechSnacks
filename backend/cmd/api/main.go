package main

import (
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/domain/payment"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"fmt"
)

func main() {
	price, _ := shared.NewMoney(10000) 
	p1, _ := product.NewProduct("1", "Honda Civic Type R Model", price)

	myOrder := order.NewOrder("order-123")
	item, _ := order.NewCartItem(p1, 1)
	myOrder.AddItem(*item)


	discountAmount, _ := shared.NewMoney(1000)
	discount, _ := payment.NewDiscount(discountAmount)
	myOrder.ApplyDiscount(discount)

	pix := payment.NewPixStrategy(5) 
	myOrder.SetPaymentStrategy(pix)

	myOrder.Checkout()
	myOrder.Pay()

	fmt.Printf("Order Total: %d cents\n", myOrder.CalculateTotal().Cents())
}
package usecase

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"fmt"
	"time"
)

type CheckoutItemInputDTO struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CheckoutInputDTO struct {
	OrderID string                 `json:"order_id"`
	Items   []CheckoutItemInputDTO `json:"items"`
}

type CheckoutOutputDTO struct {
	OrderID    string `json:"order_id"`
	TotalCents int64  `json:"total_cents"`
	Status     string `json:"status"`
}

type CheckoutUseCase struct {
	orderRepo   port.OrderRepository
	productRepo port.ProductRepository
}

func NewCheckoutUseCase(or port.OrderRepository, pr port.ProductRepository) *CheckoutUseCase {
	return &CheckoutUseCase{
		orderRepo:   or,
		productRepo: pr,
	}
}

func (uc *CheckoutUseCase) Execute(input CheckoutInputDTO) (CheckoutOutputDTO, error) {
	if input.OrderID == "" {
		input.OrderID = fmt.Sprintf("ord-%d", time.Now().Unix())
	}

	newOrder := order.NewOrder(input.OrderID)

	for _, item := range input.Items {
		prod, err := uc.productRepo.FindByID(item.ProductID)
		if err != nil {
			return CheckoutOutputDTO{}, err
		}

		cartItem, err := order.NewCartItem(prod, item.Quantity)
		if err != nil {
			return CheckoutOutputDTO{}, err
		}

		err = newOrder.AddItem(*cartItem)
		if err != nil {
			return CheckoutOutputDTO{}, err
		}
	}

	err := newOrder.Checkout()
	if err != nil {
		return CheckoutOutputDTO{}, err
	}

	err = uc.orderRepo.Save(newOrder)
	if err != nil {
		return CheckoutOutputDTO{}, err
	}

	return CheckoutOutputDTO{
		OrderID:    newOrder.ID(),
		TotalCents: newOrder.CalculateTotal().Cents(),
		Status:     "AWAITING_PAYMENT",
	}, nil
}

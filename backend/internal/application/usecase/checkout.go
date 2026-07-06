package usecase

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
)

type CheckoutInputDTO struct {
	OrderID string
	ItemIDs []string
}

type CheckoutOutputDTO struct {
	OrderID    string
	TotalCents int64
	Status     string
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
	newOrder := order.NewOrder(input.OrderID)

	for _, itemID := range input.ItemIDs {
		prod, err := uc.productRepo.FindByID(itemID)
		if err != nil {
			return CheckoutOutputDTO{}, err
		}

		cartItem, err := order.NewCartItem(prod, 1)
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
		OrderID:    input.OrderID,
		TotalCents: newOrder.CalculateTotal().Cents(),
		Status:     "AWAITING_PAYMENT",
	}, nil
}
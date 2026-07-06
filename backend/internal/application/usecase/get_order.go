package usecase

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
)

type GetOrderOutputDTO struct {
	OrderID    string           `json:"order_id"`
	TotalCents int64            `json:"total_cents"`
	Items      []ReceiptItemDTO `json:"items"`
}

type ReceiptItemDTO struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
	SubTotal    int64  `json:"sub_total"`
}

type GetOrderUseCase struct {
	repo port.OrderRepository
}

func NewGetOrderUseCase(r port.OrderRepository) *GetOrderUseCase {
	return &GetOrderUseCase{repo: r}
}

func (uc *GetOrderUseCase) Execute(orderID string) (GetOrderOutputDTO, error) {
	o, err := uc.repo.FindByID(orderID)
	if err != nil {
		return GetOrderOutputDTO{}, err
	}

	var itemsDTO []ReceiptItemDTO
	for _, item := range o.Cart().Items() {
		itemsDTO = append(itemsDTO, ReceiptItemDTO{
			ProductName: item.Product().Name(),
			Quantity:    item.Quantity(),
			SubTotal:    item.SubTotal().Cents(),
		})
	}

	return GetOrderOutputDTO{
		OrderID:    o.ID(),
		TotalCents: o.CalculateTotal().Cents(),
		Items:      itemsDTO,
	}, nil
}
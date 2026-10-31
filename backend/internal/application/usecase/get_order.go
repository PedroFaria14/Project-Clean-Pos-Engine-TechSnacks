package usecase

import (
	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
)

type GetOrderOutputDTO struct {
	OrderID    string           `json:"order_id"`
	TotalCents int64            `json:"total_cents"`
	Status     string           `json:"status"`
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
	return uc.execute(orderID, nil)
}

func (uc *GetOrderUseCase) ExecuteForUser(orderID string, actor identity.Principal) (GetOrderOutputDTO, error) {
	if actor.ID == "" || actor.Role != identity.Admin && actor.Role != identity.Operator {
		return GetOrderOutputDTO{}, ErrUnauthenticated
	}
	return uc.execute(orderID, &actor)
}

func (uc *GetOrderUseCase) execute(orderID string, actor *identity.Principal) (GetOrderOutputDTO, error) {
	o, err := uc.repo.FindByID(orderID)
	if err != nil {
		return GetOrderOutputDTO{}, err
	}
	if actor != nil && actor.Role != identity.Admin && o.CreatorID() != actor.ID {
		return GetOrderOutputDTO{}, port.ErrOrderNotFound
	}

	items := o.Cart().Items()
	itemsDTO := make([]ReceiptItemDTO, 0, len(items))
	for _, item := range items {
		itemsDTO = append(itemsDTO, ReceiptItemDTO{
			ProductName: item.Product().Name(),
			Quantity:    item.Quantity(),
			SubTotal:    item.SubTotal().Cents(),
		})
	}

	return GetOrderOutputDTO{
		OrderID:    o.ID(),
		TotalCents: o.CalculateTotal().Cents(),
		Status:     string(o.Status()),
		Items:      itemsDTO,
	}, nil
}

package payment

import (
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

type PixStrategy struct {
	discountPercentage int
}

func NewPixStrategy(discountPercentage int) *PixStrategy {
	return &PixStrategy{discountPercentage: discountPercentage}
}

func (s *PixStrategy) CalculateFinalTotal(subtotal shared.Money) shared.Money {
	discountAmount := (subtotal.Cents() * int64(s.discountPercentage)) / 100
	result, _ := shared.NewMoney(subtotal.Cents() - discountAmount)
	return result
}

func (s *PixStrategy) Installments() int {
	return 1
}
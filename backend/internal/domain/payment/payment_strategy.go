package payment

import (
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

type PaymentStrategy interface {
	CalculateFinalTotal(subtotal shared.Money) shared.Money
	Installments() int
}
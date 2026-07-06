package payment

import (
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"errors"
)
type CreditCardStrategy struct {
	installments int
	interestRate int
}

func NewCreditCardStrategy(installments, interestRate int) (*CreditCardStrategy, error) {
	if installments < 1 {
		return nil, errors.New("installments must be at least 1")
	}
	return &CreditCardStrategy{
		installments: installments,
		interestRate: interestRate,
	}, nil
}

func (s *CreditCardStrategy) CalculateFinalTotal(subtotal shared.Money) shared.Money {
	if s.installments <= 3 {
		return subtotal
	}
	interestAmount := (subtotal.Cents() * int64(s.interestRate)) / 100
	
	result, _ := shared.NewMoney(subtotal.Cents() + interestAmount)
	return result
}
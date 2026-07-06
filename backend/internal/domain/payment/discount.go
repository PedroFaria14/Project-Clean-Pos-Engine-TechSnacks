package payment

import (
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"errors"
)

type Discount struct {
	amount shared.Money
}

func NewDiscount(amount shared.Money) (*Discount, error) {
	if amount.Cents() < 0 {
		return nil, errors.New("discount cannot be negative")
	}
	return &Discount{amount: amount}, nil
}

func (d *Discount) Apply(subtotal shared.Money) shared.Money {
	if d.amount.Cents() > subtotal.Cents() {
		zero, _ := shared.NewMoney(0)
		return zero
	}
	
	result, _ := shared.NewMoney(subtotal.Cents() - d.amount.Cents())
	return result
}
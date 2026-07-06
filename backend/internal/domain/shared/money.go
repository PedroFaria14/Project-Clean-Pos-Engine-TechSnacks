package shared

import "errors"

type Money struct {
	cents int64
}

func NewMoney(amount int64) (Money, error) {
	if amount < 0 {
		return Money{}, errors.New("amount cannot be negative")
	}
	return Money{cents: amount}, nil
}

func (m Money) Cents() int64 {
	return m.cents
}

func (m Money) Add(other Money) Money {
	return Money{cents: m.cents + other.cents}
}
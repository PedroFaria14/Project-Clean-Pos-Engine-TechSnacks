package shared

import (
	"errors"
	"math"
)

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

// Add returns the sum, rejecting amounts that cannot be represented by int64.
func (m Money) Add(other Money) (Money, error) {
	if other.cents > math.MaxInt64-m.cents {
		return Money{}, errors.New("money addition exceeds int64 limit")
	}
	return Money{cents: m.cents + other.cents}, nil
}

// Multiply returns the amount multiplied by a nonnegative quantity without overflow.
func (m Money) Multiply(quantity int64) (Money, error) {
	if quantity < 0 {
		return Money{}, errors.New("quantity cannot be negative")
	}
	if quantity != 0 && m.cents > math.MaxInt64/quantity {
		return Money{}, errors.New("money multiplication exceeds int64 limit")
	}
	return Money{cents: m.cents * quantity}, nil
}

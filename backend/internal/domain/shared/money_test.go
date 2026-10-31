package shared_test

import (
	"math"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/domain/shared"
)

func money(t *testing.T, cents int64) shared.Money {
	t.Helper()
	value, err := shared.NewMoney(cents)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestNewMoney(t *testing.T) {
	for _, cents := range []int64{0, 650, math.MaxInt64} {
		value, err := shared.NewMoney(cents)
		if err != nil || value.Cents() != cents {
			t.Fatalf("NewMoney(%d) = %v, %v", cents, value, err)
		}
	}
	if _, err := shared.NewMoney(-1); err == nil {
		t.Fatal("negative money must be rejected")
	}
}

func TestMoneyAdd(t *testing.T) {
	tests := []struct {
		name    string
		left    int64
		right   int64
		want    int64
		wantErr bool
	}{
		{name: "zero", left: 0, right: 0, want: 0},
		{name: "regular amounts", left: 650, right: 350, want: 1000},
		{name: "exact limit", left: math.MaxInt64 - 1, right: 1, want: math.MaxInt64},
		{name: "limit plus zero", left: math.MaxInt64, right: 0, want: math.MaxInt64},
		{name: "overflow", left: math.MaxInt64, right: 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := money(t, tt.left).Add(money(t, tt.right))
			if (err != nil) != tt.wantErr || value.Cents() != tt.want {
				t.Fatalf("Add() = %d, %v; want %d, error=%v", value.Cents(), err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestMoneyMultiply(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		quantity int64
		want     int64
		wantErr  bool
	}{
		{name: "regular amount", cents: 650, quantity: 2, want: 1300},
		{name: "zero quantity", cents: math.MaxInt64, quantity: 0, want: 0},
		{name: "zero price", cents: 0, quantity: math.MaxInt64, want: 0},
		{name: "exact limit", cents: math.MaxInt64, quantity: 1, want: math.MaxInt64},
		{name: "near limit", cents: math.MaxInt64 / 2, quantity: 2, want: math.MaxInt64 - 1},
		{name: "overflow", cents: math.MaxInt64, quantity: 2, wantErr: true},
		{name: "positive wraparound", cents: math.MaxInt64, quantity: 3, wantErr: true},
		{name: "negative quantity", cents: 650, quantity: -1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := money(t, tt.cents).Multiply(tt.quantity)
			if (err != nil) != tt.wantErr || value.Cents() != tt.want {
				t.Fatalf("Multiply() = %d, %v; want %d, error=%v", value.Cents(), err, tt.want, tt.wantErr)
			}
		})
	}
}

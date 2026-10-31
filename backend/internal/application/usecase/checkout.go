package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
)

var (
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrCheckoutUnavailable   = errors.New("checkout temporarily unavailable")
)

func ValidIdempotencyKey(key string) bool {
	if len(key) < 1 || len(key) > 128 {
		return false
	}
	for _, c := range key {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

type CheckoutItemInputDTO struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CheckoutInputDTO struct {
	OrderID string                 `json:"order_id"`
	Items   []CheckoutItemInputDTO `json:"items"`
}

type CheckoutOutputDTO struct {
	OrderID    string `json:"order_id"`
	TotalCents int64  `json:"total_cents"`
	Status     string `json:"status"`
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
	return uc.ExecuteWithKey(context.Background(), input, "")
}

func checkoutOutput(record port.CheckoutRecord) CheckoutOutputDTO {
	return CheckoutOutputDTO{OrderID: record.OrderID, TotalCents: record.TotalCents, Status: record.Status}
}

func (uc *CheckoutUseCase) ExecuteWithKey(ctx context.Context, input CheckoutInputDTO, key string) (CheckoutOutputDTO, error) {
	return uc.execute(ctx, input, key, "")
}

func (uc *CheckoutUseCase) ExecuteForUser(ctx context.Context, input CheckoutInputDTO, key string, actor identity.Principal) (CheckoutOutputDTO, error) {
	if actor.ID == "" || actor.Role != identity.Operator && actor.Role != identity.Admin {
		return CheckoutOutputDTO{}, ErrUnauthenticated
	}
	return uc.execute(ctx, input, key, actor.ID)
}

func (uc *CheckoutUseCase) execute(ctx context.Context, input CheckoutInputDTO, key, creator string) (CheckoutOutputDTO, error) {
	var hash string
	if key != "" {
		if !ValidIdempotencyKey(key) {
			return CheckoutOutputDTO{}, ErrInvalidIdempotencyKey
		}
		if creator != "" {
			key = "user:" + creator + ":" + key
		}
		body, err := json.Marshal(input)
		if err != nil {
			return CheckoutOutputDTO{}, err
		}
		digest := sha256.Sum256(body)
		hash = hex.EncodeToString(digest[:])
		record, err := uc.orderRepo.FindCheckout(ctx, key)
		if err == nil {
			if record.RequestHash != hash {
				return CheckoutOutputDTO{}, port.ErrIdempotencyConflict
			}
			return checkoutOutput(record), nil
		}
		if !errors.Is(err, port.ErrCheckoutNotFound) {
			return CheckoutOutputDTO{}, fmt.Errorf("%w: %w", ErrCheckoutUnavailable, err)
		}
	}
	if input.OrderID == "" {
		input.OrderID = "ord-" + rand.Text()
	}

	newOrder := order.NewOrder(input.OrderID)
	if creator != "" {
		if err := newOrder.AssignCreator(creator); err != nil {
			return CheckoutOutputDTO{}, err
		}
	}

	for _, item := range input.Items {
		prod, err := uc.productRepo.FindByID(item.ProductID)
		if err != nil {
			if errors.Is(err, port.ErrProductNotFound) {
				return CheckoutOutputDTO{}, err
			}
			return CheckoutOutputDTO{}, fmt.Errorf("%w: %w", ErrCheckoutUnavailable, err)
		}

		cartItem, err := order.NewCartItem(prod, item.Quantity)
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

	if key != "" {
		record, err := uc.orderRepo.SaveCheckout(ctx, newOrder, port.CheckoutRecord{
			Key: key, RequestHash: hash, OrderID: newOrder.ID(),
			TotalCents: newOrder.CalculateTotal().Cents(), Status: string(newOrder.Status()),
		})
		if err != nil {
			if errors.Is(err, port.ErrIdempotencyConflict) || errors.Is(err, port.ErrOrderConflict) {
				return CheckoutOutputDTO{}, err
			}
			return CheckoutOutputDTO{}, fmt.Errorf("%w: %w", ErrCheckoutUnavailable, err)
		}
		return checkoutOutput(record), nil
	}
	if err = uc.orderRepo.Save(newOrder); err != nil {
		if errors.Is(err, port.ErrOrderConflict) {
			return CheckoutOutputDTO{}, err
		}
		return CheckoutOutputDTO{}, fmt.Errorf("%w: %w", ErrCheckoutUnavailable, err)
	}

	return CheckoutOutputDTO{
		OrderID:    newOrder.ID(),
		TotalCents: newOrder.CalculateTotal().Cents(),
		Status:     string(newOrder.Status()),
	}, nil
}

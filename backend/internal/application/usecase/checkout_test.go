package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/domain/shared"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"

	"github.com/gin-gonic/gin"
)

func authenticatedRouter() *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(api.AuthenticatedSessionKey, usecase.AuthSession{User: identity.Principal{ID: "test-user", Username: "test-user", Role: identity.Admin}})
		c.Next()
	})
	return router
}

type orderRepositoryStub struct {
	mu    sync.Mutex
	saved []*order.Order
	err   error
}

func (r *orderRepositoryStub) Save(o *order.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.saved = append(r.saved, o)
	return nil
}

func (r *orderRepositoryStub) FindByID(string) (*order.Order, error) {
	return nil, port.ErrOrderNotFound
}

func (r *orderRepositoryStub) FindCheckout(context.Context, string) (port.CheckoutRecord, error) {
	if r.err != nil {
		return port.CheckoutRecord{}, r.err
	}
	return port.CheckoutRecord{}, port.ErrCheckoutNotFound
}

func (r *orderRepositoryStub) SaveCheckout(_ context.Context, o *order.Order, record port.CheckoutRecord) (port.CheckoutRecord, error) {
	return record, r.Save(o)
}

type productRepositoryStub struct {
	product *product.Product
	err     error
}

func (r productRepositoryStub) FindByID(id string) (*product.Product, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.product == nil || id != r.product.ID() {
		return nil, port.ErrProductNotFound
	}
	return r.product, nil
}

func (r productRepositoryStub) Save(*product.Product) error { return nil }
func (r productRepositoryStub) Delete(string) error         { return nil }
func (r productRepositoryStub) Update(context.Context, string, *string, *int64) (*product.Product, error) {
	return nil, errors.New("not implemented by checkout stub")
}
func (r productRepositoryStub) FindAll() ([]*product.Product, error) {
	return []*product.Product{r.product}, r.err
}

func checkoutProduct(t *testing.T, cents int64) *product.Product {
	t.Helper()
	price, err := shared.NewMoney(cents)
	if err != nil {
		t.Fatal(err)
	}
	p, err := product.NewProduct("product-1", "Coxinha", price)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheckoutRegistersAwaitingPaymentUsingCatalogPrice(t *testing.T) {
	repo := &orderRepositoryStub{}
	uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650)})
	output, err := uc.Execute(usecase.CheckoutInputDTO{
		OrderID: "client-order-1",
		Items:   []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.OrderID != "client-order-1" || output.TotalCents != 1300 || output.Status != "AWAITING_PAYMENT" {
		t.Fatalf("unexpected checkout response: %+v", output)
	}
	if len(repo.saved) != 1 || repo.saved[0].CalculateTotal().Cents() != output.TotalCents {
		t.Fatal("checkout must persist exactly one order with the returned total")
	}
}

func TestCheckoutGeneratesDistinctIDsForConcurrentOrders(t *testing.T) {
	const count = 256
	repo := &orderRepositoryStub{}
	uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650)})
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.Execute(usecase.CheckoutInputDTO{
				Items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 1}},
			})
			if err != nil {
				t.Errorf("checkout failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if len(repo.saved) != count {
		t.Fatalf("saved %d orders, want %d", len(repo.saved), count)
	}
	ids := make(map[string]bool, count)
	for _, o := range repo.saved {
		if !strings.HasPrefix(o.ID(), "ord-") {
			t.Fatalf("unexpected generated ID: %q", o.ID())
		}
		if ids[o.ID()] {
			t.Fatalf("different checkouts generated duplicate ID %q", o.ID())
		}
		ids[o.ID()] = true
	}
}

func TestCheckoutRejectsInvalidOrdersWithoutSaving(t *testing.T) {
	tests := []struct {
		name  string
		price int64
		items []usecase.CheckoutItemInputDTO
	}{
		{name: "empty cart", price: 650},
		{name: "missing product", price: 650, items: []usecase.CheckoutItemInputDTO{{ProductID: "missing", Quantity: 1}}},
		{name: "zero quantity", price: 650, items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 0}}},
		{name: "negative quantity", price: 650, items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: -1}}},
		{name: "subtotal overflow", price: math.MaxInt64, items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 2}}},
		{name: "cart overflow", price: math.MaxInt64, items: []usecase.CheckoutItemInputDTO{
			{ProductID: "product-1", Quantity: 1},
			{ProductID: "product-1", Quantity: 1},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &orderRepositoryStub{}
			uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, tt.price)})
			output, err := uc.Execute(usecase.CheckoutInputDTO{Items: tt.items})
			if err == nil {
				t.Fatalf("invalid checkout succeeded: %+v", output)
			}
			if len(repo.saved) != 0 {
				t.Fatal("invalid orders must not be saved")
			}
			if output != (usecase.CheckoutOutputDTO{}) {
				t.Fatalf("failed checkout returned success data: %+v", output)
			}
		})
	}
}

func TestCheckoutPropagatesRepositoryErrors(t *testing.T) {
	dependencyErr := errors.New("dependency unavailable")
	tests := []struct {
		name       string
		orderErr   error
		productErr error
	}{
		{name: "product repository", productErr: dependencyErr},
		{name: "order repository", orderErr: dependencyErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &orderRepositoryStub{err: tt.orderErr}
			uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650), err: tt.productErr})
			_, err := uc.Execute(usecase.CheckoutInputDTO{
				Items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 1}},
			})
			if !errors.Is(err, dependencyErr) {
				t.Fatalf("got %v, want dependency error", err)
			}
			if len(repo.saved) != 0 {
				t.Fatal("repository failure must not record a successful save")
			}
		})
	}
}

func TestCheckoutPreservesHTTPContract(t *testing.T) {
	tests := []struct {
		name   string
		price  int64
		body   string
		status int
	}{
		{name: "success", price: 650, body: `{"items":[{"product_id":"product-1","quantity":2}]}`, status: http.StatusOK},
		{name: "invalid json", price: 650, body: `{"items":`, status: http.StatusBadRequest},
		{name: "incompatible quantity type", price: 650, body: `{"items":[{"product_id":"product-1","quantity":"two"}]}`, status: http.StatusBadRequest},
		{name: "empty cart", price: 650, body: `{"items":[]}`, status: http.StatusUnprocessableEntity},
		{name: "missing product", price: 650, body: `{"items":[{"product_id":"missing","quantity":1}]}`, status: http.StatusUnprocessableEntity},
		{name: "zero quantity", price: 650, body: `{"items":[{"product_id":"product-1","quantity":0}]}`, status: http.StatusUnprocessableEntity},
		{name: "subtotal overflow", price: math.MaxInt64, body: `{"items":[{"product_id":"product-1","quantity":2}]}`, status: http.StatusUnprocessableEntity},
		{name: "cart overflow", price: math.MaxInt64, body: `{"items":[{"product_id":"product-1","quantity":1},{"product_id":"product-1","quantity":1}]}`, status: http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &orderRepositoryStub{}
			uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, tt.price)})
			router := authenticatedRouter()
			router.POST("/checkout", api.NewCheckoutHandler(uc).Handle)
			request := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, tt.status, response.Body.String())
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tt.status == http.StatusOK {
				var output usecase.CheckoutOutputDTO
				if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
					t.Fatal(err)
				}
				if len(body) != 3 || output.OrderID == "" || output.TotalCents != 1300 || output.Status != "AWAITING_PAYMENT" {
					t.Fatalf("success contract changed: %s", response.Body.String())
				}
				return
			}
			var message string
			if err := json.Unmarshal(body["error"], &message); err != nil || message == "" {
				t.Fatalf("error contract changed: %s", response.Body.String())
			}
			if tt.status == http.StatusBadRequest {
				if len(body) != 2 || body["details"] == nil {
					t.Fatal("400 must include field details")
				}
			} else if len(body) != 1 {
				t.Fatal("unexpected legacy domain error fields")
			}
			if len(repo.saved) != 0 {
				t.Fatal("failed HTTP checkout persisted an order")
			}
		})
	}
}

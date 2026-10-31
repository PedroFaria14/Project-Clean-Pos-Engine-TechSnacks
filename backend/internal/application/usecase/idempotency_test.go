package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/domain/order"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
	"PedroFaria14/clean-pos-engine/internal/infra/repository"
)

func TestIdempotentCheckoutReplaysWithoutCatalogAccess(t *testing.T) {
	repo := repository.NewMemoryOrderRepository()
	input := usecase.CheckoutInputDTO{Items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 2}}}
	firstUC := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650)})
	first, err := firstUC.ExecuteWithKey(context.Background(), input, "sale-1")
	if err != nil {
		t.Fatal(err)
	}
	// Even deletion or unavailability of the catalog cannot change a committed result.
	retryUC := usecase.NewCheckoutUseCase(repo, productRepositoryStub{err: errors.New("catalog unavailable")})
	replay, err := retryUC.ExecuteWithKey(context.Background(), input, "sale-1")
	if err != nil || replay != first {
		t.Fatalf("replay = %+v, %v; want %+v", replay, err, first)
	}
	input.Items[0].Quantity = 3
	if _, err := retryUC.ExecuteWithKey(context.Background(), input, "sale-1"); !errors.Is(err, port.ErrIdempotencyConflict) {
		t.Fatalf("changed content must conflict: %v", err)
	}
	o, err := repo.FindByID(first.OrderID)
	if err != nil || string(o.Status()) != first.Status || o.CalculateTotal().Cents() != 1300 {
		t.Fatal("persisted checkout snapshot is incorrect")
	}
	receipt, err := usecase.NewGetOrderUseCase(repo).Execute(first.OrderID)
	if err != nil || receipt.Status != first.Status || receipt.TotalCents != 1300 {
		t.Fatal("receipt must expose the persisted state")
	}
}

func TestConcurrentIdempotentCheckoutsHaveOneResult(t *testing.T) {
	repo := repository.NewMemoryOrderRepository()
	uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650)})
	input := usecase.CheckoutInputDTO{Items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 2}}}
	results := make(chan usecase.CheckoutOutputDTO, 64)
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			output, err := uc.ExecuteWithKey(context.Background(), input, "concurrent-sale")
			if err != nil {
				t.Errorf("checkout failed: %v", err)
				return
			}
			results <- output
		}()
	}
	wg.Wait()
	close(results)
	var first usecase.CheckoutOutputDTO
	count := 0
	for result := range results {
		if count == 0 {
			first = result
		}
		if result != first {
			t.Fatal("same key produced different results")
		}
		count++
	}
	if count != 64 {
		t.Fatalf("got %d results, want 64", count)
	}
}

func TestCheckoutKeysAreIndependentAndInvalidInputDoesNotReserveKey(t *testing.T) {
	repo := repository.NewMemoryOrderRepository()
	uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650)})
	input := usecase.CheckoutInputDTO{Items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 0}}}
	if _, err := uc.ExecuteWithKey(context.Background(), input, "key-1"); err == nil {
		t.Fatal("zero quantity accepted")
	}
	if _, err := repo.FindCheckout(context.Background(), "key-1"); !errors.Is(err, port.ErrCheckoutNotFound) {
		t.Fatal("failed checkout reserved a key")
	}
	input.Items[0].Quantity = 1
	first, err := uc.ExecuteWithKey(context.Background(), input, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := uc.ExecuteWithKey(context.Background(), input, "key-2")
	if err != nil || first.OrderID == second.OrderID {
		t.Fatal("different sales were deduplicated")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.ExecuteWithKey(ctx, input, "cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was lost: %v", err)
	}
	if _, err := repo.FindCheckout(context.Background(), "cancelled"); !errors.Is(err, port.ErrCheckoutNotFound) {
		t.Fatal("cancelled request persisted a record")
	}
}

func TestIdempotencyHTTPContract(t *testing.T) {
	repo := repository.NewMemoryOrderRepository()
	uc := usecase.NewCheckoutUseCase(repo, productRepositoryStub{product: checkoutProduct(t, 650)})
	router := authenticatedRouter()
	router.POST("/checkout", api.NewCheckoutHandler(uc).Handle)
	request := func(keys []string, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		for _, key := range keys {
			r.Header.Add("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, r)
		return response
	}
	body := `{"items":[{"product_id":"product-1","quantity":2}]}`
	first := request([]string{"http-key"}, body)
	second := request([]string{"http-key"}, ` { "items": [ { "quantity": 2, "product_id": "product-1" } ] } `)
	if first.Code != 200 || second.Code != 200 || first.Body.String() != second.Body.String() {
		t.Fatal("equivalent retries must preserve the success response")
	}
	if got := request([]string{"http-key"}, strings.Replace(body, ":2", ":3", 1)); got.Code != 409 {
		t.Fatalf("changed content status: %d", got.Code)
	}
	for _, keys := range [][]string{{""}, {"contains spaces"}, {"x", "y"}, {strings.Repeat("a", 129)}, {"chave-á"}} {
		response := request(keys, body)
		if response.Code != 400 {
			t.Fatalf("invalid keys %v returned %d", keys, response.Code)
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || len(result) != 2 || result["error"] == nil || result["details"] == nil {
			t.Fatal("error envelope changed")
		}
	}
}

func TestIdempotentStorageErrorsAreUnavailableAndNotExposed(t *testing.T) {
	uc := usecase.NewCheckoutUseCase(&orderRepositoryStub{err: errors.New("sensitive database details")}, productRepositoryStub{})
	router := authenticatedRouter()
	router.POST("/checkout", api.NewCheckoutHandler(uc).Handle)
	r := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(`{"items":[]}`))
	r.Header.Set("Idempotency-Key", "unavailable")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 503 || strings.Contains(w.Body.String(), "sensitive") {
		t.Fatalf("unsafe storage failure response: %d %s", w.Code, w.Body.String())
	}
}

type failingOrderRead struct{ port.OrderRepository }

func (failingOrderRead) FindByID(string) (*order.Order, error) {
	return nil, errors.New("sensitive database details")
}

type failingCheckoutCommit struct{ port.OrderRepository }

func (failingCheckoutCommit) SaveCheckout(context.Context, *order.Order, port.CheckoutRecord) (port.CheckoutRecord, error) {
	return port.CheckoutRecord{}, errors.New("commit failed")
}

func TestFailedIdempotentCommitCanBeRetried(t *testing.T) {
	repo := repository.NewMemoryOrderRepository()
	input := usecase.CheckoutInputDTO{Items: []usecase.CheckoutItemInputDTO{{ProductID: "product-1", Quantity: 1}}}
	products := productRepositoryStub{product: checkoutProduct(t, 650)}
	failing := usecase.NewCheckoutUseCase(failingCheckoutCommit{OrderRepository: repo}, products)
	if _, err := failing.ExecuteWithKey(context.Background(), input, "retryable"); !errors.Is(err, usecase.ErrCheckoutUnavailable) {
		t.Fatalf("commit error not classified as unavailable: %v", err)
	}
	if _, err := repo.FindCheckout(context.Background(), "retryable"); !errors.Is(err, port.ErrCheckoutNotFound) {
		t.Fatal("failed commit reserved a key")
	}
	if _, err := usecase.NewCheckoutUseCase(repo, products).ExecuteWithKey(context.Background(), input, "retryable"); err != nil {
		t.Fatal(err)
	}
}

func TestOrderReadHTTPContract(t *testing.T) {
	repo := repository.NewMemoryOrderRepository()
	if err := repo.Save(order.NewOrder("open-empty")); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		repo   port.OrderRepository
		id     string
		status int
	}{
		{name: "empty open order", repo: repo, id: "open-empty", status: 200},
		{name: "not found", repo: repo, id: "missing", status: 404},
		{name: "storage failure", repo: failingOrderRead{}, id: "any", status: 503},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := authenticatedRouter()
			router.GET("/orders/:id", api.NewOrderHandler(usecase.NewGetOrderUseCase(tt.repo)).GetByID)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+tt.id, nil))
			if w.Code != tt.status || strings.Contains(w.Body.String(), "sensitive") {
				t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
			}
			if tt.status == 200 {
				var result map[string]json.RawMessage
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if string(result["items"]) != "[]" || string(result["status"]) != `"OPEN"` {
					t.Fatalf("invalid receipt: %s", w.Body.String())
				}
			}
		})
	}
}

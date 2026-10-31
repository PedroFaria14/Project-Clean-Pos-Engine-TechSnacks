package usecase_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/port"
	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/domain/product"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
)

type unavailableProducts struct{ port.ProductRepository }

func (unavailableProducts) Save(*product.Product) error {
	return errors.New("sensitive infrastructure details")
}
func (unavailableProducts) FindAll() ([]*product.Product, error) {
	return nil, errors.New("sensitive infrastructure details")
}
func (unavailableProducts) Delete(string) error {
	return errors.New("sensitive infrastructure details")
}

func TestLegacyProductHandlersDoNotExposeDependencyErrors(t *testing.T) {
	r := authenticatedRouter()
	h := api.NewProductHandler(usecase.NewProductUseCase(unavailableProducts{}))
	r.POST("/products", h.Create)
	r.GET("/products", h.List)
	r.DELETE("/products/:id", h.Delete)
	for _, method := range []string{"POST", "GET", "DELETE"} {
		path := "/products"
		if method == "DELETE" {
			path += "/p"
		}
		req := httptest.NewRequest(method, path, strings.NewReader(`{"id":"p","name":"Valid","price":0}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 503 || strings.Contains(w.Body.String(), "sensitive") {
			t.Fatalf("unsafe response: %d %s", w.Code, w.Body.String())
		}
	}
}

//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/usecase"
)

func TestPostgresProductEditingPreservesHistoryAndAuthorization(t *testing.T) {
	f := newAuthFixture(t)
	adminCookie, admin := fixtureLogin(t, f, "admin.one")
	operatorCookie, operator := fixtureLogin(t, f, "operator.one")
	if w := authRequest(f.router, "PATCH", "/products/p", `{"name":"Updated"}`, nil, "", authTestOrigin); w.Code != 401 {
		t.Fatal("anonymous edit accepted")
	}
	if w := authRequest(f.router, "PATCH", "/products/p", `{"name":"Updated"}`, operatorCookie, operator.CSRFToken, authTestOrigin); w.Code != 403 {
		t.Fatal("operator edited catalog")
	}
	if w := authRequest(f.router, "PATCH", "/products/p", `{"name":"Updated"}`, adminCookie, "", authTestOrigin); w.Code != 403 {
		t.Fatal("edit accepted without CSRF")
	}
	saleBody := `{"items":[{"product_id":"p","quantity":2}]}`
	first := authRequest(f.router, "POST", "/checkout", saleBody, adminCookie, admin.CSRFToken, authTestOrigin)
	if first.Code != 200 {
		t.Fatal(first.Body.String())
	}
	for _, body := range []string{`{}`, `null`, `{"name":null}`, `{"price":null}`, `{"name":"   "}`, `{"price":-1}`, `{"price":1.5}`, `{"price":9007199254740992}`, `{"id":"another"}`, `{"name":"x"} {}`, `{"name":"` + strings.Repeat("x", 121) + `"}`} {
		w := authRequest(f.router, "PATCH", "/products/p", body, adminCookie, admin.CSRFToken, authTestOrigin)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "details") {
			t.Fatalf("invalid patch response for %s: %d %s", body, w.Code, w.Body.String())
		}
	}
	if w := authRequest(f.router, "PATCH", "/products/missing", `{"name":"Valid"}`, adminCookie, admin.CSRFToken, authTestOrigin); w.Code != 404 {
		t.Fatal("missing product did not return 404")
	}
	w := authRequest(f.router, "PATCH", "/products/p", `{"name":"  Coxinha artesanal  ","price":750}`, adminCookie, admin.CSRFToken, authTestOrigin)
	if w.Code != 200 {
		t.Fatalf("edit failed: %d %s", w.Code, w.Body.String())
	}
	var updated usecase.ProductOutputDTO
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != "p" || updated.Name != "Coxinha artesanal" || updated.Price != 750 {
		t.Fatalf("incorrect edit: %+v", updated)
	}
	replay := authRequest(f.router, "POST", "/checkout", saleBody, adminCookie, admin.CSRFToken, authTestOrigin)
	if replay.Body.String() != first.Body.String() {
		t.Fatal("catalog edit changed the idempotent result")
	}
	var oldSale usecase.CheckoutOutputDTO
	if err := json.Unmarshal(first.Body.Bytes(), &oldSale); err != nil {
		t.Fatal(err)
	}
	receipt := authRequest(f.router, "GET", "/orders/"+oldSale.OrderID, "", adminCookie, "", "")
	var history usecase.GetOrderOutputDTO
	if err := json.Unmarshal(receipt.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if history.TotalCents != 1300 || history.Items[0].ProductName != "Coxinha" {
		t.Fatal("catalog edit changed historical order values")
	}
	fresh := authRequest(f.router, "POST", "/checkout", saleBody, adminCookie, admin.CSRFToken, authTestOrigin, "after-price-edit")
	var newSale usecase.CheckoutOutputDTO
	if err := json.Unmarshal(fresh.Body.Bytes(), &newSale); err != nil {
		t.Fatal(err)
	}
	if fresh.Code != 200 || newSale.TotalCents != 1500 {
		t.Fatal("new sale did not use current catalog price")
	}
	if w := authRequest(f.router, "PATCH", "/products/p", `{"price":0}`, adminCookie, admin.CSRFToken, authTestOrigin); w.Code != 200 {
		t.Fatal("zero price was not updated")
	}
	// Disjoint partial updates must not overwrite a field read before another update.
	repo := NewGormProductRepository(f.db)
	name, price := "Concurrent name", int64(900)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if _, err := repo.Update(context.Background(), "p", &name, nil); err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		if _, err := repo.Update(context.Background(), "p", nil, &price); err != nil {
			t.Error(err)
		}
	}()
	wg.Wait()
	current, err := repo.FindByID("p")
	if err != nil || current.Name() != name || current.Price().Cents() != price {
		t.Fatal("partial updates lost a concurrent field change")
	}
}

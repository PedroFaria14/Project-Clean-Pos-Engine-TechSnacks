//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/application/usecase"
	"PedroFaria14/clean-pos-engine/internal/domain/identity"
	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
	"PedroFaria14/clean-pos-engine/internal/infra/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const authTestPassword = "test-only account password 2026"
const authTestOrigin = "http://localhost:5173"

type authFixture struct {
	db      *gorm.DB
	router  *gin.Engine
	repo    *GormAuthRepository
	service *usecase.AuthService
}

func newAuthFixture(t *testing.T) authFixture {
	t.Helper()
	db := migratedPostgres(t)
	repo := NewGormAuthRepository(db)
	hasher := security.Argon2id{}
	for _, account := range []struct {
		id, name string
		role     identity.Role
	}{{"operator-one", "operator.one", identity.Operator}, {"operator-two", "operator.two", identity.Operator}, {"admin-one", "admin.one", identity.Admin}} {
		hash, err := hasher.Hash(authTestPassword)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateUser(context.Background(), identity.User{Principal: identity.Principal{ID: account.id, Username: account.name, Role: account.role}, PasswordHash: hash, Active: true}); err != nil {
			t.Fatal(err)
		}
	}
	svc, err := usecase.NewAuthService(repo, hasher)
	if err != nil {
		t.Fatal(err)
	}
	products := NewGormProductRepository(db)
	items := testOrder(t, "source").Cart().Items()
	if err := products.Save(items[0].Product()); err != nil {
		t.Fatal(err)
	}
	orders := NewGormOrderRepository(db)
	router, err := api.NewRouter(api.NewAuthHandler(svc, api.AuthConfig{AllowedOrigins: []string{authTestOrigin}}),
		api.NewCheckoutHandler(usecase.NewCheckoutUseCase(orders, products)), api.NewProductHandler(usecase.NewProductUseCase(products)), api.NewOrderHandler(usecase.NewGetOrderUseCase(orders)))
	if err != nil {
		t.Fatal(err)
	}
	return authFixture{db: db, router: router, repo: repo, service: svc}
}

func authRequest(router *gin.Engine, method, path, body string, cookie *http.Cookie, csrf, origin string, keys ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	r.Header.Set("Idempotency-Key", "test-sale-key")
	if len(keys) != 0 {
		r.Header.Set("Idempotency-Key", keys[0])
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func fixtureLogin(t *testing.T, fixture authFixture, username string) (*http.Cookie, usecase.AuthSession) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": authTestPassword, "role": "admin"})
	w := authRequest(fixture.router, "POST", "/sessions", string(body), nil, "", authTestOrigin)
	if w.Code != 201 {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var session usecase.AuthSession
	if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Path != "/" || cookies[0].MaxAge != 28800 || len(cookies[0].Value) != 43 {
		t.Fatal("unsafe session cookie")
	}
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), cookies[0].Value) {
		t.Fatal("credentials or raw token exposed")
	}
	return cookies[0], session
}

func TestPostgresAuthenticationAuthorizationAndCSRF(t *testing.T) {
	f := newAuthFixture(t)
	if w := authRequest(f.router, "GET", "/products", "", nil, "", ""); w.Code != 401 {
		t.Fatal("anonymous business access accepted")
	}
	if w := authRequest(f.router, "POST", "/sessions", `{"username":"operator.one","password":"`+authTestPassword+`"}`, nil, "", "https://evil.example"); w.Code != 403 {
		t.Fatal("cross-site login accepted")
	}
	if w := authRequest(f.router, "POST", "/sessions", `{`, nil, "", authTestOrigin); w.Code != 400 || !strings.Contains(w.Body.String(), "details") {
		t.Fatal("validation did not return field details")
	}
	operatorCookie, operator := fixtureLogin(t, f, "operator.one")
	if operator.User.Role != identity.Operator {
		t.Fatal("client-supplied role escalated privileges")
	}
	if w := authRequest(f.router, "GET", "/products", "", operatorCookie, "", ""); w.Code != 200 {
		t.Fatal("authenticated catalog denied")
	}
	body := `{"items":[{"product_id":"p","quantity":2}]}`
	for _, csrf := range []string{"", "invalid"} {
		if w := authRequest(f.router, "POST", "/checkout", body, operatorCookie, csrf, authTestOrigin); w.Code != 403 {
			t.Fatal("checkout accepted invalid CSRF")
		}
	}
	if w := authRequest(f.router, "POST", "/checkout", body, operatorCookie, operator.CSRFToken, "https://evil.example"); w.Code != 403 {
		t.Fatal("cross-origin sale accepted")
	}
	if w := authRequest(f.router, "DELETE", "/products/p", "", operatorCookie, operator.CSRFToken, authTestOrigin); w.Code != 403 {
		t.Fatal("operator deleted catalog item")
	}
	first := authRequest(f.router, "POST", "/checkout", body, operatorCookie, operator.CSRFToken, authTestOrigin)
	if first.Code != 200 {
		t.Fatalf("checkout: %d %s", first.Code, first.Body.String())
	}
	replay := authRequest(f.router, "POST", "/checkout", body, operatorCookie, operator.CSRFToken, authTestOrigin)
	if replay.Body.String() != first.Body.String() {
		t.Fatal("authenticated checkout did not replay")
	}
	var sale usecase.CheckoutOutputDTO
	if err := json.Unmarshal(first.Body.Bytes(), &sale); err != nil {
		t.Fatal(err)
	}
	if w := authRequest(f.router, "GET", "/orders/"+sale.OrderID, "", operatorCookie, "", ""); w.Code != 200 {
		t.Fatal("operator could not read its own persisted order")
	}
	legacy := testOrder(t, "legacy-unowned-order")
	if err := NewGormOrderRepository(f.db).Save(legacy); err != nil {
		t.Fatal(err)
	}
	if w := authRequest(f.router, "GET", "/orders/legacy-unowned-order", "", operatorCookie, "", ""); w.Code != 404 {
		t.Fatal("operator accessed an unowned legacy order")
	}
	otherCookie, other := fixtureLogin(t, f, "operator.two")
	if w := authRequest(f.router, "GET", "/orders/"+sale.OrderID, "", otherCookie, "", ""); w.Code != 404 {
		t.Fatal("operator read another account's order")
	}
	second := authRequest(f.router, "POST", "/checkout", body, otherCookie, other.CSRFToken, authTestOrigin)
	if second.Code != 200 || second.Body.String() == first.Body.String() {
		t.Fatal("idempotency was not isolated by account")
	}
	adminCookie, admin := fixtureLogin(t, f, "admin.one")
	if w := authRequest(f.router, "GET", "/orders/legacy-unowned-order", "", adminCookie, "", ""); w.Code != 200 {
		t.Fatal("admin could not read an unowned legacy order")
	}
	if w := authRequest(f.router, "GET", "/orders/"+sale.OrderID, "", adminCookie, "", ""); w.Code != 200 {
		t.Fatal("administrator could not inspect order")
	}
	if w := authRequest(f.router, "DELETE", "/products/p", "", adminCookie, admin.CSRFToken, authTestOrigin); w.Code != 204 {
		t.Fatal("administrator could not manage catalog")
	}
	if w := authRequest(f.router, "DELETE", "/sessions/current", "", operatorCookie, operator.CSRFToken, authTestOrigin); w.Code != 204 {
		t.Fatal("logout failed")
	}
	if w := authRequest(f.router, "GET", "/sessions/current", "", operatorCookie, "", ""); w.Code != 401 {
		t.Fatal("revoked cookie still authenticated")
	}
	if w := authRequest(f.router, "GET", "/sessions/current", "", otherCookie, "", ""); w.Code != 200 {
		t.Fatal("logout revoked another session")
	}
	if err := f.db.Exec("UPDATE user_sessions SET expires_at = NOW() - interval '1 second'").Error; err != nil {
		t.Fatal(err)
	}
	if w := authRequest(f.router, "GET", "/sessions/current", "", adminCookie, "", ""); w.Code != 401 {
		t.Fatal("expired session accepted")
	}
}

func TestPostgresAuthenticationLimitsSessionsAndSecretStorage(t *testing.T) {
	f := newAuthFixture(t)
	badKnown := authRequest(f.router, "POST", "/sessions", `{"username":"operator.one","password":"incorrect"}`, nil, "", authTestOrigin)
	badUnknown := authRequest(f.router, "POST", "/sessions", `{"username":"unknown.account","password":"incorrect"}`, nil, "", authTestOrigin)
	if badKnown.Code != 401 || badKnown.Body.String() != badUnknown.Body.String() {
		t.Fatal("login disclosed account existence")
	}
	for range 9 {
		authRequest(f.router, "POST", "/sessions", `{"username":"operator.one","password":"incorrect"}`, nil, "", authTestOrigin)
	}
	limited := authRequest(f.router, "POST", "/sessions", `{"username":"operator.one","password":"incorrect"}`, nil, "", authTestOrigin)
	if limited.Code != 429 || limited.Header().Get("Retry-After") == "" {
		t.Fatal("login brute-force limit did not apply")
	}
	cookie, _ := fixtureLogin(t, f, "operator.two")
	var hashes []string
	if err := f.db.Table("user_sessions").Pluck("token_hash", &hashes).Error; err != nil {
		t.Fatal(err)
	}
	for _, hash := range hashes {
		if hash == cookie.Value || len(hash) != 64 {
			t.Fatal("raw cookie stored in database")
		}
	}
	if err := f.db.Model(&UserModel{}).Where("id = ?", "operator-two").Update("active", false).Error; err != nil {
		t.Fatal(err)
	}
	if w := authRequest(f.router, "GET", "/sessions/current", "", cookie, "", ""); w.Code != 401 {
		t.Fatal("disabled account retained access")
	}
	// Rate limiting is enforced by the database even across repository/service instances.
	repo := NewGormAuthRepository(f.db)
	var wg sync.WaitGroup
	results := make(chan bool, 20)
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			account := sha256.Sum256([]byte("shared-account"))
			address := sha256.Sum256([]byte(fmt.Sprintf("address-%d", i)))
			allowed, err := repo.AllowLogin(context.Background(), hex.EncodeToString(account[:]), hex.EncodeToString(address[:]))
			if err != nil {
				t.Error(err)
				return
			}
			results <- allowed
		}()
	}
	wg.Wait()
	close(results)
	allowed := 0
	for result := range results {
		if result {
			allowed++
		}
	}
	if allowed != 10 {
		t.Fatalf("concurrent limiter allowed %d attempts, want 10", allowed)
	}
	// Production cookie attributes are tested without needing to expose a real server.
	secureHandler := api.NewAuthHandler(f.service, api.AuthConfig{SecureCookies: true})
	router := gin.New()
	router.POST("/sessions", secureHandler.Login)
	w := authRequest(router, "POST", "/sessions", `{"username":"admin.one","password":"`+authTestPassword+`"}`, nil, "", authTestOrigin)
	if w.Code != 201 || !w.Result().Cookies()[0].Secure {
		t.Fatal("production cookie lacks Secure")
	}
}

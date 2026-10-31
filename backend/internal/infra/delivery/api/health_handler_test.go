package api_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
	"github.com/gin-gonic/gin"
)

func TestProbesSeparateProcessHealthFromDatabaseAvailability(t *testing.T) {
	checks := 0
	r := gin.New()
	api.RegisterHealth(r, func(ctx context.Context) error {
		checks++
		if _, ok := ctx.Deadline(); !ok {
			t.Error("readiness has no deadline")
		}
		return errors.New("sensitive database details")
	})
	for _, tt := range []struct {
		path string
		code int
	}{{"/health/live", 200}, {"/health/ready", 503}} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
		if w.Code != tt.code || strings.Contains(w.Body.String(), "sensitive") {
			t.Fatalf("invalid probe response: %d %s", w.Code, w.Body.String())
		}
	}
	if checks != 1 {
		t.Fatal("liveness must not query the database")
	}
}

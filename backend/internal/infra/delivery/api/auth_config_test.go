package api_test

import (
	"testing"

	"PedroFaria14/clean-pos-engine/internal/infra/delivery/api"
)

func TestProductionConfigRequiresExplicitHTTPSOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ALLOWED_ORIGINS", "")
	if _, err := api.AuthConfigFromEnv(); err == nil {
		t.Fatal("production accepted missing origin configuration")
	}
	for _, origin := range []string{"*", "https://*.example.com", "http://example.com", "https://example.com/path", "https://user:pass@example.com", "https://example.com?x=y"} {
		t.Setenv("APP_ALLOWED_ORIGINS", origin)
		if _, err := api.AuthConfigFromEnv(); err == nil {
			t.Fatalf("unsafe origin accepted: %q", origin)
		}
	}
	t.Setenv("APP_ALLOWED_ORIGINS", "https://example.com")
	config, err := api.AuthConfigFromEnv()
	if err != nil || !config.SecureCookies || len(config.AllowedOrigins) != 1 {
		t.Fatal("valid production config was rejected")
	}
}

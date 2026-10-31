package database

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func configEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"DB_PASSWORD", "DB_PASSWORD_FILE", "DB_SSLMODE", "DB_SSLROOTCERT", "DB_SEARCH_PATH", "DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_STATEMENT_TIMEOUT"} {
		t.Setenv(key, "")
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "test-user")
	t.Setenv("DB_NAME", "techsnacks_test_config")
}

func TestDatabaseConfigProtectsSecretsTLSAndPool(t *testing.T) {
	configEnvironment(t)
	password := "fixture-only p@ss/% value"
	t.Setenv("DB_PASSWORD", password)
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(c.dsn())
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := u.User.Password()
	if decoded != password || u.Query().Get("connect_timeout") != "5" {
		t.Fatal("database credentials were not encoded safely")
	}
	t.Setenv("APP_ENV", "production")
	c, err = LoadConfig()
	if err != nil || c.SSLMode != "verify-full" {
		t.Fatal("production did not default to verified TLS")
	}
	t.Setenv("DB_SSLMODE", "disable")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("production accepted unverified database transport")
	}
	t.Setenv("DB_SSLMODE", "verify-full")
	t.Setenv("DB_PASSWORD", "")
	file := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(file, []byte(password+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DB_PASSWORD_FILE", file)
	c, err = LoadConfig()
	if err != nil || c.Password != password {
		t.Fatal("password file was not read safely")
	}
	t.Setenv("DB_PASSWORD", "also-set")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("ambiguous secret configuration accepted")
	}
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "21")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("pool idle limit exceeded the open limit")
	}
}

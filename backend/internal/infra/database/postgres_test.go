package database

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func configEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"DATABASE_URL", "Data_Base", "DB_PASSWORD", "DB_PASSWORD_FILE", "DB_SSLMODE", "DB_SSLROOTCERT", "DB_SEARCH_PATH", "DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_STATEMENT_TIMEOUT"} {
		t.Setenv(key, "")
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "test-user")
	t.Setenv("DB_NAME", "techsnacks_test_config")
}

func urlEnvironment(t *testing.T) {
	t.Helper()
	configEnvironment(t)
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_NAME"} {
		t.Setenv(key, "")
	}
}

func TestNeonURLAliasesPreserveBindingAndVerifiedTLS(t *testing.T) {
	urlEnvironment(t)
	fixture := "postgresql://fixture_user:fixture%40password@db.example.invalid/neondb?sslmode=require&channel_binding=require"
	t.Setenv("Data_Base", fixture)
	t.Setenv("APP_ENV", "production")
	c, err := LoadConfig()
	if err != nil {
		t.Fatal("valid URL was rejected")
	}
	if c.Host != "db.example.invalid" || c.User != "fixture_user" || c.Name != "neondb" || c.Password != "fixture@password" {
		t.Fatal("URL components were not decoded correctly")
	}
	parsed, err := pgx.ParseConfig(c.dsn())
	if err != nil {
		t.Fatal("generated driver configuration is invalid")
	}
	if parsed.ChannelBinding != "require" || parsed.TLSConfig == nil || parsed.TLSConfig.InsecureSkipVerify {
		t.Fatal("channel binding or TLS verification was lost")
	}
	t.Setenv("DATABASE_URL", fixture)
	if _, err := LoadConfig(); err != nil {
		t.Fatal("identical alias values should be compatible")
	}
	t.Setenv("DATABASE_URL", strings.Replace(fixture, "/neondb?", "/different?", 1))
	if _, err := LoadConfig(); err == nil || strings.Contains(err.Error(), "fixture@password") || strings.Contains(err.Error(), "fixture%40password") {
		t.Fatal("conflicting URL aliases were accepted or leaked credentials")
	}
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "also-configured")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("mixed connection identities were silently merged")
	}
}

func TestDatabaseURLValidationDoesNotExposeInput(t *testing.T) {
	urlEnvironment(t)
	for _, fixture := range []string{
		"not-a-url-secret", "https://u:fixture-secret@host/db", "postgresql://u:fixture-secret@host/", "postgresql://u:fixture-secret@host/db?sslmode=disable",
		"postgresql://u:fixture-secret@host/db?sslmode=unknown", "postgresql://u:fixture-secret@host/db?host=other", "postgresql://u:fixture-secret@host/db?channel_binding=require&channel_binding=disable",
	} {
		t.Setenv("APP_ENV", "production")
		t.Setenv("DATABASE_URL", fixture)
		_, err := LoadConfig()
		if err == nil || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatal("invalid connection was accepted or exposed")
		}
	}
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

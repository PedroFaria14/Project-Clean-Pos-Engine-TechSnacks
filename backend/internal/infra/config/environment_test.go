package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/infra/config"
)

func clean(t *testing.T) {
	t.Helper()
	for _, key := range []string{"APP_ENV", "RENDER", "DATABASE_URL", "Data_Base", "DB_HOST", "APP_ALLOWED_ORIGINS"} {
		previous, exists := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(key, previous)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
	t.Chdir(t.TempDir())
}

func TestLocalEnvLoadingPreservesProcessSettings(t *testing.T) {
	clean(t)
	t.Setenv("APP_ALLOWED_ORIGINS", "https://process.example")
	if err := os.WriteFile(".env", []byte("Data_Base=postgresql://fixture:fixture@host/db\nAPP_ENV=development\nAPP_ALLOWED_ORIGINS=https://file.example\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadEnvironment(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("Data_Base") == "" || os.Getenv("APP_ALLOWED_ORIGINS") != "https://process.example" {
		t.Fatal("file was ignored or process settings overwritten")
	}
}

func TestRenderAndExplicitDatabaseSettingsNeverLoadLocalSecrets(t *testing.T) {
	for _, setting := range []string{"RENDER", "DATABASE_URL", "DB_HOST", "APP_ENV"} {
		t.Run(setting, func(t *testing.T) {
			clean(t)
			value := "configured"
			if setting == "APP_ENV" {
				value = "production"
			}
			t.Setenv(setting, value)
			if err := os.WriteFile(".env", []byte("Data_Base=postgresql://fixture:fixture@host/db\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := config.LoadEnvironment(); err != nil || os.Getenv("Data_Base") != "" {
				t.Fatal("runtime unexpectedly loaded a local database")
			}
		})
	}
}

func TestRootDirectoryFallbackAndSanitizedParseErrors(t *testing.T) {
	clean(t)
	if err := os.Mkdir("backend", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("backend", ".env"), []byte("Data_Base=postgresql://fixture:fixture@host/db\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadEnvironment(); err != nil || os.Getenv("Data_Base") == "" {
		t.Fatal("backend/.env was not discovered")
	}
	_ = os.Unsetenv("Data_Base")
	if err := os.WriteFile(".env", []byte("DATABASE_URL='fixture-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	err := config.LoadEnvironment()
	if err == nil || strings.Contains(err.Error(), "fixture-secret") {
		t.Fatal("dotenv error was not sanitized")
	}
}

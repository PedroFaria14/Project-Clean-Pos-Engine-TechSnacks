package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadEnvironment loads only a local development file, never overriding process values.
// Render and explicitly configured database environments never fall back to a local file.
func LoadEnvironment() error {
	if os.Getenv("APP_ENV") == "production" || os.Getenv("RENDER") != "" || os.Getenv("DATABASE_URL") != "" || os.Getenv("Data_Base") != "" || os.Getenv("DB_HOST") != "" {
		return nil
	}
	for _, file := range []string{".env", filepath.Join("backend", ".env")} {
		info, err := os.Stat(file)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("cannot read local environment file")
		}
		values, err := godotenv.Read(file)
		if err != nil {
			return errors.New("invalid local environment file; use KEY=value entries without exposing the connection string")
		}
		if values["DATABASE_URL"] == "" && values["Data_Base"] == "" && values["DB_HOST"] == "" {
			return errors.New("local environment file must define DATABASE_URL, Data_Base or DB_HOST")
		}
		for key, value := range values {
			if _, defined := os.LookupEnv(key); !defined {
				if err := os.Setenv(key, value); err != nil {
					return errors.New("cannot load local environment settings")
				}
			}
		}
		return nil
	}
	return nil
}

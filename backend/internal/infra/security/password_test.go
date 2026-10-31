package security_test

import (
	"strings"
	"testing"

	"PedroFaria14/clean-pos-engine/internal/infra/security"
)

func TestArgon2idSaltVerificationAndBounds(t *testing.T) {
	hasher := security.Argon2id{}
	password := "a test-only password 2026"
	first, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || strings.Contains(first, password) || !strings.HasPrefix(first, "$argon2id$v=19$m=65536,t=3,p=1$") {
		t.Fatal("password hash lacks a unique salt or correct work parameters")
	}
	if match, err := hasher.Verify(first, password); err != nil || !match {
		t.Fatal("correct password rejected")
	}
	if match, err := hasher.Verify(first, "incorrect password"); err != nil || match {
		t.Fatal("incorrect password accepted")
	}
	for _, encoded := range []string{"broken", strings.Replace(first, "m=65536", "m=4294967295", 1), strings.Replace(first, "t=3", "t=0", 1), strings.Replace(first, "p=1", "p=0", 1)} {
		if match, err := hasher.Verify(encoded, password); err == nil || match {
			t.Fatal("invalid or unbounded PHC parameters accepted")
		}
	}
	for _, invalid := range []string{"short", strings.Repeat("x", 129)} {
		if _, err := hasher.Hash(invalid); err == nil {
			t.Fatal("password length limits were ignored")
		}
	}
}

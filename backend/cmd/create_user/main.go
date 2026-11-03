package main

import (
	"context"
	"crypto/rand"
	"flag"
	"log"
	"os"

	"PedroFaria14/clean-pos-engine/internal/domain/identity"
	"PedroFaria14/clean-pos-engine/internal/infra/config"
	"PedroFaria14/clean-pos-engine/internal/infra/database"
	"PedroFaria14/clean-pos-engine/internal/infra/repository"
	"PedroFaria14/clean-pos-engine/internal/infra/security"
)

func main() {
	if err := config.LoadEnvironment(); err != nil {
		log.Fatal(err)
	}
	username := flag.String("username", "", "Account name (3 to 64 ASCII characters)")
	role := flag.String("role", "operator", "operator or admin")
	flag.Parse()
	name, valid := identity.NormalizeUsername(*username)
	if !valid || (*role != "operator" && *role != "admin") {
		log.Fatal("Invalid username or role")
	}
	password := os.Getenv("TECHSNACKS_USER_PASSWORD")
	os.Unsetenv("TECHSNACKS_USER_PASSWORD")
	hash, err := (security.Argon2id{}).Hash(password)
	password = ""
	if err != nil {
		log.Fatal("Password must contain 12 to 128 bytes; provide TECHSNACKS_USER_PASSWORD")
	}
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		log.Fatal("Database pool unavailable")
	}
	defer pool.Close()
	if err := database.CheckOrderSchema(db); err != nil {
		log.Fatal(err)
	}
	user := identity.User{Principal: identity.Principal{ID: rand.Text(), Username: name, Role: identity.Role(*role)}, PasswordHash: hash, Active: true}
	if err := repository.NewGormAuthRepository(db).CreateUser(context.Background(), user); err != nil {
		log.Fatal("Could not create account; verify database availability and account uniqueness")
	}
	log.Print("Account created successfully")
}

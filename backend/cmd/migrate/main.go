package main

import (
	"log"

	"PedroFaria14/clean-pos-engine/internal/infra/database"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		log.Fatal("Database pool unavailable")
	}
	defer pool.Close()
	if err := database.ApplyMigrations(db); err != nil {
		log.Fatal(err)
	}
	log.Print("Database migrations applied successfully")
}

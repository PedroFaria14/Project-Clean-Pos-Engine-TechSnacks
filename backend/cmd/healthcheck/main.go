package main

import (
	"net/http"
	"os"
	"time"

	"PedroFaria14/clean-pos-engine/internal/infra/config"
	"PedroFaria14/clean-pos-engine/internal/infra/server"
)

func main() {
	if err := config.LoadEnvironment(); err != nil {
		os.Exit(1)
	}
	address, err := server.ReadinessURL()
	if err != nil {
		os.Exit(1)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(address)
	if err != nil {
		os.Exit(1)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}

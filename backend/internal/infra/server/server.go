package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

// AddressFromEnv supports platform-provided PORT while preserving HTTP_ADDR overrides.
func AddressFromEnv() (string, error) {
	if address := os.Getenv("HTTP_ADDR"); address != "" {
		host, port, err := net.SplitHostPort(address)
		if err != nil || !validPort(port) {
			return "", errors.New("invalid HTTP_ADDR")
		}
		if host == "" {
			host = "0.0.0.0"
		}
		return net.JoinHostPort(host, port), nil
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if !validPort(port) {
		return "", errors.New("invalid PORT")
	}
	return net.JoinHostPort("0.0.0.0", port), nil
}

func validPort(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	port, err := strconv.Atoi(value)
	return err == nil && port >= 1 && port <= 65535
}

// ReadinessURL uses the same port resolution as the application, including Render PORT.
func ReadinessURL() (string, error) {
	address, err := AddressFromEnv()
	if err != nil {
		return "", err
	}
	host, port, _ := net.SplitHostPort(address)
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	if host == "::" {
		host = "::1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/health/ready", nil
}

// New configures finite HTTP timeouts; TLS terminates at the deployment proxy.
func New(address string, handler http.Handler) *http.Server {
	return &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
}

// Run drains active requests before closing, forcing closure only on deadline.
func Run(ctx context.Context, instance *http.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	result := make(chan error, 1)
	go func() { result <- instance.Serve(listener) }()
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := instance.Shutdown(shutdown); err != nil {
			_ = instance.Close()
			<-result
			return fmt.Errorf("HTTP shutdown did not complete: %w", err)
		}
		if err := <-result; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

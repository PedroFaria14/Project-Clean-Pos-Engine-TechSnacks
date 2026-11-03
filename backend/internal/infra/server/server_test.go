package server_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"PedroFaria14/clean-pos-engine/internal/infra/server"
)

func TestGracefulShutdownDrainsAnActiveRequest(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	instance := server.New(listener.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = w.Write([]byte("completed"))
	}))
	if instance.ReadHeaderTimeout <= 0 || instance.ReadTimeout <= 0 || instance.WriteTimeout <= 0 || instance.IdleTimeout <= 0 {
		t.Fatal("HTTP timeout policy missing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- server.Run(ctx, instance, listener, 2*time.Second) }()
	response := make(chan string, 1)
	go func() {
		r, err := http.Get("http://" + listener.Addr().String())
		if err != nil {
			response <- "failed"
			return
		}
		defer r.Body.Close()
		data, _ := io.ReadAll(r.Body)
		response <- string(data)
	}()
	<-started
	cancel()
	select {
	case err := <-finished:
		t.Fatalf("shutdown abandoned an active request: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not finish")
	}
	if body := <-response; body != "completed" {
		t.Fatal("active response was interrupted")
	}
}

func TestPlatformPortAndExplicitAddress(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("HTTP_ADDR", "")
	if address, err := server.AddressFromEnv(); err != nil || address != "0.0.0.0:8080" {
		t.Fatal("default bind address is incorrect")
	}
	t.Setenv("PORT", "10000")
	if address, err := server.AddressFromEnv(); err != nil || address != "0.0.0.0:10000" {
		t.Fatal("platform PORT was ignored")
	}
	if url, err := server.ReadinessURL(); err != nil || url != "http://127.0.0.1:10000/health/ready" {
		t.Fatal("probe and application ports differ")
	}
	t.Setenv("HTTP_ADDR", "127.0.0.1:8123")
	if address, err := server.AddressFromEnv(); err != nil || address != "127.0.0.1:8123" {
		t.Fatal("HTTP_ADDR override was lost")
	}
	t.Setenv("HTTP_ADDR", "")
	for _, invalid := range []string{"0", "65536", "-1", "10000;bad", "${PORT}"} {
		t.Setenv("PORT", invalid)
		if _, err := server.AddressFromEnv(); err == nil {
			t.Fatalf("invalid port accepted: %q", invalid)
		}
	}
}

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

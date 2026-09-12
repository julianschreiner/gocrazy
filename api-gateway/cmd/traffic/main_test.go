package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gateway/internal/config"
	"gateway/internal/gateway"
)

func TestDashboardTracksGatewayTraffic(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("X-Backend", "users")
		w.Header().Set("X-Backend-Duration-Ms", "12.50")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "backend failure")
	}))
	t.Cleanup(backend.Close)
	handler, err := gateway.New(&config.Config{
		Routes:        []config.RouteConfig{{Name: "users", Methods: []string{"GET"}, PathPrefix: "/users", UpstreamPool: "users"}},
		UpstreamPools: []config.UpstreamPoolConfig{{Name: "users", Targets: []string{backend.URL}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	d := newDashboard()
	observed := d.observe(handler)
	gatewayDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed.ServeHTTP(w, r)
		close(gatewayDone)
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	clientDone := make(chan struct{})
	go func() {
		d.send(ctx, server.Client(), server.URL, 10)
		close(clientDone)
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("request did not reach backend")
	}
	d.Lock()
	active, completed := d.stats["users"].active, d.stats["users"].completed
	d.Unlock()
	if active != 1 || completed != 0 {
		t.Errorf("while backend is blocked: active = %d, completed = %d; want 1, 0", active, completed)
	}
	close(release)
	for _, done := range []chan struct{}{clientDone, gatewayDone} {
		select {
		case <-done:
		case <-ctx.Done():
			t.Fatal("request did not finish")
		}
	}
	d.Lock()
	stats := *d.stats["users"]
	recent := *d.recent[0]
	remaining := len(d.active)
	d.Unlock()
	if stats.active != 0 || stats.completed != 1 || stats.failed != 1 || stats.gatewayCount != 1 || remaining != 0 {
		t.Errorf("finished stats = %+v, active requests = %d", stats, remaining)
	}
	if recent.status != 503 || recent.backend != "users" || recent.backendMS != 12.5 || recent.gatewayMS <= 0 || recent.clientMS <= 0 || recent.failure != "" {
		t.Errorf("recent request = %+v", recent)
	}
	var output bytes.Buffer
	d.render(&output, server.URL, 8, 8, false)
	if strings.Contains(output.String(), "\033") || !strings.Contains(output.String(), "503") || !strings.Contains(output.String(), "users") {
		t.Errorf("plain dashboard output = %q", output.String())
	}
}

func TestDashboardTracksTransportFailure(t *testing.T) {
	d := newDashboard()
	client := &http.Client{Transport: failedTransport{}}
	d.send(context.Background(), client, "http://gateway.invalid", 3)
	stats := d.stats["orders"]
	if stats.active != 0 || stats.completed != 1 || stats.failed != 1 || len(d.active) != 0 {
		t.Fatalf("transport failure stats = %+v", stats)
	}
	if got := d.recent[0]; got.backend != "-" || got.status != 0 || got.failure == "" {
		t.Errorf("transport failure request = %+v", got)
	}
}

type failedTransport struct{}

func (failedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, io.ErrUnexpectedEOF
}

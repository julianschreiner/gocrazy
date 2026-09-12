//go:build integration

package integration_test

import (
	"encoding/json"
	"gateway/internal/config"
	"gateway/internal/gateway"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGatewayIntegration(t *testing.T) {
	cfg, err := config.Load("../../config/integration.yaml")
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	for _, pool := range cfg.UpstreamPools {
		resp, err := client.Get(pool.Targets[0] + "/health")
		if err != nil {
			t.Fatalf("backend %s unavailable: %v; start backends with docker compose up -d --wait", pool.Name, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("backend %s health status = %d, want 200", pool.Name, resp.StatusCode)
		}
	}
	handler, err := gateway.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tests := []struct {
		name, method, uri, body, service, path, query string
		status                                        int
	}{
		{name: "users", method: "GET", uri: "/users", service: "users", path: "/users", status: 200},
		{name: "nested users and query", method: "GET", uri: "/users/42?role=admin&role=reader", service: "users", path: "/users/42", query: "role=admin&role=reader", status: 200},
		{name: "create user", method: "POST", uri: "/users?status=201", body: `{"name":"Ada"}`, service: "users", path: "/users", query: "status=201", status: 201},
		{name: "update user", method: "PUT", uri: "/users/42", body: `{"name":"Grace"}`, service: "users", path: "/users/42", status: 200},
		{name: "delete user", method: "DELETE", uri: "/users/42", service: "users", path: "/users/42", status: 200},
		{name: "orders uses its own pool", method: "GET", uri: "/orders", service: "orders", path: "/orders", status: 200},
		{name: "create order", method: "POST", uri: "/orders", body: `{"user_id":42}`, service: "orders", path: "/orders", status: 200},
		{name: "backend not found", method: "GET", uri: "/users?status=404", service: "users", path: "/users", query: "status=404", status: 404},
		{name: "backend error", method: "GET", uri: "/orders?status=503", service: "orders", path: "/orders", query: "status=503", status: 503},
		{name: "unknown route", method: "GET", uri: "/missing", status: 404},
		{name: "prefix boundary", method: "GET", uri: "/users-other", status: 404},
		{name: "unsupported method", method: "PATCH", uri: "/users", status: 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, server.URL+tt.uri, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-Request-ID", "integration-123")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.status)
			}
			if got := resp.Header.Get("X-Backend"); got != tt.service {
				t.Errorf("X-Backend = %q, want %q", got, tt.service)
			}
			if tt.service == "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				if string(body) != "404 page not found\n" {
					t.Errorf("unmatched route body = %q", body)
				}
				return
			}
			if got := resp.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var got struct {
				Service   string `json:"service"`
				Method    string `json:"method"`
				Path      string `json:"path"`
				Query     string `json:"query"`
				Body      string `json:"body"`
				RequestID string `json:"request_id"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.Service != tt.service || got.Method != tt.method || got.Path != tt.path || got.Query != tt.query || got.Body != tt.body || got.RequestID != "integration-123" {
				t.Errorf("backend received %+v; want service %q, method %q, path %q, query %q, body %q, request ID integration-123", got, tt.service, tt.method, tt.path, tt.query, tt.body)
			}
		})
	}
}

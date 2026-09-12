package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestForward(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		targetPath string
		requestURI string
		wantURI    string
		body       string
		status     int
	}{
		{name: "users path preserved", method: http.MethodGet, requestURI: "/users", wantURI: "/users", status: http.StatusOK},
		{name: "write request preserved", method: http.MethodPost, requestURI: "/users?role=admin&role=reader", wantURI: "/users?role=admin&role=reader", body: `{"name":"Ada"}`, status: http.StatusCreated},
		{name: "target base path and query", method: http.MethodGet, targetPath: "/api?source=gateway", requestURI: "/users?active=true", wantURI: "/api/users?source=gateway&active=true", status: http.StatusOK},
		{name: "escaped path preserved", method: http.MethodGet, requestURI: "/users/a%2Fb", wantURI: "/users/a%2Fb", status: http.StatusOK},
		{name: "backend not found", method: http.MethodGet, requestURI: "/users/missing", wantURI: "/users/missing", status: http.StatusNotFound},
		{name: "backend error", method: http.MethodGet, requestURI: "/users", wantURI: "/users", status: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type receivedRequest struct {
				method, uri, body, header string
				err                       error
			}
			received := make(chan receivedRequest, 1)
			const responseBody = `{"source":"backend"}`
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				received <- receivedRequest{r.Method, r.RequestURI, string(body), r.Header.Get("X-Request-ID"), err}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Backend", "users-service")
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, responseBody)
			}))
			t.Cleanup(backend.Close)
			p, err := New([]Pool{{Name: "users", Targets: []string{backend.URL + tt.targetPath}}})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(tt.method, "http://gateway.example"+tt.requestURI, strings.NewReader(tt.body))
			req.Header.Set("X-Request-ID", "request-123")
			recorder := httptest.NewRecorder()
			p.Forward(recorder, req, "users")

			select {
			case got := <-received:
				if got.err != nil {
					t.Fatal(got.err)
				}
				if got.method != tt.method || got.uri != tt.wantURI || got.body != tt.body || got.header != "request-123" {
					t.Errorf("backend received %+v; want method %q, URI %q, body %q, and request ID request-123", got, tt.method, tt.wantURI, tt.body)
				}
			default:
				t.Fatal("backend did not receive the request")
			}
			if recorder.Code != tt.status {
				t.Errorf("status = %d, want %d", recorder.Code, tt.status)
			}
			if got := recorder.Body.String(); got != responseBody {
				t.Errorf("body = %q, want %q", got, responseBody)
			}
			if got := recorder.Header().Get("X-Backend"); got != "users-service" {
				t.Errorf("X-Backend = %q, want users-service", got)
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
		})
	}
}

func TestForwardUnknownPool(t *testing.T) {
	p, err := New([]Pool{{Name: "users", Targets: []string{"http://backend.invalid"}}})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	p.Forward(recorder, httptest.NewRequest(http.MethodGet, "/users", nil), "missing")
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
	if got := recorder.Body.String(); got != "upstream pool not found\n" {
		t.Errorf("body = %q, want upstream pool not found", got)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("backend connection failed")
}

func TestForwardUpstreamFailure(t *testing.T) {
	p, err := New([]Pool{{Name: "users", Targets: []string{"http://backend.invalid"}}})
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a connection failure without relying on an unused port or DNS.
	p.pools["users"].Transport = failingTransport{}
	recorder := httptest.NewRecorder()
	p.Forward(recorder, httptest.NewRequest(http.MethodGet, "/users", nil), "users")
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

func TestForwardReusesConnectionsAcrossBursts(t *testing.T) {
	const concurrency = 8
	entered := make(chan struct{}, concurrency)
	gates := map[string]chan struct{}{"first": make(chan struct{}), "second": make(chan struct{})}
	var connections atomic.Int64
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		select {
		case <-gates[r.URL.Query().Get("wave")]:
		case <-r.Context().Done():
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	backend.Start()
	t.Cleanup(backend.Close)
	p, err := New([]Pool{{Name: "users", Targets: []string{backend.URL}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, wave := range []string{"first", "second"} {
		responses := make(chan *httptest.ResponseRecorder, concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				request := httptest.NewRequest(http.MethodGet, "/users?wave="+wave, nil).WithContext(ctx)
				response := httptest.NewRecorder()
				p.Forward(response, request, "users")
				responses <- response
			}()
		}
		// Hold every request open to require a whole burst of distinct connections.
		for i := 0; i < concurrency; i++ {
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal("burst did not reach backend")
			}
		}
		close(gates[wave])
		for i := 0; i < concurrency; i++ {
			select {
			case response := <-responses:
				if response.Code != http.StatusOK || response.Body.String() != "ok" {
					t.Errorf("response = %d %q, want 200 ok", response.Code, response.Body.String())
				}
			case <-ctx.Done():
				t.Fatal("burst did not complete")
			}
		}
	}
	if got := connections.Load(); got != concurrency {
		t.Errorf("opened %d upstream connections for two bursts; want %d reused connections", got, concurrency)
	}
}

func TestForwardExpiresIdleUpstreamConnections(t *testing.T) {
	closed := make(chan struct{}, 1)
	var connections atomic.Int64
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	backend.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			connections.Add(1)
		case http.StateClosed:
			select {
			case closed <- struct{}{}:
			default:
			}
		}
	}
	backend.Start()
	t.Cleanup(backend.Close)
	p, err := New([]Pool{{Name: "users", Targets: []string{backend.URL}}})
	if err != nil {
		t.Fatal(err)
	}
	forward := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		response := httptest.NewRecorder()
		p.Forward(response, httptest.NewRequest(http.MethodGet, "/users", nil).WithContext(ctx), "users")
		if response.Code != http.StatusOK || response.Body.String() != "ok" {
			t.Fatalf("response = %d %q, want 200 ok", response.Code, response.Body.String())
		}
	}
	forward()
	// This backend has no idle timeout, so closure must come from the gateway.
	select {
	case <-closed:
	case <-time.After(6 * time.Second):
		t.Fatal("gateway retained the idle upstream connection beyond its expiry window")
	}
	forward()
	if got := connections.Load(); got != 2 {
		t.Errorf("upstream connections = %d, want a fresh connection after idle expiry", got)
	}
}

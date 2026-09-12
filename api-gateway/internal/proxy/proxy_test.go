package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

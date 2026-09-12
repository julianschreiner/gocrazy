package routing_test

import (
	"gateway/internal/routing"
	"net/http"
	"reflect"
	"testing"
)

func TestMatch(t *testing.T) {
	routes := []routing.Route{
		{Name: "users-read", Methods: []string{http.MethodGet}, PathPrefix: "/users", UpstreamPool: "users"},
		{Name: "users-write", Methods: []string{http.MethodPost, http.MethodPut, http.MethodDelete}, PathPrefix: "/users", UpstreamPool: "users"},
		{Name: "users-me", Methods: []string{http.MethodGet}, Path: "/users/me", UpstreamPool: "profile"},
		{Name: "health", Methods: []string{http.MethodGet}, Path: "/health", UpstreamPool: "health"},
		{Name: "no-methods", Path: "/disabled", PathPrefix: "/disabled", UpstreamPool: "disabled"},
		{Name: "nested-prefix", Methods: []string{http.MethodGet}, PathPrefix: "/users/admin", UpstreamPool: "admin"},
		{Name: "duplicate-exact", Methods: []string{http.MethodGet}, Path: "/health", UpstreamPool: "other"},
	}
	router := routing.New(routes)

	tests := []struct {
		name   string
		method string
		path   string
		index  int // -1 means no matching route.
	}{
		{name: "prefix itself", method: http.MethodGet, path: "/users", index: 0},
		{name: "prefix trailing slash", method: http.MethodGet, path: "/users/", index: 0},
		{name: "prefix descendant", method: http.MethodGet, path: "/users/42/orders", index: 0},
		{name: "prefix respects segment boundary", method: http.MethodGet, path: "/users-other", index: -1},
		{name: "exact wins over earlier prefix", method: http.MethodGet, path: "/users/me", index: 2},
		{name: "exact method mismatch falls back to prefix", method: http.MethodPost, path: "/users/me", index: 1},
		{name: "post route", method: http.MethodPost, path: "/users", index: 1},
		{name: "put route", method: http.MethodPut, path: "/users/42", index: 1},
		{name: "delete route", method: http.MethodDelete, path: "/users/42", index: 1},
		{name: "unsupported method", method: http.MethodPatch, path: "/users", index: -1},
		{name: "head requires explicit method", method: http.MethodHead, path: "/users", index: -1},
		{name: "exact path", method: http.MethodGet, path: "/health", index: 3},
		{name: "exact does not match descendant", method: http.MethodGet, path: "/health/details", index: -1},
		{name: "exact does not match trailing slash", method: http.MethodGet, path: "/health/", index: -1},
		{name: "exact method mismatch", method: http.MethodPost, path: "/health", index: -1},
		{name: "empty methods reject exact", method: http.MethodGet, path: "/disabled", index: -1},
		{name: "empty methods reject prefix", method: http.MethodGet, path: "/disabled/child", index: -1},
		{name: "first matching prefix wins", method: http.MethodGet, path: "/users/admin/42", index: 0},
		{name: "unknown path", method: http.MethodGet, path: "/missing", index: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := router.Match(tt.method, tt.path)
			want := routing.Route{}
			if tt.index >= 0 {
				want = routes[tt.index]
			}
			if found != (tt.index >= 0) || !reflect.DeepEqual(got, want) {
				t.Fatalf("Match(%q, %q) = (%+v, %t), want (%+v, %t)", tt.method, tt.path, got, found, want, tt.index >= 0)
			}
		})
	}
}

func TestMatchEmptyRouter(t *testing.T) {
	got, found := routing.New(nil).Match(http.MethodGet, "/users")
	if found || !reflect.DeepEqual(got, routing.Route{}) {
		t.Fatalf("Match on empty router = (%+v, %t), want zero route and false", got, found)
	}
}

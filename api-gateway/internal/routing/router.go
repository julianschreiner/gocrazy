package routing

import (
	"slices"
	"strings"
)

type Route struct {
	Name         string
	Methods      []string
	Path         string
	PathPrefix   string
	UpstreamPool string
}

type Router struct {
	routes []Route
}

func New(routes []Route) *Router {
	return &Router{routes: routes}
}

func (r *Router) Match(method, path string) (Route, bool) {
	for _, route := range r.routes {
		if route.Path == path && matchesMethod(route.Methods, method) {
			return route, true
		}
	}

	for _, route := range r.routes {
		if route.PathPrefix != "" &&
			matchesPrefix(path, route.PathPrefix) &&
			matchesMethod(route.Methods, method) {
			return route, true
		}
	}

	return Route{}, false
}

func matchesMethod(methods []string, method string) bool {
	if len(methods) == 0 {
		return false
	}

	return slices.Contains(methods, method)
}

func matchesPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

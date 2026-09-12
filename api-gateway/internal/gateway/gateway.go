package gateway

import (
	"fmt"
	"net/http"

	"gateway/internal/config"
	"gateway/internal/proxy"
	"gateway/internal/routing"
)

type Gateway struct {
	router *routing.Router
	proxy  *proxy.Proxy
}

func New(cfg *config.Config) (*Gateway, error) {
	router := routing.New(toRoutingRoutes(cfg.Routes))

	p, err := proxy.New(toProxyPools(cfg.UpstreamPools))
	if err != nil {
		return nil, fmt.Errorf("create proxy: %w", err)
	}

	return &Gateway{
		router: router,
		proxy:  p,
	}, nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, found := g.router.Match(r.Method, r.URL.Path)
	if !found {
		http.NotFound(w, r)
		return
	}

	g.proxy.Forward(w, r, route.UpstreamPool)
}

func toRoutingRoutes(configRoutes []config.RouteConfig) []routing.Route {
	routes := make([]routing.Route, 0, len(configRoutes))

	for _, route := range configRoutes {
		routes = append(routes,
			routing.Route{
				Name:         route.Name,
				Methods:      route.Methods,
				Path:         route.Path,
				PathPrefix:   route.PathPrefix,
				UpstreamPool: route.UpstreamPool,
			})
	}

	return routes
}

func toProxyPools(upstreamPoolConfig []config.UpstreamPoolConfig) []proxy.Pool {
	pools := make([]proxy.Pool, 0, len(upstreamPoolConfig))

	for _, upstream := range upstreamPoolConfig {
		pools = append(pools, proxy.Pool{
			Name:    upstream.Name,
			Targets: upstream.Targets,
		})
	}

	return pools
}

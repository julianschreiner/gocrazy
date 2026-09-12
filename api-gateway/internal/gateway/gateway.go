package gateway

import (
	"gateway/internal/config"
	"gateway/internal/routing"
)

type Gateway struct {
	router *routing.Router
}

func New(cfg *config.Config) *Gateway {
	router := routing.New(toRoutingRoutes(cfg.Routes))
	return &Gateway{router: router}
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

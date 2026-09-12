package config

import (
	"errors"
	"fmt"
	"net/url"
)

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if c.Server.Address == "" {
		return errors.New("server address is empty")
	}

	pools := make(map[string]struct{}, len(c.UpstreamPools))
	for _, pool := range c.UpstreamPools {
		if pool.Name == "" {
			return errors.New("upstream pool name is required")
		}
		if len(pool.Targets) == 0 {
			return fmt.Errorf("upstream pool %q has no targets", pool.Name)
		}

		pools[pool.Name] = struct{}{}

		for _, target := range pool.Targets {
			url, err := url.ParseRequestURI(target)
			if err != nil || url.Scheme == "" || url.Host == "" {
				return fmt.Errorf("pool %q has invalid target %q", pool.Name, target)
			}
		}
	}

	for _, route := range c.Routes {
		if route.Name == "" {
			return errors.New("route name is required")
		}
		if route.Path == "" && route.PathPrefix == "" {
			return fmt.Errorf("route %q needs path or path_prefix", route.Name)
		}
		if _, ok := pools[route.UpstreamPool]; !ok {
			return fmt.Errorf(
				"route %q references unknown upstream pool %q",
				route.Name,
				route.UpstreamPool,
			)
		}
	}

	return nil
}

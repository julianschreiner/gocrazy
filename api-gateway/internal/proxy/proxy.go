package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Pool struct {
	Name    string
	Targets []string
}

type Proxy struct {
	// TODO:
	// when we add load balancing, change the map value from one *httputil.ReverseProxy
	// to multiple targets.
	pools map[string]*httputil.ReverseProxy
}

func New(pools []Pool) (*Proxy, error) {
	proxies := make(map[string]*httputil.ReverseProxy, len(pools))

	if len(pools) == 0 {
		return nil, fmt.Errorf("Proxy pool must not be empty")
	}

	for _, pool := range pools {
		if len(pool.Targets) != 1 {
			return nil, fmt.Errorf("Pool %q must have exactly one target", pool.Name)
		}

		target, err := url.Parse(pool.Targets[0])
		if err != nil {
			return nil, fmt.Errorf(
				"parse target for pool %q: %w",
				pool.Name,
				err,
			)
		}

		proxies[pool.Name] = httputil.NewSingleHostReverseProxy(target)
	}

	return &Proxy{
		pools: proxies,
	}, nil
}

func (p *Proxy) Forward(
	w http.ResponseWriter,
	r *http.Request,
	upstreamPoolName string,
) {
	reverseProxy, found := p.pools[upstreamPoolName]
	if !found {
		http.Error(w, "upstream pool not found", http.StatusBadGateway)
	}

	reverseProxy.ServeHTTP(w, r)
}

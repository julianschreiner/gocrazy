package config

/*
*
* TODO add support for:
* auth_policies
* rate_limit_policies
* observability
* e.g:
auth_policies:
  - name: user-api
    type: jwt
    issuer: https://issuer.example.com/
    audience: users-api
    jwks_url: https://issuer.example.com/.well-known/jwks.json
    allowed_algorithms: [RS256]

rate_limit_policies:
  - name: standard
    requests_per_second: 20
    burst: 40
    key: api_key # or ip, subject, tenant

observability:

	log:
	  level: info
	  format: json
	metrics:
	  enabled: true
	  path: /metrics
	tracing:
	  enabled: true
	  service_name: api-gateway
	  endpoint: http://otel-collector:4318/v1/traces
*/
type Config struct {
	Server        ServerConfig         `yaml:"server"`
	Routes        []RouteConfig        `yaml:"routes"`
	UpstreamPools []UpstreamPoolConfig `yaml:"upstream_pools"`
}

/*
TODO add support:
read_timeout: 5s

	write_timeout: 30s
	idle_timeout: 60s
	shutdown_timeout: 10s
	max_header_bytes: 1048576
*/
type ServerConfig struct {
	Address string `yaml:"address"`
}

/*
TODO add support:
strip_prefix: true

	auth:
	  required: true
	  policy: user-api
	rate_limit:
	  policy: standard
*/
type RouteConfig struct {
	Name         string   `yaml:"name"`
	Methods      []string `yaml:"methods"`
	Path         string   `yaml:"path"`
	PathPrefix   string   `yaml:"path_prefix"`
	UpstreamPool string   `yaml:"upstream_pool"`
}

/*
TODO add support:
strategy: round_robin

	health_check:
	  path: /health
	  interval: 10s
	  timeout: 2s
	  unhealthy_threshold: 3
*/
type UpstreamPoolConfig struct {
	Name    string   `yaml:"name"`
	Targets []string `yaml:"targets"`
}

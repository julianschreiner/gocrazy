package config

type Config struct {
	Server        ServerConfig         `yaml:"server"`
	Routes        []RouteConfig        `yaml:"routes"`
	UpstreamPools []UpstreamPoolConfig `yaml:"upstream_pools"`
}

type ServerConfig struct {
	Address string `yaml:"address"`
}

type RouteConfig struct {
	Name         string   `yaml:"name"`
	Methods      []string `yaml:"methods"`
	Path         string   `yaml:"path"`
	PathPrefix   string   `yaml:"path_prefix"`
	UpstreamPool string   `yaml:"upstream_pool"`
}

type UpstreamPoolConfig struct {
	Name    string   `yaml:"name"`
	Targets []string `yaml:"targets"`
}

package main

import (
	"flag"
	"log"

	"gateway/internal/config"
	"gateway/internal/gateway"
)

func main() {
	configPath := flag.String("config", "config/gateway.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	_ = gateway.New(cfg) // creates gateway -> creates router

	// TODO: add proxying, expose the gateway as an http.Handler,
	// then start an HTTP server using cfg.Server.Address.
}

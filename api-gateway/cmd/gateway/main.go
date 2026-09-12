package main

import (
	"flag"
	"log"

	"gateway/internal/config"
)

func main() {
	configPath := flag.String("config", "config/gateway.yaml", "path to config file")
	flag.Parse()

	_, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Create router, proxy, middleware and HTTP server based on cfg
}

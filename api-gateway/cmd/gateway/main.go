package main

import (
	"flag"
	"log"
	"net/http"

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

	gateway, err := gateway.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:    cfg.Server.Address,
		Handler: gateway,
	}

	log.Fatal(server.ListenAndServe())
}

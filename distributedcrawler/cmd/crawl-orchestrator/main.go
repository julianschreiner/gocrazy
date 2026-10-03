package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	coordinator "github.com/julianschreiner/distributedcrawler/internal/coordinator"
	messenger "github.com/julianschreiner/distributedcrawler/internal/messaging"
)

// TODO:
// create stream
// create KV bucket
// create durable consumers (Create one shared durable job consumer; every worker instance binds to it.)
// long running orchestrator
// runtime responsibility:
// tracking crawl progress
// coordinate discovered URLs
// and decide when a crawl is complete

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	messenger, err := messenger.New()
	if err != nil {
		log.Fatal("Could not create nats jetstream " + err.Error())
	}
	defer messenger.NATSConnection.Close()
	if err := messenger.EnsureInfrastructure(ctx); err != nil {
		log.Fatal(err)
	}

	coordinator, err := coordinator.New(messenger)
	if err != nil {
		log.Fatal(err)
	}
	if err := coordinator.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

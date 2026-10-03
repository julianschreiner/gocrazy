package main

import (
	"context"
	"log"
	"net/http"

	"github.com/julianschreiner/distributedcrawler/internal/api"
	messenger "github.com/julianschreiner/distributedcrawler/internal/messaging"
)

func main() {
	messenger, err := messenger.New()
	if err != nil {
		log.Fatal("Could not create nats jetstream " + err.Error())
	}
	defer messenger.NATSConnection.Close()

	err = messenger.EnsureAPIStreamAndBucket(context.Background())
	if err != nil {
		log.Fatal("api dependencies unavailable " + err.Error())
	}

	api := api.API{
		Messenger: messenger,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /crawl", api.Crawl)
	mux.HandleFunc("GET /crawl/{crawlId}", api.CrawlStatus)
	// todo: need these endpoints as well:
	// return discovered pages and broken links (GET)
	// allow active crawl to be cancelled (DELETE)
	log.Fatal(http.ListenAndServe(":8089", mux))
}

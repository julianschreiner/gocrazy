// Package api managing incoming client requests from our crawl-api
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/julianschreiner/distributedcrawler/internal/messaging"
	"github.com/julianschreiner/distributedcrawler/internal/shared"

	"github.com/nats-io/nats.go/jetstream"
)

type API struct {
	Messenger *messaging.Messenger
}

func (a *API) Crawl(w http.ResponseWriter, r *http.Request) {
	var request shared.CrawlRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	// TODO validate incoming request before processing it. Have some blacklist for some kind of URLs
	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Println("received")
	fmt.Fprintln(os.Stdout, request)

	crawlID, status, err := a.handoffIncomingCrawlRequest(r.Context(), request)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		http.Error(w, "could not queue incoming job request "+err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusAccepted)

	response := &shared.CrawlStatus{
		CrawlID: *crawlID,
		Status:  status,
	}

	json.NewEncoder(w).Encode(response)
}

func (a *API) CrawlStatus(w http.ResponseWriter, r *http.Request) {
	crawlID := r.PathValue("crawlId")
	if crawlID == "" {
		http.Error(w, "crawlId is required", http.StatusBadRequest)
		return
	}

	parsedCrawlID, err := uuid.Parse(crawlID)
	if err != nil || parsedCrawlID.Version() != uuid.Version(7) {
		http.Error(w, "given path value is invalid", http.StatusBadRequest)
		return
	}

	fmt.Println("received")
	fmt.Fprintln(os.Stdout, parsedCrawlID)

	entry, err := a.Messenger.CrawlStatuses.Get(r.Context(), parsedCrawlID.String())
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		http.Error(w, "crawl not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "could not retrieve crawl status", http.StatusInternalServerError)
		return
	}

	var response shared.CrawlStatus
	if err := json.Unmarshal(entry.Value(), &response); err != nil {
		http.Error(w, "invalid stored status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func (a *API) handoffIncomingCrawlRequest(ctx context.Context, request shared.CrawlRequest) (*uuid.UUID, shared.JobStatus, error) {
	eventID, err := uuid.NewV7()
	if err != nil {
		return nil, shared.StatusFailed, err
	}

	startRequest := shared.CrawlStartRequest{
		CrawlID: eventID,
		Request: request,
	}
	message, err := json.Marshal(startRequest)
	if err != nil {
		return nil, shared.StatusFailed, fmt.Errorf("marshal crawl start request: %w", err)
	}

	ack, err := a.Messenger.JetStream.Publish(
		ctx,
		messaging.CrawlRequestSubject,
		message,
		jetstream.WithMsgID(eventID.String()),
	)
	if err != nil {
		return nil, shared.StatusFailed, fmt.Errorf("publish crawl start request: %w", err)
	}

	if ack.Duplicate {
		return nil, shared.StatusFailed, fmt.Errorf("crawl start request %s already published", eventID)
	}

	return &eventID, shared.StatusPending, nil
}
